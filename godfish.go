// Package godfish is a database migration library built to support the command
// line tool.
package godfish

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rafaelespinoza/godfish/internal"
)

// CreateMigrationFiles takes care of setting up a new DB migration by
// generating empty migration files in a directory at dirpath. Passing in true
// for reversible means that a complementary file will be made for rollbacks.
// Names for directions in the filename could be overridden from their default
// values (forward and reverse) with the input vars fwdlabel, revlabel when
// non-empty.
func CreateMigrationFiles(migrationName string, reversible bool, dirpath, fwdlabel, revlabel string) (err error) {
	params, err := internal.NewMigrationParams(migrationName, reversible, dirpath, fwdlabel, revlabel)
	if err != nil {
		return
	}
	err = params.GenerateFiles()
	return
}

// Migrate executes all migrations at directoryPath in the specified direction.
func Migrate(driver Driver, directoryPath string, forward bool, finishAtVersion string) (err error) {
	var (
		dsn        string
		migrations []internal.Migration
	)
	if dsn, err = getDSN(); err != nil {
		return
	}
	if err = driver.Connect(dsn); err != nil {
		return
	}
	defer driver.Close()

	direction := internal.DirReverse
	if forward {
		direction = internal.DirForward
	}

	if finishAtVersion == "" && direction == internal.DirForward {
		finishAtVersion = internal.MaxVersion
	} else if finishAtVersion == "" && direction == internal.DirReverse {
		finishAtVersion = internal.MinVersion
	}

	finder := migrationFinder{
		direction:       direction,
		directoryPath:   directoryPath,
		finishAtVersion: finishAtVersion,
	}
	if migrations, err = finder.query(driver); err != nil {
		return
	}

	for _, mig := range migrations {
		pathToFile := directoryPath + "/" + internal.MakeMigrationFilename(mig)
		if err = runMigration(driver, pathToFile, mig); err != nil {
			return
		}
	}
	return
}

// ErrSchemaMigrationsDoesNotExist means there is no database table to
// record migration status.
var ErrSchemaMigrationsDoesNotExist = errors.New("schema migrations table does not exist")

// ApplyMigration runs a migration at directoryPath with the specified version
// and direction.
func ApplyMigration(driver Driver, directoryPath string, forward bool, version string) (err error) {
	var (
		dsn        string
		pathToFile string
		mig        internal.Migration
	)

	if dsn, err = getDSN(); err != nil {
		return
	}
	if err = driver.Connect(dsn); err != nil {
		return
	}
	defer driver.Close()

	direction := internal.DirReverse
	if forward {
		direction = internal.DirForward
	}

	if version == "" {
		// attempt to find the next version to apply in the direction
		var limit string
		if direction == internal.DirForward {
			limit = internal.MaxVersion
		}
		finder := migrationFinder{
			direction:       direction,
			directoryPath:   directoryPath,
			finishAtVersion: limit,
		}
		if toApply, ierr := finder.query(driver); ierr != nil {
			err = fmt.Errorf("specified no version; error attempting to find one; %v", ierr)
			return
		} else if len(toApply) < 1 {
			err = fmt.Errorf("version %w", internal.ErrNotFound)
			return
		} else {
			version = toApply[0].Version().String()
		}
	}

	if pathToFile, err = figureOutBasename(directoryPath, direction, version); err != nil {
		return
	}
	fn := internal.Filename(directoryPath + "/" + pathToFile)
	if mig, err = internal.ParseMigration(fn); err != nil {
		return
	}
	err = runMigration(driver, pathToFile, mig)
	return
}

