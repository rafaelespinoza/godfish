// Package godfish is a database migration library built to support the command
// line tool.
package godfish

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Direction describes which way the change goes.
type Direction uint8

const (
	// DirUnknown is a fallback value for an invalid direction.
	DirUnknown Direction = iota
	// DirForward is like migrate up, typically the change you want to apply to
	// the DB.
	DirForward
	// DirReverse is like migrate down; used for rollbacks. Not all changes can
	// be rolled back.
	DirReverse
)

func (d Direction) String() string {
	return [...]string{"Unknown", "Forward", "Reverse"}[d]
}

const (
	// TimeFormat provides a consistent timestamp layout for migration
	// filenames. Formatting time in go works a little differently than in other
	// languages. Read more at: https://golang.org/pkg/time/#pkg-constants.
	TimeFormat        = "20060102150405"
	filenameDelimeter = "-"
)

// filename is just a string with a specific format to migration files. One part
// has a generated timestamp, one part has a direction, another has a name.
type filename string

// makeFilename creates a filename based on the independent parts. Format:
// "${direction}-${version}-${name}.sql"
func makeFilename(version string, direction Direction, name string) (filename, error) {
	vLen := len(version)
	if vLen < len(TimeFormat) {
		return "", fmt.Errorf("version must have length %d", len(TimeFormat))
	} else if vLen > len(TimeFormat) {
		version = version[:len(TimeFormat)]
	}
	if match, err := regexp.MatchString(`\d{14}`, version); err != nil {
		return "", fmt.Errorf("developer error %v", err)
	} else if !match {
		return "", fmt.Errorf("version %q does not match pattern", version)
	}

	if direction == DirUnknown {
		return "", fmt.Errorf("cannot have unknown direction")
	}

	dir := strings.ToLower(direction.String()) + filenameDelimeter
	ver := version + filenameDelimeter
	return filename(dir + ver + name + ".sql"), nil
}

func parseMigration(name filename) (mig Migration, err error) {
	var ts time.Time
	var dir Direction
	base := filepath.Base(string(name))

	if strings.HasPrefix(base, strings.ToLower(DirForward.String())) {
		dir = DirForward
	} else if strings.HasPrefix(base, strings.ToLower(DirReverse.String())) {
		dir = DirReverse
	} else {
		err = errInvalidFilename
		return
	}
	// index of the start of timestamp
	i := len(dir.String()) + len(filenameDelimeter)
	timestamp := string(base[i : i+len(TimeFormat)])
	if ts, err = time.Parse(TimeFormat, timestamp); err != nil {
		return
	}
	// index of the start of migration name
	j := i + len(timestamp) + len(filenameDelimeter)

	mig, err = newMutation(
		ts,
		dir,
		strings.TrimSuffix(string(base[j:]), ".sql"),
	)
	return
}

// A Migration is a database change with a direction name and timestamp.
// Typically, a Migration with a DirForward Direction is paired with another
// migration of DirReverse that has the same name.
type Migration interface {
	Direction() Direction
	Name() string
	Timestamp() time.Time
}

// Basename generates a migration file's basename. The output format is:
// "${direction}-${timestamp}-${name}.sql".
func Basename(mig Migration) (string, error) {
	out, err := makeMigrationFilename(mig)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// mutation implements the Migration interface.
type mutation struct {
	direction Direction
	name      string
	timestamp time.Time
}

var _ Migration = (*mutation)(nil)

// newMutation constructs a mutation and returns a pointer. Its internal
// timestamp field is set to UTC.
func newMutation(ts time.Time, dir Direction, name string) (*mutation, error) {
	if dir == DirUnknown {
		return nil, fmt.Errorf("cannot have unknown direction")
	}
	return &mutation{
		direction: dir,
		name:      name,
		timestamp: ts.UTC(),
	}, nil
}

func (m *mutation) Direction() Direction { return m.direction }
func (m *mutation) Name() string         { return m.name }
func (m *mutation) Timestamp() time.Time { return m.timestamp }

// MigrationParams collects inputs needed to generate migration files. Setting
// Reversible to true will generate a migration file for each direction.
// Otherwise, it only generates a file in the forward direction. The Directory
// field refers to the path to the directory with the migration files.
type MigrationParams struct {
	Forward    Migration
	Reverse    Migration
	Reversible bool
	Directory  *os.File
}

// NewMigrationParams constructs a MigrationParams that's ready to use. Passing
// in true for reversible means that a complementary SQL file will be made for
// rolling back. The directory parameter specifies which directory to output the
// files to.
func NewMigrationParams(name string, reversible bool, directory *os.File) (*MigrationParams, error) {
	var out MigrationParams
	var err error
	var info os.FileInfo
	if info, err = directory.Stat(); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("input dir %q should be a directory", info.Name())
	}
	out.Directory = directory

	out.Reversible = reversible
	timestamp := time.Now()
	var mut *mutation
	if mut, err = newMutation(timestamp, DirForward, name); err != nil {
		return nil, err
	}
	out.Forward = mut
	if mut, err = newMutation(timestamp, DirReverse, name); err != nil {
		return nil, err
	}
	out.Reverse = mut
	return &out, nil
}

