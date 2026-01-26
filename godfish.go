// Package godfish is a database migration library built to support the command
// line tool.
package godfish

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

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

// Migrate executes all migrations at the directory dirFS in the specified
// direction. When forward is true, it will seek migrations with a forward
// direction and apply them up to and including the one with a version matching
// finishAtVersion. Likewise, when forward is false, then it seeks migrations
// with a reverse direction and runs them.
//
// The migrationsTable input sets the DB table for storing the current DB
// migration state. If empty, then it's set to a default value of
// "schema_migrations". The named DB table will be automatically created unless
// it already exists.
func Migrate(ctx context.Context, driver Driver, dirFS fs.FS, forward bool, finishAtVersion string, migrationsTable string) (err error) {
	migrationsTable = cmp.Or(migrationsTable, internal.DefaultMigrationsTableName)
	var migrations []*internal.Migration
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
		dirFS:           dirFS,
		finishAtVersion: finishAtVersion,
	}
	if migrations, err = finder.query(ctx, driver, migrationsTable); err != nil {
		return
	}

	for _, mig := range migrations {
		pathToFile := string(mig.ToFilename())
		if err = runMigration(ctx, driver, dirFS, pathToFile, mig, migrationsTable); err != nil {
			return
		}
	}
	return
}

// ErrSchemaMigrationsDoesNotExist means there is no database table to
// record migration status.
var ErrSchemaMigrationsDoesNotExist = errors.New("schema migrations table does not exist")

// ApplyMigration runs a migration at the directory dirFS with the specified
// version and direction. When forward is true, it will target a migration with
// a forward direction. Likewise when forward is false, then it targets a
// migration with a reverse direction.
//
// The migrationsTable input sets the DB table for storing the current DB
// migration state. If empty, then it's set to a default value of
// "schema_migrations". The named DB table will be automatically created unless
// it already exists.
func ApplyMigration(ctx context.Context, driver Driver, dirFS fs.FS, forward bool, version, migrationsTable string) (err error) {
	migrationsTable = cmp.Or(migrationsTable, internal.DefaultMigrationsTableName)
	var (
		pathToFile string
		mig        *internal.Migration
	)

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
			dirFS:           dirFS,
			finishAtVersion: limit,
		}
		if toApply, ierr := finder.query(ctx, driver, migrationsTable); ierr != nil {
			err = fmt.Errorf("specified no version; error attempting to find one; %v", ierr)
			return
		} else if len(toApply) < 1 {
			err = fmt.Errorf("version %w", internal.ErrNotFound)
			return
		} else {
			version = toApply[0].Version.String()
		}
	}

	if pathToFile, err = figureOutBasename(dirFS, direction, version); err != nil {
		return
	}
	fn := internal.Filename(filepath.Clean(pathToFile))
	if mig, err = internal.ParseMigration(fn); err != nil {
		return
	}
	err = runMigration(ctx, driver, dirFS, pathToFile, mig, migrationsTable)
	return
}

