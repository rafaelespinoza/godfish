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
	"strconv"
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
	TimeFormat = "20060102150405"

	filenameDelimeter   = "-"
	unixTimestampSecLen = len("1574079194")
)

var timeformatMatcher = regexp.MustCompile(`\d{4,14}`)

type (
	// Indirection associates a label with a migration direction. It helps with
	// interpreting filenames.
	Indirection struct {
		Value Direction
		Label string
	}

	// Version is for comparing migrations to each other.
	Version interface {
		Before(u Version) bool
		String() string
		Value() int64
	}

	timestamp struct {
		value int64
		label string
	}
)

var _ Version = (*timestamp)(nil)

func (v *timestamp) Before(u Version) bool {
	// Until there's more than 1 interface implementation, this is fine. So,
	// panic here?  Yeah, maybe. Fail loudly, not silently.
	w := u.(*timestamp)
	return v.value < w.value
}

func (v *timestamp) String() string {
	if v.label == "" {
		return strconv.FormatInt(int64(v.value), 10)
	}
	return v.label
}

func (v *timestamp) Value() int64 { return v.value }

// filename is just a string with a specific format to migration files. One part
// has a generated timestamp, one part has a direction, another has a label.
type filename string

// makeFilename creates a filename based on the independent parts. Format:
// "${direction}-${version}-${label}.sql"
func makeFilename(version string, indirection Indirection, label string) (filename, error) {
	var dir string
	if indirection.Value == DirUnknown {
		dir = "*" + filenameDelimeter
	} else {
		dir = strings.ToLower(indirection.Label) + filenameDelimeter
	}

	// the length will top out at the high quantifier for this regexp.
	ver := timeformatMatcher.FindString(version) + filenameDelimeter
	return filename(dir + ver + label + ".sql"), nil
}

var (
	forwardDirections = []string{
		strings.ToLower(DirForward.String()),
		"migrate",
		"up",
	}
	reverseDirections = []string{
		strings.ToLower(DirReverse.String()),
		"rollback",
		"down",
	}
)

func parseDirection(basename string) (dir Indirection) {
	lo := strings.ToLower(basename)
	for _, pre := range forwardDirections {
		if strings.HasPrefix(lo, pre) {
			dir.Value = DirForward
			dir.Label = pre
			return
		}
	}
	for _, pre := range reverseDirections {
		if strings.HasPrefix(lo, pre) {
			dir.Value = DirReverse
			dir.Label = pre
			return
		}
	}
	return
}

func parseVersion(basename string) (version Version, err error) {
	written := timeformatMatcher.FindString(basename)
	if ts, perr := time.Parse(TimeFormat, written); perr != nil {
		err = perr // keep going
	} else {
		version = &timestamp{value: ts.UTC().Unix(), label: written}
		return
	}

	if perr, ok := err.(*time.ParseError); ok {
		if len(perr.Value) < len(TimeFormat) {
			ts, qerr := time.Parse(TimeFormat[:len(perr.Value)], perr.Value)
			if qerr == nil {
				version = &timestamp{value: ts.UTC().Unix(), label: perr.Value}
				err = nil
				return
			}
		}
	}

	// try parsing as unix epoch timestamp
	num, err := strconv.ParseInt(written[:unixTimestampSecLen], 10, 64)
	if err != nil {
		return
	}
	version = &timestamp{value: num, label: written}
	return
}

func parseMigration(name filename) (mig Migration, err error) {
	basename := filepath.Base(string(name))
	direction := parseDirection(basename)
	if direction.Value == DirUnknown {
		err = fmt.Errorf(
			"%w; could not parse Direction for filename %q",
			errDataInvalid, name,
		)
		return
	}

	// index of the start of timestamp
	i := len(direction.Label) + len(filenameDelimeter)
	version, err := parseVersion(basename)
	if err != nil {
		err = fmt.Errorf(
			"%w, could not parse timestamp for filename %q; %v",
			errDataInvalid, version, err,
		)
		return
	}

	// index of the start of migration label
	j := i + len(version.String()) + len(filenameDelimeter)
	mig, err = newMutation(
		version,
		direction,
		strings.TrimSuffix(string(basename[j:]), ".sql"),
	)
	return
}

// A Migration is a database change with a direction name and timestamp.
// Typically, a Migration with a DirForward Direction is paired with another
// migration of DirReverse that has the same label.
type Migration interface {
	Indirection() Indirection
	Label() string
	Version() Version
}