// GenerateFiles creates the migration files. If the migration is reversible it
// generates files in forward and reverse directions; otherwise is generates
// just one migration file in the forward direction. It closes each file handle
// when it's done.
func (m *MigrationParams) GenerateFiles() (err error) {
	var forwardFile, reverseFile *os.File
	defer func() {
		forwardFile.Close()
		reverseFile.Close()
	}()
	baseDir := m.Directory.Name()
	if forwardFile, err = newMigrationFile(m.Forward, baseDir); err != nil {
		return
	}
	log.Println("created forward file, ", forwardFile.Name())
	if !m.Reversible {
		log.Println("migration marked irreversible, did not create reverse file")
		return
	}
	if reverseFile, err = newMigrationFile(m.Reverse, baseDir); err != nil {
		return
	}
	log.Println("created reverse file, ", reverseFile.Name())
	return
}

func newMigrationFile(m Migration, baseDir string) (*os.File, error) {
	filename, err := makeMigrationFilename(m)
	if err != nil {
		return nil, err
	}
	return os.Create(baseDir + "/" + string(filename))
}

// makeMigrationFilename passes in a Migration's fields to create a filename. An
// error could be returned if m is found to be an unsuitable filename.
func makeMigrationFilename(m Migration) (filename, error) {
	return makeFilename(
		m.Timestamp().Format(TimeFormat),
		m.Direction(),
		m.Name(),
	)
}

// pathToMigrationFile is a convenience function for prepending a directory path
// to the base filename of a migration. An error could be returned if the
// Migration's fields are unsuitable for a filename.
func pathToMigrationFile(dir string, mig Migration) (string, error) {
	filename, err := makeMigrationFilename(mig)
	if err != nil {
		return "", err
	}
	return dir + "/" + string(filename), nil
}

var (
	minVersion = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Format(TimeFormat)
	maxVersion = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC).Format(TimeFormat)
)

// Migrate executes all migrations at directoryPath in the specified direction.
func Migrate(driver Driver, directoryPath string, direction Direction, finishAtVersion string) (err error) {
	var migrations []Migration
	defer driver.Close()

	if finishAtVersion == "" && direction == DirForward {
		finishAtVersion = maxVersion
	} else if finishAtVersion == "" && direction == DirReverse {
		finishAtVersion = minVersion
	}

	if _, err = driver.Connect(); err != nil {
		return
	}
	if migrations, err = listMigrationsToApply(driver, directoryPath, direction, finishAtVersion, false); err != nil {
		return
	}

	for _, mig := range migrations {
		var pathToFile string
		if pathToFile, err = pathToMigrationFile(directoryPath, mig); err != nil {
			return
		}
		if err = runMigration(driver, pathToFile, mig); err != nil {
			return
		}
	}
	return
}

var (
	// ErrNoFilesFound is returned when there are no migration files.
	ErrNoFilesFound = errors.New("no files found")
	// ErrNoVersionFound means no matching migration version is found.
	ErrNoVersionFound = errors.New("no version found")
	// ErrSchemaMigrationsDoesNotExist means there is no database table to
	// record migration status.
	ErrSchemaMigrationsDoesNotExist = errors.New("table \"schema_migrations\" does not exist")
	errInvalidFilename              = errors.New("invalid filename")
)

