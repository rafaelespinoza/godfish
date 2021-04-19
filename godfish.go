// Package godfish is a database migration library built to support the command
// line tool.
package godfish

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migrate executes all migrations at directoryPath in the specified direction.
func Migrate(driver Driver, directoryPath string, direction Direction, finishAtVersion string) (err error) {
	var (
		dsn        string
		migrations []Migration
	)
	if dsn, err = getDSN(); err != nil {
		return
	}
	if _, err = driver.Connect(dsn); err != nil {
		return
	}
	defer driver.Close()

	if finishAtVersion == "" && direction == DirForward {
		finishAtVersion = maxVersion
	} else if finishAtVersion == "" && direction == DirReverse {
		finishAtVersion = minVersion
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
		pathToFile := directoryPath + "/" + makeMigrationFilename(mig)
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
	var (
		dsn        string
		pathToFile string
		mig        Migration
	)

	if dsn, err = getDSN(); err != nil {
		return
	}
	if _, err = driver.Connect(dsn); err != nil {
		return
	}
	defer driver.Close()

	if direction == DirUnknown {
		err = fmt.Errorf("unknown Direction %q", direction)
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
	var filenames []string
	// glob as many filenames as possible that match the "version" segment, then
	// narrow it down from there.
	glob := directoryPath + "/" + makeFilename(version, Indirection{}, "*")
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

type runMigrationError struct {
	driverName    string
	originalError error
	path          string
}

func (e *runMigrationError) Error() string {
	return fmt.Sprintf(
		"driver: %q, path: %q, error: %v",
		e.driverName, e.path, e.originalError,
	)
}

// runMigration executes a migration against the database. The input, pathToFile
// should be relative to the current working directory.
func runMigration(driver Driver, pathToFile string, mig Migration) (err error) {
	var data []byte
	if data, err = os.ReadFile(pathToFile); err != nil {
		return
	}
	gerund := "migrating"
	if mig.Indirection().Value == DirReverse {
		gerund = "rolling back"
	}
	fmt.Printf("%s version %q ... ", gerund, mig.Version().String())

	if err = driver.Execute(string(data)); err != nil {
		err = &runMigrationError{
			driverName:    driver.Name(),
			originalError: err,
			path:          pathToFile,
		}
		return
	}
	if err = driver.CreateSchemaMigrationsTable(); err != nil {
		return
	}
	err = driver.UpdateSchemaMigrations(
		mig.Indirection().Value,
		mig.Version().String(),
	)
	if err == nil {
		fmt.Println("ok")
	}
	return
}

// CreateSchemaMigrationsTable creates a table to track status of migrations on
// the database. Running any migration will create the table, so you don't
// usually need to call this function.
func CreateSchemaMigrationsTable(driver Driver) (err error) {
	var dsn string
	if dsn, err = getDSN(); err != nil {
		return
	}
	if _, err = driver.Connect(dsn); err != nil {
		return err
	}
	defer driver.Close()
	return driver.CreateSchemaMigrationsTable()
}

// Info displays the outputs of various helper functions.
func Info(driver Driver, directoryPath string, direction Direction, finishAtVersion string) (err error) {
	var dsn string
	if dsn, err = getDSN(); err != nil {
		return
	}
	if _, err = driver.Connect(dsn); err != nil {
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

// Config is for various runtime settings.
type Config struct {
	PathToFiles string `json:"path_to_files"`
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
	if data, err = json.MarshalIndent(Config{}, "", "\t"); err != nil {
		return err
	}
	return os.WriteFile(
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
		version := mig.Version()
		if m.direction == DirForward && finish.Before(version) {
			break
		}
		if m.direction == DirReverse {
			if version.Before(finish) {
				break
			}
			if !useDefaultRollbackVersion && version.Before(finish) {
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
				mut := &mutation{
					indirection: Indirection{Value: DirReverse},
					label:       mig.Label(),
					version:     mig.Version(),
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
						mut.indirection.Value, mut.version, mut.label,
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
	fmt.Printf("\t%-20s | %-s\n", "version", "basename")
	fmt.Printf("\t%-20s | %-s\n", "-------", "--------")
	for _, mig := range migrations {
		fmt.Printf("\t%-20s | %-s\n", mig.Version().String(), makeMigrationFilename(mig))
	}
}

const dsnKey = "DB_DSN"

func getDSN() (dsn string, err error) {
	dsn = os.Getenv(dsnKey)
	if dsn == "" {
		err = fmt.Errorf("missing environment variable: %s", dsnKey)
	}
	return
}