// Basename generates a migration file's basename. The output format is:
// "${direction}-${timestamp}-${label}.sql".
func Basename(mig Migration) (string, error) {
	out, err := makeMigrationFilename(mig)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// mutation implements the Migration interface.
type mutation struct {
	indirection Indirection
	label       string
	timestamp   Version
}

var _ Migration = (*mutation)(nil)

// newMutation constructs a mutation and returns a pointer. Its internal
// timestamp field is set to UTC.
func newMutation(version Version, ind Indirection, label string) (*mutation, error) {
	if ind.Value == DirUnknown {
		return nil, fmt.Errorf("cannot have unknown direction")
	}
	return &mutation{
		indirection: ind,
		label:       label,
		timestamp:   version,
	}, nil
}

func (m *mutation) Indirection() Indirection { return m.indirection }
func (m *mutation) Label() string            { return m.label }
func (m *mutation) Version() Version         { return m.timestamp }

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
func NewMigrationParams(label string, reversible bool, directory *os.File) (*MigrationParams, error) {
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
	now := time.Now().UTC()
	version := &timestamp{value: now.Unix(), label: now.Format(TimeFormat)}
	var mut *mutation
	if mut, err = newMutation(version, Indirection{Value: DirForward, Label: "forward"}, label); err != nil {
		return nil, err
	}
	out.Forward = mut
	if mut, err = newMutation(version, Indirection{Value: DirReverse, Label: "reverse"}, label); err != nil {
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
		m.Version().String(),
		m.Indirection(),
		m.Label(),
	)
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
	finder := migrationFinder{
		direction:       direction,
		directoryPath:   directoryPath,
		finishAtVersion: finishAtVersion,
		verbose:         false,
	}
	if migrations, err = finder.query(driver); err != nil {
		return
	}

	for _, mig := range migrations {
		fn, ierr := makeMigrationFilename(mig)
		if ierr != nil {
			return
		}
		pathToFile := directoryPath + "/" + string(fn)
		if err = runMigration(driver, pathToFile, mig); err != nil {
			return
		}
	}
	return
}

var (
	// ErrSchemaMigrationsDoesNotExist means there is no database table to
	// record migration status.
	ErrSchemaMigrationsDoesNotExist = errors.New("schema migrations table does not exist")

	errNotFound    = errors.New("not found")
	errDataInvalid = errors.New("data invalid")
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
		var limit string
		if direction == DirForward {
			limit = maxVersion
		}
		finder := migrationFinder{
			direction:       direction,
			directoryPath:   directoryPath,
			finishAtVersion: limit,
			verbose:         false,
		}
		if toApply, ierr := finder.query(driver); ierr != nil {
			err = fmt.Errorf("specified no version; error attempting to find one; %v", ierr)
			return
		} else if len(toApply) < 1 {
			err = fmt.Errorf("version %w", errNotFound)
			return
		} else {
			version = toApply[0].Version().String()
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
	// glob as many filenames as possible that match the "version" segment, then
	// narrow it down from there.
	if baseGlob, e = makeFilename(version, Indirection{}, "*"); e != nil {
		return
	}
	glob := directoryPath + "/" + string(baseGlob)
	if filenames, e = filepath.Glob(glob); e != nil {
		return
	}

	var directionNames []string
	if direction == DirForward {
		directionNames = forwardDirections
	} else if direction == DirReverse {
		directionNames = reverseDirections
	}

	for _, fn := range filenames {
		for _, alias := range directionNames {
			if strings.HasPrefix(filepath.Base(fn), alias) {
				f = fn
				return
			}
		}
	}
	if f == "" {
		e = fmt.Errorf("files %w", errNotFound)
	}
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
		mig.Indirection().Value,
		mig.Version().String(),
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
	finder := migrationFinder{
		direction:       direction,
		directoryPath:   directoryPath,
		finishAtVersion: finishAtVersion,
		verbose:         true,
	}
	_, err = finder.query(driver)
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

// migrationFinder is a collection of named parameters to use when searching
// for migrations to apply.
type migrationFinder struct {
	direction       Direction
	directoryPath   string
	finishAtVersion string
	verbose         bool
}

// query returns a list of Migrations to apply.
func (m *migrationFinder) query(driver Driver) (out []Migration, err error) {
	available, err := m.available()
	if err != nil {
		return
	}
	if m.verbose {
		fmt.Println("-- All available migrations")
		printMigrations(available)
		fmt.Println()
	}

	applied, err := scanAppliedVersions(driver, m.directoryPath)
	if err == ErrSchemaMigrationsDoesNotExist {
		// The next invocation of CreateSchemaMigrationsTable should fix this.
		// We can continue with zero value for now.
		if m.verbose {
			fmt.Printf("no migrations applied yet; %v\n", err)
		}
	} else if err != nil {
		return
	}
	if m.verbose {
		fmt.Println("-- Applied migrations")
		printMigrations(applied)
		fmt.Println()
	}

	toApply, err := m.filter(applied, available)
	if err != nil {
		return
	}
	var useDefaultRollbackVersion bool
	if m.finishAtVersion == "" && m.direction == DirForward {
		m.finishAtVersion = maxVersion
	} else if m.finishAtVersion == "" && m.direction == DirReverse {
		if len(toApply) == 0 {
			return
		}
		useDefaultRollbackVersion = true
		m.finishAtVersion = toApply[0].Version().String()
	}
	var finish Version
	if finish, err = parseVersion(m.finishAtVersion); err != nil {
		return
	}
	for _, mig := range toApply {
		timestamp := mig.Version()
		if m.direction == DirForward && finish.Before(timestamp) {
			break
		}
		if m.direction == DirReverse {
			if timestamp.Before(finish) {
				break
			}
			if !useDefaultRollbackVersion && timestamp.Before(finish) {
				break
			}
		}
		out = append(out, mig)
	}
	if m.verbose {
		fmt.Println("-- Migrations to apply")
		printMigrations(out)
	}
	return
}

// available returns a list of Migration values in a specified direction.
func (m *migrationFinder) available() (out []Migration, err error) {
	var fileDir *os.File
	var filenames []string
	if fileDir, err = os.Open(m.directoryPath); err != nil {
		if _, ok := err.(*os.PathError); ok {
			err = fmt.Errorf("path to migration files %q %w", m.directoryPath, errNotFound)
		}
		return
	}
	defer fileDir.Close()
	if filenames, err = fileDir.Readdirnames(0); err != nil {
		return
	}
	if m.direction == DirForward {
		sort.Strings(filenames)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(filenames)))
	}
	for _, fn := range filenames {
		mig, ierr := parseMigration(filename(fn))
		if errors.Is(ierr, errDataInvalid) {
			fmt.Println(ierr)
			continue
		} else if ierr != nil {
			err = ierr
			return
		}
		dir := mig.Indirection().Value
		if dir != m.direction {
			continue
		}
		out = append(out, mig)
	}
	return
}

func scanAppliedVersions(driver Driver, directoryPath string) (out []Migration, err error) {
	var rows AppliedVersions
	if rows, err = driver.AppliedVersions(); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var version, basename string
		var mig Migration
		if err = rows.Scan(&version); err != nil {
			return
		}
		basename, err = figureOutBasename(directoryPath, DirForward, version)
		if errors.Is(err, errNotFound) {
			err = nil // swallow error and continue
		} else if err != nil {
			return
		}
		mig, err = parseMigration(filename(basename))
		if errors.Is(err, errDataInvalid) {
			err = nil // swallow error and continue
		} else if mig != nil {
			out = append(out, mig)
		}
	}
	return
}

