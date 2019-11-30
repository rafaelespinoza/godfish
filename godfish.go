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

// Filename is just a string with a specific format to migration files. One part
// has a generated timestamp, one part has a direction, another has a name.
type Filename string

// MakeFilename creates a filename based on the independent parts. Format:
// "2006010215040506.${direction}.${name}.sql"
func MakeFilename(version string, direction Direction, name string) (Filename, error) {
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
	return Filename(head + dir + tail), nil
}

func ParseMutation(filename Filename) (mut Mutation, err error) {
	var ts time.Time
	var dir Direction
	base := filepath.Base(string(filename))
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

	mut, err = NewModification(ts, dir, parts[2])
	return
}

// A Mutation is a database change with a direction name and timestamp.
// Typically, a Mutation with a DirForward Direction is paired with another
// migration of DirReverse that has the same name.
type Mutation interface {
	Direction() Direction
	Name() string
	Timestamp() time.Time
}

// Modification implements the Mutation interface.
type Modification struct {
	direction Direction
	name      string
	timestamp time.Time
}

var _ Mutation = (*Modification)(nil)

// NewModification constructs a Modification and returns a pointer. Its internal
// timestamp field is set to UTC.
func NewModification(ts time.Time, dir Direction, name string) (*Modification, error) {
	if dir == DirUnknown {
		return nil, fmt.Errorf("cannot have unknown direction")
	}
	return &Modification{
		direction: dir,
		name:      name,
		timestamp: ts.UTC(),
	}, nil
}

func (m *Modification) Direction() Direction { return m.direction }
func (m *Modification) Name() string         { return m.name }
func (m *Modification) Timestamp() time.Time { return m.timestamp }

// A Migration composes database changes in forward and reverse directions.
// Setting Reversible to true will generate a migration file for each direction.
// Otherwise, it only generates a file in the forward direction.
type Migration struct {
	Forward    Mutation
	Reverse    Mutation
	Reversible bool
	Dir        *os.File
}

// NewMigration constructs a Migration value that's ready to use.
func NewMigration(name string, reversible bool, dir *os.File) (*Migration, error) {
	var out Migration
	var err error
	var info os.FileInfo
	if dir == nil {
		if dir, err = os.Open(DefaultMigrationFileDirectory); err != nil {
			return nil, err
		}
	}
	if info, err = dir.Stat(); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("input dir %q should be a directory", info.Name())
	}
	out.Dir = dir

	out.Reversible = reversible
	timestamp := time.Now()
	var mod *Modification
	if mod, err = NewModification(timestamp, DirForward, name); err != nil {
		return nil, err
	}
	out.Forward = mod
	if mod, err = NewModification(timestamp, DirReverse, name); err != nil {
		return nil, err
	}
	out.Reverse = mod
	return &out, nil
}

// GenerateFiles creates the forward and reverse migration files if the
// migration is reversible, otherwise is generates just one migration file in
// the forward direction. It closes each file handle when it's done.
func (m *Migration) GenerateFiles() (err error) {
	var forwardFile, reverseFile *os.File
	defer func() {
		forwardFile.Close()
		reverseFile.Close()
	}()
	baseDir := m.Dir.Name()
	if forwardFile, err = newMutationFile(m.Forward, baseDir); err != nil {
		return
	}
	log.Println("created forward file, ", forwardFile.Name())
	if !m.Reversible {
		log.Println("migration marked irreversible, did not create reverse file")
		return
	}
	if reverseFile, err = newMutationFile(m.Reverse, baseDir); err != nil {
		return
	}
	log.Println("created reverse file, ", reverseFile.Name())
	return
}

func newMutationFile(m Mutation, baseDir string) (*os.File, error) {
	filename, err := MakeMutationFilename(m)
	if err != nil {
		return nil, err
	}
	return os.Create(baseDir + "/" + string(filename))
}

// MakeMutationFilename passes in a Mutation's fields to create a Filename. An
// error could be returned if mut is found to be an unsuitable Filename.
func MakeMutationFilename(m Mutation) (Filename, error) {
	return MakeFilename(
		m.Timestamp().Format(TimeFormat),
		m.Direction(),
		m.Name(),
	)
}