// ApplyMigration runs a migration at directoryPath with the specified version
// and direction.
func ApplyMigration(driver Driver, directoryPath string, direction Direction, version string) (err error) {
	var mig Migration
	var pathToFile string
	defer driver.Close()

	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}
	if _, err = driver.Connect(); err != nil {
		return
	}
	if version == "" {
		// attempt to find the next version to apply in the direction
		limit := maxVersion
		if direction == DirReverse {
			limit = minVersion
		}
		if toApply, ierr := listMigrationsToApply(driver, directoryPath, direction, limit, false); ierr != nil {
			err = fmt.Errorf("specified no version; error attempting to find one; %v", ierr)
			return
		} else if len(toApply) < 1 {
			err = ErrNoVersionFound
			return
		} else {
			version = toApply[0].Timestamp().Format(TimeFormat)
		}
	}

	if pathToFile, err = figureOutBasename(directoryPath, direction, version); err != nil {
		return
	}
	fn := filename(directoryPath + "/" + pathToFile)
	if mig, err = parseMigration(fn); err != nil {
		return
	}
	err = runMigration(driver, pathToFile, mig)
	return
}

func figureOutBasename(directoryPath string, direction Direction, version string) (f string, e error) {
	var baseGlob filename
	var filenames []string
	if baseGlob, e = makeFilename(version, direction, "*"); e != nil {
		return
	}
	glob := directoryPath + "/" + string(baseGlob)
	if filenames, e = filepath.Glob(glob); e != nil {
		return
	} else if len(filenames) == 0 {
		e = ErrNoFilesFound
		return
	} else if len(filenames) > 1 {
		e = fmt.Errorf("need 1 matching filename; got %v", filenames)
		return
	}
	f = filenames[0]
	return
}

// runMigration executes a migration against the database. The input, pathToFile
// should be relative to the current working directory.
func runMigration(driver Driver, pathToFile string, mig Migration) (err error) {
	var file *os.File
	var info os.FileInfo
	defer file.Close()
	if file, err = os.Open(pathToFile); err != nil {
		return
	}
	if info, err = file.Stat(); err != nil {
		return
	}
	data := make([]byte, int(info.Size()))
	if _, err = file.Read(data); err != nil {
		return
	}
	if err = driver.Execute(string(data)); err != nil {
		return
	}
	if err = driver.CreateSchemaMigrationsTable(); err != nil {
		return
	}
	err = driver.UpdateSchemaMigrations(
		mig.Direction(),
		mig.Timestamp().Format(TimeFormat),
	)
	return
}

// ConnectionParams is what to use when initializing a DSN.
type ConnectionParams struct {
	Encoding string // Encoding is the client encoding for the connection.
	Host     string // Host is the name of the host to connect to.
	Name     string // Name is the database name.
	Pass     string // Pass is the password to use for the connection.
	Port     string // Port is the connection port.
	User     string // User is the name of the user to connect as.
}

// DSN describes a type which generates a data source name for connecting to a
// database. The output will be passed to the standard library's sql.Open method
// to connect to a database.
type DSN interface {
	// Boot takes inputs from the host environment so it can create a Driver.
	Boot(ConnectionParams) error
	// NewDriver calls the constructor of the corresponding Driver.
	NewDriver(*MigrationsConf) (Driver, error)
	// String uses connection parameters to form the data source name.
	String() string
}