// filter compares lists of applied and available migrations, then selects a
// list of migrations to apply.
func (m *migrationFinder) filter(applied, available []Migration) (out []Migration, err error) {
	allVersions := make(map[int64]Migration)
	uniqueToApplied := make(map[int64]Migration)
	for _, mig := range applied {
		version := mig.Version().Value()
		uniqueToApplied[version] = mig
		allVersions[version] = mig
	}
	uniqueToAvailable := make(map[int64]Migration)
	for _, mig := range available {
		version := mig.Version().Value()
		if _, ok := uniqueToApplied[version]; ok {
			delete(uniqueToApplied, version)
		} else {
			uniqueToAvailable[version] = mig
			allVersions[version] = mig
		}
	}

	if m.direction == DirForward {
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
				// The Migration direction is artificially set to Forward from a
				// previous step. Here, we correct it. Also, we're guessing what
				// the original filename was, by assuming that the list of
				// forward directions is in the same order as the corresponding
				// reverse directions. It's kind of hacky, I know.
				mut, ierr := newMutation(mig.Version(), Indirection{Value: DirReverse}, mig.Label())
				if ierr != nil {
					err = ierr
					return
				}
				for i, fwd := range forwardDirections {
					// Another assumption, the filename format will never
					// change. If it does change, for example: it is
					// "${version}-${direction}-${label}", instead of
					// "${direction}-${version}-${label}", then this won't work.
					if mig.Indirection().Label == fwd {
						mut.indirection.Label = reverseDirections[i]
						break
					}
				}
				if mut.indirection.Label == "" {
					err = fmt.Errorf(
						"direction.Label empty; direction.Value: %q, version: %v, label: %q",
						mut.indirection.Value, mut.timestamp, mut.label,
					)
					return
				}
				out = append(out, mut)
			}
		}
	}
	if m.direction == DirForward {
		sort.Slice(out, func(i, j int) bool {
			return out[i].Version().Before(out[j].Version())
		})
	} else {
		sort.Slice(out, func(i, j int) bool {
			return out[j].Version().Before(out[i].Version())
		})
	}
	return
}

func printMigrations(migrations []Migration) {
	fmt.Printf("\t%-10s | %-20s | %-s\n", "direction", "version", "label")
	fmt.Printf("\t%-10s | %-20s | %-s\n", "---------", "-------", "----")
	for _, mig := range migrations {
		fmt.Printf(
			"\t%-10s | %-20s | %-s\n",
			mig.Indirection().Value, mig.Version().String(), mig.Label(),
		)
	}
}