// PathToMutationFile is a convenience function for prepending a directory path
// to the base filename of a mutation. An error could be returned if the
// Mutation's fields are unsuitable for a Filename.
func PathToMutationFile(dir string, mut Mutation) (string, error) {
	filename, err := MakeMutationFilename(mut)
	if err != nil {
		return "", err
	}
	return dir + "/" + string(filename), nil
}

// RunMutation executes a migration against the database. It takes in a database
// connection handler and a path to the migration file. The pathToMigrationFile
// should be relative to your current working directory.
func RunMutation(db *sql.DB, pathToMigrationFile string) (err error) {
	var file *os.File
	var info os.FileInfo
	var rows *sql.Rows
	defer file.Close()
	if file, err = os.Open(pathToMigrationFile); err != nil {
		return
	}
	if info, err = file.Stat(); err != nil {
		return
	}
	data := make([]byte, int(info.Size()))
	if _, err = file.Read(data); err != nil {
		return
	}
	if rows, err = db.Query(string(data)); err != nil {
		return
	}
	log.Printf("%v\n", rows)
	return nil
}

func Connect(driverName string, dsnParams DSNParams) (db *sql.DB, err error) {
	db, err = sql.Open(driverName, dsnParams.String())
	return
}

// DSNParams describes a type which generates a data source name (in other
// words, a connection URL) for connecting to a database. The output will be
// passed to the standard library's sql.Open method to connect to a database.
type DSNParams interface {
	String() string
}

type Driver interface {
	Name() string
	DSNParams() DSNParams
	CreateSchemaMigrationsTable(conn *sql.DB) error
	DumpSchema() error
	AppliedVersions(conn *sql.DB) (*sql.Rows, error)
	ApplyMigration(conn *sql.DB, version string, dir Direction) error
}

func MakeDriver(driverName string) (driver Driver, err error) {
	switch driverName {
	case "postgres":
		driver, err = NewPostgres()
	default:
		err = fmt.Errorf("unknown driver %q", driverName)
	}
	return
}

type MigrationsConf struct {
	PathToFiles string
}

func CreateSchemaMigrationsTable(driver Driver) error {
	conn, err := Connect(driver.Name(), driver.DSNParams())
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
	var muts []Mutation
	var appliedVersions, availableVersions, versionsToApply []string
	if muts, err = AllAvailableMutations(direction, path); err != nil {
		return
	}
	fmt.Println("-- all available mutations")
	for _, mut := range muts {
		fmt.Printf("%#v\n", mut)
	}
	if appliedVersions, err = listAppliedVersions(driver); err != nil {
		return
	}
	fmt.Println("-- applied versions")
	for _, version := range appliedVersions {
		fmt.Println(version)
	}
	availableVersions = listAvailableVersions(muts)
	fmt.Println("-- available versions")
	for _, version := range availableVersions {
		fmt.Println(version)
	}
	if versionsToApply, err = ListVersionsToApply(
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

// AllAvailableMutations returns a list of Mutation values at path in a
// specified direction.
func AllAvailableMutations(direction Direction, path string) (out []Mutation, err error) {
	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
		return
	}
	var fileDir *os.File
	var filenames []string
	if fileDir, err = os.Open(path); err != nil {
		return
	}
	defer fileDir.Close()
	if filenames, err = fileDir.Readdirnames(0); err != nil {
		return
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		var mut Mutation
		if mut, err = ParseMutation(Filename(filename)); err != nil {
			return
		}
		if mut.Direction() != direction {
			continue
		}
		out = append(out, mut)
	}
	return
}

func listAppliedVersions(driver Driver) (out []string, err error) {
	var conn *sql.DB
	var rows *sql.Rows
	if conn, err = Connect(driver.Name(), driver.DSNParams()); err != nil {
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

// listAvailableVersions extracts the versions from mutation files and formats
// them to TimeFormat.
func listAvailableVersions(mutations []Mutation) []string {
	out := make([]string, len(mutations))
	for i, mut := range mutations {
		out[i] = mut.Timestamp().Format(TimeFormat)
	}
	return out
}

func ListVersionsToApply(direction Direction, applied, available []string) (out []string, err error) {
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