// A Driver describes what a database driver (anything at
// https://github.com/golang/go/wiki/SQLDrivers) should be able to do.
type Driver interface {
	// Name should return the name of the driver: ie: postgres, mysql, etc
	Name() string

	// Connect should open a connection (a *sql.DB) to the database and save an
	// internal reference to that connection for later use. This library might
	// call this method multiple times, so use the internal reference if it's
	// present instead of reconnecting to the database.
	Connect() (*sql.DB, error)
	// Close should check if there's an internal reference to a database
	// connection (a *sql.DB) and if it's present, close it. Then reset the
	// internal reference to that connection to nil.
	Close() error
	// DSN returns data source name info, ie: how do I connect?
	DSN() DSN

	// AppliedVersions queries the schema migrations table for migration
	// versions that have been executed against the database. If the schema
	// migrations table does not exist, the returned error should be
	// ErrSchemaMigrationsDoesNotExist.
	AppliedVersions() (AppliedVersions, error)
	// CreateSchemaMigrationsTable should create a table to record migration
	// versions once they've been applied. The version should be a timestamp as
	// a string, formatted as the TimeFormat variable in this package.
	CreateSchemaMigrationsTable() error
	// DumpSchema should output the database structure to stdout.
	DumpSchema() error
	// Execute runs the schema change and commits it to the database. The query
	// parameter is a SQL string and may contain placeholders for the values in
	// args. Input should be passed to conn so it could be sanitized, escaped.
	Execute(query string, args ...interface{}) error
	// UpdateSchemaMigrations records a timestamped version of a migration that
	// has been successfully applied by adding a new row to the schema
	// migrations table.
	UpdateSchemaMigrations(dir Direction, version string) error
}

// NewDriver initializes a Driver implementation by name and connection
// parameters.
func NewDriver(dsn DSN, migConf *MigrationsConf) (driver Driver, err error) {
	return dsn.NewDriver(migConf)
}

// MigrationsConf is intended to lend customizations such as specifying the path
// to the migration files.
type MigrationsConf struct {
	PathToFiles string `json:"path_to_files"`
}

// AppliedVersions represents an iterative list of migrations that have been run
// against the database and have been recorded in the schema migrations table.
// It's enough to convert a *sql.Rows struct when implementing the Driver
// interface since a *sql.Rows already satisfies this interface. See existing
// Driver implementations in this package for examples.
type AppliedVersions interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
}

var _ AppliedVersions = (*sql.Rows)(nil)

// CreateSchemaMigrationsTable creates a table to track status of migrations on
// the database. Running any migration will create the table, so you don't
// usually need to call this function.
func CreateSchemaMigrationsTable(driver Driver) (err error) {
	if _, err = driver.Connect(); err != nil {
		return err
	}
	defer driver.Close()
	return driver.CreateSchemaMigrationsTable()
}

// DumpSchema describes the database structure and outputs to standard out.
func DumpSchema(driver Driver) (err error) {
	if _, err = driver.Connect(); err != nil {
		return err
	}
	defer driver.Close()
	return driver.DumpSchema()
}

// Info displays the outputs of various helper functions.
func Info(driver Driver, directoryPath string, direction Direction, finishAtVersion string) (err error) {
	if _, err = driver.Connect(); err != nil {
		return err
	}
	defer driver.Close()
	_, err = listMigrationsToApply(
		driver,
		directoryPath,
		direction,
		finishAtVersion,
		true,
	)
	return
}