func figureOutBasename(dirFS fs.FS, direction internal.Direction, version string) (f string, e error) {
	var filenames []string
	// glob as many filenames as possible that match the "version" segment, then
	// narrow it down from there.
	glob := internal.MakeFilename(version, internal.Indirection{}, "*")
	if filenames, e = fs.Glob(dirFS, string(glob)); e != nil {
		return
	}

	var directionNames []string
	switch direction {
	case internal.DirForward:
		directionNames = internal.ForwardDirections
	case internal.DirReverse:
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

// runMigration executes a migration against the database. The input, pathToFile
// should be relative to the current working directory.
func runMigration(ctx context.Context, driver Driver, dir fs.FS, pathToFile string, mig *internal.Migration, migrationsTable string) (err error) {
	var data []byte
	if data, err = fs.ReadFile(dir, filepath.Clean(pathToFile)); err != nil {
		return
	}
	gerund := "migrating"
	if mig.Indirection.Value == internal.DirReverse {
		gerund = "rolling back"
	}

	lgr := slog.With(slog.String("path_to_file", pathToFile), slog.String("version", mig.Version.String()))
	lgr.Info(gerund + " ...")
	startTime := time.Now()

	if err = driver.Execute(ctx, string(data)); err != nil {
		err = fmt.Errorf("%w; path_to_file: %s; %w", internal.ErrExecutingMigration, pathToFile, err)
		lgr.Error("executing migration", slog.Any("error", err), makeDurationMSAttr(startTime))
		return
	}
	if err = driver.CreateSchemaMigrationsTable(ctx, migrationsTable); err != nil {
		lgr.Error("creating schema migrations table", slog.Any("error", err), makeDurationMSAttr(startTime))
		return
	}
	err = driver.UpdateSchemaMigrations(
		ctx,
		migrationsTable,
		mig.Indirection.Value == internal.DirForward,
		mig.Version.String(),
		mig.Label,
	)
	if err != nil {
		lgr.Error("updating schema migrations table", slog.Any("error", err), makeDurationMSAttr(startTime))
	} else {
		lgr.Info("ok", makeDurationMSAttr(startTime))
	}
	return
}

// makeDurationMSAttr calculates how much time, in milliseconds, has transpired
// since startedAt and returns a slog.KindInt64 attr with the key duration_ms.
func makeDurationMSAttr(startedAt time.Time) slog.Attr {
	dur := time.Since(startedAt)
	return slog.Int64("duration_ms", dur.Milliseconds())
}

// Info writes status of migrations to w in formats json or tsv.
//
// The migrationsTable input sets the DB table for storing the current DB
// migration state. If empty, then it's set to a default value of
// "schema_migrations". Unlike other functions that use the DB table to check
// the migration state, this function does not create a new table, nor does it
// have the need to.
func Info(ctx context.Context, driver Driver, directory fs.FS, forward bool, finishAtVersion string, w io.Writer, format string, migrationsTable string) (err error) {
	migrationsTable = cmp.Or(migrationsTable, internal.DefaultMigrationsTableName)

	direction := internal.DirReverse
	if forward {
		direction = internal.DirForward
	}

	finder := migrationFinder{
		direction:       direction,
		dirFS:           directory,
		finishAtVersion: finishAtVersion,
		infoPrinter:     choosePrinter(format, w),
	}
	_, err = finder.query(ctx, driver, migrationsTable)
	return
}

func choosePrinter(format string, w io.Writer) (out internal.InfoPrinter) {
	if format == "json" {
		out = internal.NewJSON(w)
		return
	}

	if format != "tsv" && format != "" {
		slog.Warn("unknown format, defaulting to tsv", slog.String("format", format))
	}
	out = internal.NewTSV(w)
	return
}

// Init creates a configuration file at pathToFile unless it already exists.
func Init(pathToFile string) (err error) {
	_, err = os.Stat(pathToFile)
	if err == nil {
		slog.Info("config file already present", slog.String("path_to_file", pathToFile))
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
	dirFS           fs.FS
	finishAtVersion string
	infoPrinter     internal.InfoPrinter
}

// query returns a list of Migrations to apply.
func (m *migrationFinder) query(ctx context.Context, driver Driver, migrationsTable string) (out []*internal.Migration, err error) {
	lgr := slog.With(slog.String("func", "(*migrationFinder).query"))

	available, err := m.available()
	if err != nil {
		return
	}
	lgr.Debug("found available migrations",
		slog.Int("count", len(available)),
		slog.Any("available", internal.Migrations(available)),
	)
	applied, err := scanAppliedVersions(ctx, driver, migrationsTable)
	if errors.Is(err, ErrSchemaMigrationsDoesNotExist) {
		// The next invocation of CreateSchemaMigrationsTable should fix this.
		// We can continue with zero value for now.
		slog.Info(
			"no migrations applied yet, continuing...",
			slog.Any("message", err), slog.String("migrations_table", migrationsTable),
		)
	} else if err != nil {
		return
	}
	lgr.Debug("scanned applied migrations",
		slog.Int("count", len(applied)),
		slog.Any("applied", internal.Migrations(applied)),
	)
	if m.infoPrinter != nil {
		if err = printMigrations(m.infoPrinter, "up", applied); err != nil {
			return
		}
	}
	lgr.Debug("about to filter migrations", slog.Group("migration_finder",
		slog.String("direction", m.direction.String()),
		slog.String("finish_at_version", m.finishAtVersion),
	))

	toApply, err := m.filter(applied, available)
	if err != nil {
		return
	}
	lgr.Debug("filtered migrations toApply", slog.Group("to_apply",
		slog.Int("count", len(toApply)),
		slog.Any("vals", internal.Migrations(toApply)),
	))

	var useDefaultRollbackVersion bool
	if m.finishAtVersion == "" && m.direction == internal.DirForward {
		m.finishAtVersion = internal.MaxVersion
	} else if m.finishAtVersion == "" && m.direction == internal.DirReverse {
		if len(toApply) == 0 {
			return
		}
		useDefaultRollbackVersion = true
		m.finishAtVersion = toApply[0].Version.String()
	}
	var finish internal.Version
	if finish, err = internal.ParseVersion(m.finishAtVersion); err != nil {
		return
	}
	lgr.Debug("about to collect migrations to apply in a loop",
		slog.String("finish_at_version", finish.String()),
		slog.String("direction", m.direction.String()),
		slog.Bool("use_default_rollback_version", useDefaultRollbackVersion),
		slog.Int("num_to_apply", len(toApply)),
	)

	for i, mig := range toApply {
		version := mig.Version
		lgr.Debug("considering migration to apply", slog.Int("i", i), slog.String("version", version.String()))
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
		lgr.Debug("collected migration to apply", slog.Int("i", i), slog.String("version", version.String()))
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
func (m *migrationFinder) available() (out []*internal.Migration, err error) {
	dirEntries, err := fs.ReadDir(m.dirFS, ".")
	if err != nil {
		err = fmt.Errorf("reading directory entries: %w", err)
		return
	}
	if m.direction != internal.DirForward {
		slices.Reverse(dirEntries)
	}
	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		if dirEntry.IsDir() {
			slog.Info("searching for available migrations and found directory, skipping", slog.String("path", name))
			continue
		}

		mig, ierr := internal.ParseMigration(internal.Filename(name))
		if errors.Is(ierr, internal.ErrDataInvalid) {
			slog.Warn("parsing migration filename, skipping over this one", slog.String("filename", name), slog.String("error", ierr.Error()))
			continue
		} else if ierr != nil {
			err = ierr
			return
		}
		dir := mig.Indirection.Value
		if dir != m.direction {
			continue
		}
		out = append(out, mig)
	}
	return
}

func scanAppliedVersions(ctx context.Context, driver Driver, migrationsTable string) (out []*internal.Migration, err error) {
	var rows AppliedVersions
	if rows, err = driver.AppliedVersions(ctx, migrationsTable); err != nil {
		return
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			slog.Warn("closing rows from func scanAppliedVersions", slog.Any("error", cerr))
		}
	}()
	for rows.Next() {
		var version, label string
		var executedAt int64
		if err = rows.Scan(&version, &label, &executedAt); err != nil {
			return
		}

		ver, verr := internal.ParseVersion(version)
		if verr != nil {
			err = fmt.Errorf("%w; while scanning applied versions, parsing version (%v) from DB: %w", internal.ErrDataInvalid, version, verr)
			return
		}
		var executedAtTime time.Time
		if executedAt > 0 {
			executedAtTime = time.Unix(executedAt, 0).UTC()
		}
		out = append(out, &internal.Migration{
			Indirection: internal.Indirection{Value: internal.DirForward},
			Label:       label,
			Version:     ver,
			ExecutedAt:  executedAtTime,
		})
	}

	return
}

// filter compares lists of applied and available migrations, then selects a
// list of migrations to apply.
func (m *migrationFinder) filter(applied, available []*internal.Migration) (out []*internal.Migration, err error) {
	allVersions := make(map[int64]*internal.Migration)
	uniqueToApplied := make(map[int64]*internal.Migration)
	for _, mig := range applied {
		version := mig.Version.Value()
		uniqueToApplied[version] = mig
		allVersions[version] = mig
	}
	uniqueToAvailable := make(map[int64]*internal.Migration)
	for _, mig := range available {
		version := mig.Version.Value()
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
				var mut *internal.Migration
				indirection := internal.Indirection{
					Value: internal.DirReverse,
					Label: "reverse", // need to have something here, it gets restored later.
				}
				mut, err = newMigration(mig.Version.String(), indirection, mig.Label)
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
					if mig.Indirection.Label == fwd {
						indirection.Label = internal.ReverseDirections[i]
						mut, err = newMigration(mig.Version.String(), indirection, mig.Label)
						if err != nil {
							return
						}
						break
					}
				}
				if mut.Label == "" {
					err = fmt.Errorf(
						"direction.Label empty; direction.Value: %q, version: %v, label: %q",
						mut.Indirection, mut.Version.String(), mut.Label,
					)
					return
				}
				out = append(out, mut)
			}
		}
	}
	if m.direction == internal.DirForward {
		sort.Slice(out, func(i, j int) bool {
			return out[i].Version.Before(out[j].Version)
		})
	} else {
		sort.Slice(out, func(i, j int) bool {
			return out[j].Version.Before(out[i].Version)
		})
	}
	return
}

func newMigration(version string, ind internal.Indirection, label string) (out *internal.Migration, err error) {
	fn := internal.MakeFilename(version, ind, label)
	out, err = internal.ParseMigration(fn)
	return
}

func printMigrations(p internal.InfoPrinter, state string, migrations []*internal.Migration) (err error) {
	for i, mig := range migrations {
		if err = p.PrintInfo(state, *mig); err != nil {
			err = fmt.Errorf("%w; item %d", err, i)
			return
		}
	}
	return
}
