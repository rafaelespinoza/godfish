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

	// this is a database driver, imported for side effects only, so we can
	// connect using the sql package.
	_ "github.com/lib/pq"
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
func Migrate(driver Driver, direction Direction, directoryPath string) (err error) {
	var migrations []Migration
	var dbHandler *sql.DB
	var finishAtVersion string
	if direction == DirForward {
		finishAtVersion = maxVersion
	} else {
		finishAtVersion = minVersion
	}

	if migrations, err = listAllAvailableMigrations(direction, directoryPath, finishAtVersion); err != nil {
		return
	}

	if dbHandler, err = connect(driver.Name(), driver.DSNParams()); err != nil {
		return
	}
	for _, mig := range migrations {
		var pathToFile string
		if pathToFile, err = pathToMigrationFile(directoryPath, mig); err != nil {
			return
		}
		if err = runMigration(dbHandler, driver, pathToFile, mig); err != nil {
			return
		}
	}
	return
}

func ApplyMigration(driver Driver, direction Direction, directoryPath, version string) (err error) {
	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}

	var baseGlob filename
	var filenames []string
	var mig Migration
	var dbHandler *sql.DB
	var pathToFile string

	if baseGlob, err = makeFilename(version, direction, "*"); err != nil {
		return
	}
	if filenames, err = filepath.Glob(directoryPath + "/" + string(baseGlob)); err != nil {
		return
	} else if len(filenames) == 0 {
		err = fmt.Errorf("could not find matching files")
		return
	} else if len(filenames) > 1 {
		err = fmt.Errorf("need 1 matching filename; got %v", filenames)
		return
	}
	if mig, err = parseMigration(filename(filenames[0])); err != nil {
		return
	}
	if dbHandler, err = connect(driver.Name(), driver.DSNParams()); err != nil {
		return
	}
	if pathToFile, err = pathToMigrationFile(directoryPath, mig); err != nil {
		return
	}
	err = runMigration(dbHandler, driver, pathToFile, mig)
	return
}

// runMigration executes a migration against the database. The input, pathToFile
// should be relative to the current working directory.
func runMigration(conn *sql.DB, driver Driver, pathToFile string, mig Migration) (err error) {
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
	if _, err = conn.Query(string(data)); err != nil {
		return
	}
	if err = driver.CreateSchemaMigrationsTable(conn); err != nil {
		return
	}
	err = driver.UpdateSchemaMigrations(
		conn,
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
	// Name should return the name of driver: ie: postgres, mysql, etc
	Name() string
	DSNParams() DSNParams
	CreateSchemaMigrationsTable(conn *sql.DB) error
	DumpSchema() error
	// AppliedVersions returns a list of migration versions that have been
	// executed against your database.
	AppliedVersions(conn *sql.DB) (*sql.Rows, error)
	UpdateSchemaMigrations(conn *sql.DB, dir Direction, version string) error
}

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

type MigrationsConf struct {
	PathToFiles string
}

func CreateSchemaMigrationsTable(driver Driver) error {
	conn, err := connect(driver.Name(), driver.DSNParams())
	if err != nil {
		return err
	}
	return driver.CreateSchemaMigrationsTable(conn)
}

func DumpSchema(driver Driver) error {
	return driver.DumpSchema()
}

// Info displays the outputs of various helper functions.
func Info(driver Driver, direction Direction, path string) (err error) {
	var migs []Migration
	var appliedVersions, availableVersions, versionsToApply []string
	var finishAtVersion string
	if direction == DirForward {
		finishAtVersion = maxVersion
	} else {
		finishAtVersion = minVersion
	}

	if migs, err = listAllAvailableMigrations(direction, path, finishAtVersion); err != nil {
		return
	}
	fmt.Println("-- all available migrations")
	for _, mig := range migs {
		fmt.Printf("%#v\n", mig)
	}
	if appliedVersions, err = listAppliedVersions(driver); err != nil {
		return
	}
	fmt.Println("-- applied versions")
	for _, version := range appliedVersions {
		fmt.Println(version)
	}
	availableVersions = listAvailableVersions(migs)
	fmt.Println("-- available versions")
	for _, version := range availableVersions {
		fmt.Println(version)
	}
	if versionsToApply, err = listVersionsToApply(
		direction,
		appliedVersions,
		availableVersions,
	); err != nil {
		return
	}
	fmt.Println("-- versions to apply")
	for _, version := range versionsToApply {
		fmt.Println(version)
	}
	return
}

// listAllAvailableMigrations returns a list of Migration values at path in a
// specified direction.
func listAllAvailableMigrations(direction Direction, path, finishAtVersion string) (out []Migration, err error) {
	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}
	var fileDir *os.File
	var filenames []string
	var finish time.Time
	if fileDir, err = os.Open(path); err != nil {
		return
	}
	defer fileDir.Close()
	if filenames, err = fileDir.Readdirnames(0); err != nil {
		return
	}
	sort.Strings(filenames)
	if finish, err = time.Parse(TimeFormat, finishAtVersion); err != nil {
		return
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
		timestamp := mig.Timestamp()
		if dir == DirForward && timestamp.After(finish) {
			continue
		} else if dir == DirReverse && timestamp.Before(finish) {
			continue
		}
		out = append(out, mig)
	}
	return
}

func listAppliedVersions(driver Driver) (out []string, err error) {
	var conn *sql.DB
	var rows *sql.Rows
	if conn, err = connect(driver.Name(), driver.DSNParams()); err != nil {
		return
	}
	if rows, err = driver.AppliedVersions(conn); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var version string
		if err = rows.Scan(&version); err != nil {
			return
		}
		out = append(out, version)
	}
	return
}

// listAvailableVersions extracts the versions from migration files and formats
// them to TimeFormat.
func listAvailableVersions(migrations []Migration) []string {
	out := make([]string, len(migrations))
	for i, mig := range migrations {
		out[i] = mig.Timestamp().Format(TimeFormat)
	}
	return out
}

func listVersionsToApply(direction Direction, applied, available []string) (out []string, err error) {
	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}

	// Collect versions in 3 "sets". Using empty struct as value because its
	// storage size is 0 bytes.
	allVersions := make(map[string]struct{})
	uniqueToApplied := make(map[string]struct{})
	for _, version := range applied {
		uniqueToApplied[version] = struct{}{}
		allVersions[version] = struct{}{}
	}
	uniqueToAvailable := make(map[string]struct{})
	for _, version := range available {
		if _, ok := uniqueToApplied[version]; ok {
			delete(uniqueToApplied, version)
		} else {
			uniqueToAvailable[version] = struct{}{}
			allVersions[version] = struct{}{}
		}
	}

	if direction == DirForward {
		for v := range allVersions {
			_, isApplied := uniqueToApplied[v]
			_, isAvailable := uniqueToAvailable[v]
			if !isApplied && isAvailable {
				out = append(out, v)
			}
		}
	} else {
		for v := range allVersions {
			_, appliedOK := uniqueToApplied[v]
			_, availableOK := uniqueToAvailable[v]
			if !appliedOK && !availableOK {
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return
}