// Init creates a configuration file at pathToFile unless it already exists.
func Init(pathToFile string) (err error) {
	_, err = os.Stat(pathToFile)
	if err == nil {
		fmt.Printf("config file %q already present\n", pathToFile)
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	var data []byte
	if data, err = json.MarshalIndent(MigrationsConf{}, "", "\t"); err != nil {
		return err
	}
	return ioutil.WriteFile(
		pathToFile,
		append(data, byte('\n')),
		os.FileMode(0644),
	)
}

func listMigrationsToApply(driver Driver, directoryPath string, direction Direction, finishAtVersion string, verbose bool) (out []Migration, err error) {
	var available, applied, toApply []Migration
	var finish time.Time
	if available, err = listAvailableMigrations(directoryPath, direction); err != nil {
		return
	}
	if verbose {
		fmt.Println("-- All available migrations")
		printMigrations(available)
		fmt.Println()
	}
	if err = scanAppliedVersions(driver, func(rows AppliedVersions) (e error) {
		var version, basename string
		var mig Migration
		if e = rows.Scan(&version); e != nil {
			return
		}
		if basename, e = figureOutBasename(directoryPath, direction, version); e != nil {
			return
		}
		if mig, e = parseMigration(filename(basename)); e != nil {
			return
		}
		applied = append(applied, mig)
		return
	}); err == ErrSchemaMigrationsDoesNotExist {
		// The next invocation of CreateSchemaMigrationsTable should fix this.
		// We can continue with zero value for now.
		if verbose {
			fmt.Printf("no migrations applied yet; %v\n", err)
		}
	} else if err != nil {
		return
	}
	if verbose {
		fmt.Println("-- Applied migrations")
		printMigrations(applied)
		fmt.Println()
	}
	if toApply, err = selectMigrationsToApply(
		direction,
		applied,
		available,
	); err != nil {
		return
	}
	if finishAtVersion == "" && direction == DirForward {
		finishAtVersion = maxVersion
	} else if finishAtVersion == "" && direction == DirReverse {
		finishAtVersion = minVersion
	}
	if finish, err = time.Parse(TimeFormat, finishAtVersion); err != nil {
		return
	}
	for _, mig := range toApply {
		timestamp := mig.Timestamp()
		if direction == DirForward && timestamp.After(finish) {
			break
		}
		if direction == DirReverse && timestamp.Before(finish) {
			break
		}
		out = append(out, mig)
	}
	if verbose {
		fmt.Println("-- Migrations to apply")
		printMigrations(out)
	}
	return
}

// listAvailableMigrations returns a list of Migration values at directoryPath in
// a specified direction.
func listAvailableMigrations(directoryPath string, direction Direction) (out []Migration, err error) {
	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}
	var fileDir *os.File
	var filenames []string
	if fileDir, err = os.Open(directoryPath); err != nil {
		if _, ok := err.(*os.PathError); ok {
			err = fmt.Errorf("path to migration files %q not found", directoryPath)
		}
		return
	}
	defer fileDir.Close()
	if filenames, err = fileDir.Readdirnames(0); err != nil {
		return
	}
	if direction == DirForward {
		sort.Strings(filenames)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(filenames)))
	}
	for _, fn := range filenames {
		mig, ierr := parseMigration(filename(fn))
		if ierr == errInvalidFilename {
			fmt.Println(ierr)
			continue
		} else if ierr != nil {
			err = ierr
			return
		}
		dir := mig.Direction()
		if dir != direction {
			continue
		}
		out = append(out, mig)
	}
	return
}

func scanAppliedVersions(driver Driver, scan func(AppliedVersions) error) (err error) {
	var rows AppliedVersions
	if rows, err = driver.AppliedVersions(); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		if err = scan(rows); err != nil {
			return
		}
	}
	return
}

func selectMigrationsToApply(direction Direction, applied, available []Migration) (out []Migration, err error) {
	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}

	allVersions := make(map[int64]Migration)
	uniqueToApplied := make(map[int64]Migration)
	for _, mig := range applied {
		version := mig.Timestamp().Unix()
		uniqueToApplied[version] = mig
		allVersions[version] = mig
	}
	uniqueToAvailable := make(map[int64]Migration)
	for _, mig := range available {
		version := mig.Timestamp().Unix()
		if _, ok := uniqueToApplied[version]; ok {
			delete(uniqueToApplied, version)
		} else {
			uniqueToAvailable[version] = mig
			allVersions[version] = mig
		}
	}

	if direction == DirForward {
		for version, mig := range allVersions {
			_, isApplied := uniqueToApplied[version]
			_, isAvailable := uniqueToAvailable[version]
			if !isApplied && isAvailable {
				out = append(out, mig)
			}
		}
	} else {
		for version, mig := range allVersions {
			_, appliedOK := uniqueToApplied[version]
			_, availableOK := uniqueToAvailable[version]
			if !appliedOK && !availableOK {
				out = append(out, mig)
			}
		}
	}
	if direction == DirForward {
		sort.Slice(out, func(i, j int) bool {
			return out[i].Timestamp().Before(out[j].Timestamp())
		})
	} else {
		sort.Slice(out, func(i, j int) bool {
			return out[j].Timestamp().Before(out[i].Timestamp())
		})
	}
	return
}

func printMigrations(migrations []Migration) {
	fmt.Printf("\t%-10s | %-20s | %-s\n", "direction", "version", "name")
	fmt.Printf("\t%-10s | %-20s | %-s\n", "---------", "-------", "----")
	for _, mig := range migrations {
		fmt.Printf(
			"\t%-10s | %-20s | %-s\n",
			mig.Direction(), mig.Timestamp().Format(TimeFormat), mig.Name(),
		)
	}
}
