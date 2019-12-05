package godfish

import (
	"database/sql"
	"fmt"
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
	filenameDelimeter = "."
	// TimeFormat provides a consistent timestamp layout for migration
	// filenames. Formatting time in go works a little differently than in other
	// languages. Read more at: https://golang.org/pkg/time/#pkg-constants.
	TimeFormat = "20060102150405"
)

// DefaultMigrationFileDirectory is the location relative to this file where
// DDL files are stored.
const DefaultMigrationFileDirectory = "../migrations" // TODO: change or remove

// filename is just a string with a specific format to migration files. One part
// has a generated timestamp, one part has a direction, another has a name.
type filename string

// makeFilename creates a filename based on the independent parts. Format:
// "2006010215040506.${direction}.${name}.sql"
func makeFilename(version string, direction Direction, name string) (filename, error) {
	if len(version) != len(TimeFormat) {
		return "", fmt.Errorf("version must have length %d", len(TimeFormat))
	} else if match, err := regexp.MatchString(`\d{14}`, version); err != nil {
		return "", fmt.Errorf("developer error %v", err)
	} else if !match {
		return "", fmt.Errorf("version %q does not match pattern", version)
	}
	if direction == DirUnknown {
		return "", fmt.Errorf("cannot have unknown direction")
	}
	if strings.Contains(name, filenameDelimeter) {
		return "", fmt.Errorf("name %q cannot contain %q", name, filenameDelimeter)
	}
	head := version + filenameDelimeter
	tail := filenameDelimeter + name + ".sql"
	dir := strings.ToLower(direction.String())
	return filename(head + dir + tail), nil
}

func parseMigration(name filename) (mig Migration, err error) {
	var ts time.Time
	var dir Direction
	base := filepath.Base(string(name))
	parts := strings.Split(base, filenameDelimeter)

	ts, err = time.Parse(TimeFormat, parts[0])
	if err != nil {
		return
	}
	if strings.ToLower(parts[1]) == "forward" {
		dir = DirForward
	} else if strings.ToLower(parts[1]) == "reverse" {
		dir = DirReverse
	} else {
		err = fmt.Errorf("unknown Direction %q", parts[1])
		return
	}

	mig, err = newMutation(ts, dir, parts[2])
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
// "2006010215040506.${direction}.${name}.sql".
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
	if directory == nil {
		if directory, err = os.Open(DefaultMigrationFileDirectory); err != nil {
			return nil, err
		}
	}
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
			err = fmt.Errorf("specified no version, did not find one to apply")
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
		e = fmt.Errorf("could not find matching files")
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

func connect(driverName string, dsnParams DSNParams) (db *sql.DB, err error) {
	db, err = sql.Open(driverName, dsnParams.String())
	return
}

// DSNParams describes a type which generates a data source name for connecting
// to a database. The output will be passed to the standard library's sql.Open
// method to connect to a database.
type DSNParams interface {
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
	DSNParams() DSNParams
	CreateSchemaMigrationsTable() error
	DumpSchema() error
	// Execute runs the schema change and commits it to the database. The query
	// parameter is a SQL string and may contain placeholders for the values in
	// args. Input should be passed to conn so it could be sanitized, escaped.
	Execute(query string, args ...interface{}) error
	// AppliedVersions returns a list of migration versions that have been
	// executed against the database.
	AppliedVersions() (AppliedVersions, error)
	UpdateSchemaMigrations(dir Direction, version string) error
}

// NewDriver initializes a Driver implementation by name and connection
// parameters. An unrecognized name returns an error. The dsnParams should also
// provide whatever is needed by the Driver.
func NewDriver(driverName string, dsnParams DSNParams) (driver Driver, err error) {
	switch driverName {
	case "postgres":
		params, ok := dsnParams.(PostgresParams)
		if !ok {
			err = fmt.Errorf("dsnParams should be a PostgresParams, got %T", params)
		} else {
			driver, err = newPostgres(params)
		}
	default:
		err = fmt.Errorf("unknown driver %q", driverName)
	}
	return
}

// MigrationsConf is intended to lend customizations such as specifying the path
// to the migration files.
type MigrationsConf struct {
	PathToFiles string
}

// AppliedVersions represents an iterative list of migrations that have been run
// against the database and have been recorded in the schema migrations table.
// It's enough to convert a sql.Rows struct when implementing the Driver
// interface since a sql.Rows already satisfies this interface. See the existing
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
	}); err != nil {
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
		var mig Migration
		if mig, err = parseMigration(filename(fn)); err != nil {
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
	fmt.Printf("\t%-20s | %-10s | %-s\n", "version", "direction", "name")
	fmt.Printf("\t%-20s | %-10s | %-s\n", "-------", "---------", "----")
	for _, mig := range migrations {
		fmt.Printf(
			"\t%-20s | %-10s | %-s\n",
			mig.Timestamp().Format(TimeFormat), mig.Direction(), mig.Name(),
		)
	}
}