func figureOutBasename(directoryPath string, direction internal.Direction, version string) (f string, e error) {
	var filenames []string
	// glob as many filenames as possible that match the "version" segment, then
	// narrow it down from there.
	glob := directoryPath + "/" + internal.MakeFilename(version, internal.Indirection{}, "*")
	if filenames, e = filepath.Glob(glob); e != nil {
		return
	}

	var directionNames []string
	if direction == internal.DirForward {
		directionNames = internal.ForwardDirections
	} else if direction == internal.DirReverse {
		directionNames = internal.ReverseDirections
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
		e = fmt.Errorf("files %w", internal.ErrNotFound)
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
func runMigration(driver Driver, pathToFile string, mig internal.Migration) (err error) {
	var data []byte
	if data, err = os.ReadFile(pathToFile); err != nil {
		return
	}
	gerund := "migrating"
	if mig.Indirection().Value == internal.DirReverse {
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
		mig.Indirection().Value == internal.DirForward,
		mig.Version().String(),
	)
	if err == nil {
		fmt.Println("ok")
	}
	return
}

// Info writes status of migrations to w in formats json or tsv.
func Info(driver Driver, directoryPath string, forward bool, finishAtVersion string, w io.Writer, format string) (err error) {
	var dsn string
	if dsn, err = getDSN(); err != nil {
		return
	}
	if err = driver.Connect(dsn); err != nil {
		return err
	}
	defer driver.Close()

	direction := internal.DirReverse
	if forward {
		direction = internal.DirForward
	}

	finder := migrationFinder{
		direction:       direction,
		directoryPath:   directoryPath,
		finishAtVersion: finishAtVersion,
		infoPrinter:     choosePrinter(format, w),
	}
	_, err = finder.query(driver)
	return
}

func choosePrinter(format string, w io.Writer) (out internal.InfoPrinter) {
	if format == "json" {
		out = internal.NewJSON(w)
		return
	}

	if format != "tsv" && format != "" {
		fmt.Fprintf(os.Stderr, "unknown format %q, defaulting to tsv\n", format)
	}
	out = internal.NewTSV(w)
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
	if data, err = json.MarshalIndent(internal.Config{}, "", "\t"); err != nil {
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
	direction       internal.Direction
	directoryPath   string
	finishAtVersion string
	infoPrinter     internal.InfoPrinter
}

// query returns a list of Migrations to apply.
func (m *migrationFinder) query(driver Driver) (out []internal.Migration, err error) {
	available, err := m.available()
	if err != nil {
		return
	}

	applied, err := scanAppliedVersions(driver, m.directoryPath)
	if err == ErrSchemaMigrationsDoesNotExist {
		// The next invocation of CreateSchemaMigrationsTable should fix this.
		// We can continue with zero value for now.
		fmt.Fprintf(os.Stderr, "no migrations applied yet; %v\n", err)
	} else if err != nil {
		return
	}
	if m.infoPrinter != nil {
		if err = printMigrations(m.infoPrinter, "up", applied); err != nil {
			return
		}
	}

	toApply, err := m.filter(applied, available)
	if err != nil {
		return
	}
	var useDefaultRollbackVersion bool
	if m.finishAtVersion == "" && m.direction == internal.DirForward {
		m.finishAtVersion = internal.MaxVersion
	} else if m.finishAtVersion == "" && m.direction == internal.DirReverse {
		if len(toApply) == 0 {
			return
		}
		useDefaultRollbackVersion = true
		m.finishAtVersion = toApply[0].Version().String()
	}
	var finish internal.Version
	if finish, err = internal.ParseVersion(m.finishAtVersion); err != nil {
		return
	}
	for _, mig := range toApply {
		version := mig.Version()
		if m.direction == internal.DirForward && finish.Before(version) {
			break
		}
		if m.direction == internal.DirReverse {
			if version.Before(finish) {
				break
			}
			if !useDefaultRollbackVersion && version.Before(finish) {
				break
			}
		}
		out = append(out, mig)
	}
	if m.infoPrinter != nil {
		if err = printMigrations(m.infoPrinter, "down", out); err != nil {
			return
		}
	}
	return
}

// available returns a list of Migration values in a specified direction.
func (m *migrationFinder) available() (out []internal.Migration, err error) {
	var fileDir *os.File
	var filenames []string
	if fileDir, err = os.Open(m.directoryPath); err != nil {
		if _, ok := err.(*os.PathError); ok {
			err = fmt.Errorf("path to migration files %q %w", m.directoryPath, internal.ErrNotFound)
		}
		return
	}
	defer fileDir.Close()
	if filenames, err = fileDir.Readdirnames(0); err != nil {
		return
	}
	if m.direction == internal.DirForward {
		sort.Strings(filenames)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(filenames)))
	}
	for _, fn := range filenames {
		mig, ierr := internal.ParseMigration(internal.Filename(fn))
		if errors.Is(ierr, internal.ErrDataInvalid) {
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

func scanAppliedVersions(driver Driver, directoryPath string) (out []internal.Migration, err error) {
	var rows AppliedVersions
	if rows, err = driver.AppliedVersions(); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var version, basename string
		var mig internal.Migration
		if err = rows.Scan(&version); err != nil {
			return
		}
		basename, err = figureOutBasename(directoryPath, internal.DirForward, version)
		if errors.Is(err, internal.ErrNotFound) {
			err = nil // swallow error and continue
		} else if err != nil {
			return
		}
		mig, err = internal.ParseMigration(internal.Filename(basename))
		if errors.Is(err, internal.ErrDataInvalid) {
			err = nil // swallow error and continue
		} else if mig != nil {
			out = append(out, mig)
		}
	}
	return
}

// filter compares lists of applied and available migrations, then selects a
// list of migrations to apply.
func (m *migrationFinder) filter(applied, available []internal.Migration) (out []internal.Migration, err error) {
	allVersions := make(map[int64]internal.Migration)
	uniqueToApplied := make(map[int64]internal.Migration)
	for _, mig := range applied {
		version := mig.Version().Value()
		uniqueToApplied[version] = mig
		allVersions[version] = mig
	}
	uniqueToAvailable := make(map[int64]internal.Migration)
	for _, mig := range available {
		version := mig.Version().Value()
		if _, ok := uniqueToApplied[version]; ok {
			delete(uniqueToApplied, version)
		} else {
			uniqueToAvailable[version] = mig
			allVersions[version] = mig
		}
	}

	if m.direction == internal.DirForward {
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
				var mut internal.Migration
				indirection := internal.Indirection{
					Value: internal.DirReverse,
					Label: "reverse", // need to have something here, it gets restored later.
				}
				mut, err = newMigration(mig.Version().String(), indirection, mig.Label())
				if err != nil {
					return
				}
				for i, fwd := range internal.ForwardDirections {
					// Restore the direction label for reverse migration based
					// on corresponding label for the known forward migration.
					//
					// Another assumption, the filename format will never
					// change. If it does change, for example: it is
					// "${version}-${direction}-${label}", instead of
					// "${direction}-${version}-${label}", then this won't work.
					if mig.Indirection().Label == fwd {
						indirection.Label = internal.ReverseDirections[i]
						mut, err = newMigration(mig.Version().String(), indirection, mig.Label())
						if err != nil {
							return
						}
						break
					}
				}
				if mut.Label() == "" {
					err = fmt.Errorf(
						"direction.Label empty; direction.Value: %q, version: %v, label: %q",
						mut.Indirection(), mut.Version().String(), mut.Label(),
					)
					return
				}
				out = append(out, mut)
			}
		}
	}
	if m.direction == internal.DirForward {
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

func newMigration(version string, ind internal.Indirection, label string) (out internal.Migration, err error) {
	fn := internal.MakeFilename(version, ind, label)
	out, err = internal.ParseMigration(internal.Filename(fn))
	return
}

func printMigrations(p internal.InfoPrinter, state string, migrations []internal.Migration) (err error) {
	for i, mig := range migrations {
		if err = p.PrintInfo(state, mig); err != nil {
			err = fmt.Errorf("%w; item %d", err, i)
			return
		}
	}
	return
}

const dsnKey = "DB_DSN"

func getDSN() (dsn string, err error) {
	dsn = os.Getenv(dsnKey)
	if dsn == "" {
		err = fmt.Errorf("missing environment variable: %s", dsnKey)
	}
	return
}
