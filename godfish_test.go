package godfish_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/compat"
	"github.com/rafaelespinoza/godfish/internal/stub"
	"github.com/rafaelespinoza/godfish/testdata"

	st "github.com/rafaelespinoza/slogtesting"
)

func TestCreateMigrationFiles(t *testing.T) {
	tests := []struct {
		name   string
		create compat.CreateMigrationFunc
	}{
		{
			name: "Deprecated APIs",
			create: func(migrationName string, reversible bool, dirpath, fwdlabel, revlabel, ext string) error {
				return godfish.CreateMigrationFiles(migrationName, reversible, dirpath, fwdlabel, revlabel)
			},
		},
		{
			name: "Replacement APIs",
			create: func(migrationName string, reversible bool, dirpath, fwdlabel, revlabel, ext string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					ForwardLabel: fwdlabel,
					ReverseLabel: revlabel,
					FilenameExt:  ext,
				})
				return godfish.CreateMigrationFilesWith(migrationName, reversible, dirpath, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testCreateMigrationFiles(t, test.create)
		})
	}
}

func testCreateMigrationFiles(t *testing.T, create compat.CreateMigrationFunc) {
	t.Run("err", func(t *testing.T) {
		err := godfish.CreateMigrationFiles("err_test", true, t.TempDir(), "bad", "bad2")
		if err == nil {
			t.Fatal(err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		testdir := t.TempDir()
		err := create("err_test", true, testdir, "", "", "")
		if err != nil {
			t.Fatal(err)
		}

		entries, err := os.ReadDir(testdir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("wrong number of entries, got %d, expected %d", len(entries), 2)
		}

		for i, direction := range []string{"forward", "reverse"} {
			got := entries[i].Name()
			if !strings.HasPrefix(got, direction) {
				t.Errorf("expected filename, %q, to have prefix %q", got, direction)
			}
			// Inspect the label + filename extension part.
			if !strings.HasSuffix(got, "err_test.sql") {
				t.Errorf("expected filename, %q, to have suffix %q", got, "err_test.sql")
			}
		}
	})
}

func TestCreateMigrationFilesWith(t *testing.T) {
	t.Run("error - validation", func(t *testing.T) {

		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithForwardLabel",
				opt:  godfish.WithForwardLabel("bad"),
			},
			{
				name: "WithReverseLabel",
				opt:  godfish.WithReverseLabel("bad"),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				testdir := t.TempDir()
				migrationName := strings.ReplaceAll(t.Name(), "/", "")
				err := godfish.CreateMigrationFilesWith(migrationName, false, testdir, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				// should contain some substring that suggests correct values.
				const looseExpectation = "should be one of"
				if m := err.Error(); !strings.Contains(m, looseExpectation) {
					t.Errorf("expected for error message (%q) to contain %q", m, looseExpectation)
				}
			})
		}
	})

	t.Run("called with no options, does not blow up", func(t *testing.T) {
		testdir := t.TempDir()
		err := godfish.CreateMigrationFilesWith("this_is_fine", true, testdir)
		if err != nil {
			t.Fatal(err)
		}

		entries, err := os.ReadDir(testdir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("wrong number of entries, got %d, expected %d", len(entries), 2)
		}

		for i, direction := range []string{"forward", "reverse"} {
			got := entries[i].Name()
			if !strings.HasPrefix(got, direction) {
				t.Errorf("expected filename, %q, to have prefix %q", got, direction)
			}
			// Inspect the label + filename extension part.
			if !strings.HasSuffix(got, "this_is_fine.sql") {
				t.Errorf("expected filename, %q, to have suffix %q", got, "this_is_fine.sql")
			}
		}
	})

	t.Run("called with options, it creates the files", func(t *testing.T) {
		testdir := t.TempDir()
		err := godfish.CreateMigrationFilesWith("everything_is_fine", true, testdir,
			godfish.WithForwardLabel("migrate"),
			godfish.WithReverseLabel("rollback"),
			godfish.WithFilenameExtension(".cql"),
		)
		if err != nil {
			t.Fatal(err)
		}

		entries, err := os.ReadDir(testdir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("wrong number of entries, got %d, expected %d", len(entries), 2)
		}

		for i, direction := range []string{"migrate", "rollback"} {
			got := entries[i].Name()
			if !strings.HasPrefix(got, direction) {
				t.Errorf("expected filename, %q, to have prefix %q", got, direction)
			}
			// Inspect the label + filename extension part.
			if !strings.HasSuffix(got, "everything_is_fine.cql") {
				t.Errorf("expected filename, %q, to have suffix %q", got, "everything_is_fine.cql")
			}
		}
	})
}

func TestMigrate(t *testing.T) {
	tests := []struct {
		name string
		up   compat.MigrateFunc
		down compat.RollbackFunc
	}{
		{
			name: "Deprecated APIs",
			up: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.Migrate(ctx, d, fsys, true, v, tbl)
			},
			down: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.Migrate(ctx, d, fsys, false, v, tbl)
			},
		},
		{
			name: "Replacement APIs",
			up: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.MigrateWith(ctx, d, fsys, opts...)
			},
			down: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.RollbackWith(ctx, d, fsys, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testMigrate(t, test.up, test.down)
		})
	}
}

func testMigrate(t *testing.T, up compat.MigrateFunc, down compat.RollbackFunc) {
	testUpDown(t, up, down)
	// There are more detailed tests in the internal/test package.
	dirFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("schema migrations table does not exist", func(t *testing.T) {
		// Check that when the table does not exist, in the happy path, the
		// "database" will handle the error by creating the table and updating it.
		var updateCalls int
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, driver.ErrSchemaMigrationsDoesNotExist
			},
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				updateCalls++
				return nil
			},
		}
		err := up(t.Context(), driver, dirFS, "2345", "test")
		if err != nil {
			t.Fatal(err)
		}
		if expNumCalls := 2; updateCalls != expNumCalls {
			t.Errorf("number of calls to UpdateSchemaMigrations; got %d, expected %d", updateCalls, expNumCalls)
		}
	})

	t.Run("handles alternate filename extensions", func(t *testing.T) {
		var appliedVersionsCalls, executeCalls int
		var createSchemaMigrationsCalls, updateSchemaMigrationsCall int
		driver := stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				appliedVersionsCalls++
				switch appliedVersionsCalls {
				case 1:
					return makeScanApplied(t)(ctx, migrationsTable)
				case 2:
					return makeScanApplied(t, "1234", "2345", "3456")(ctx, migrationsTable)
				default:
					t.Fatalf("unexpected number of calls to AppliedVersions; got %d, expected [1,2]", appliedVersionsCalls)
					return nil, nil
				}
			},
			ExecuteFn: func(ctx context.Context, q string, a ...any) error {
				executeCalls++
				return nil
			},
			CreateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				createSchemaMigrationsCalls++
				return nil
			},
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				updateSchemaMigrationsCall++
				return nil
			},
		}
		dirFS := stub.FS{
			FS: fstest.MapFS{
				"forward-1234-a.cql": &fstest.MapFile{Mode: 0x600},
				"forward-2345-b.cql": &fstest.MapFile{Mode: 0x600},
				"forward-3456-c.cql": &fstest.MapFile{Mode: 0x600},
				"reverse-1234-a.cql": &fstest.MapFile{Mode: 0x600},
				"reverse-2345-b.cql": &fstest.MapFile{Mode: 0x600},
				"reverse-3456-c.cql": &fstest.MapFile{Mode: 0x600},
			},
		}
		if err := up(t.Context(), &driver, dirFS, "", ""); err != nil {
			t.Fatal(err)
		}
		const expNumCallsAfterUp = 3
		if got := executeCalls; got != expNumCallsAfterUp {
			t.Errorf("wrong number of calls to Execute; got %d, expected %d", got, expNumCallsAfterUp)
		}
		if got := createSchemaMigrationsCalls; got != expNumCallsAfterUp {
			t.Errorf("wrong number of calls to CreateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterUp)
		}
		if got := updateSchemaMigrationsCall; got != expNumCallsAfterUp {
			t.Errorf("wrong number of calls to UpdateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterUp)
		}

		if err := down(t.Context(), &driver, dirFS, "", ""); err != nil {
			t.Fatal(err)
		}
		const expNumCallsAfterDown = 6
		if got := executeCalls; got != expNumCallsAfterDown {
			t.Errorf("wrong number of calls to Execute; got %d, expected %d", got, expNumCallsAfterDown)
		}
		if got := createSchemaMigrationsCalls; got != expNumCallsAfterDown {
			t.Errorf("wrong number of calls to CreateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterDown)
		}
		if got := updateSchemaMigrationsCall; got != expNumCallsAfterDown {
			t.Errorf("wrong number of calls to UpdateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterDown)
		}
	})
}

func TestMigrateWith(t *testing.T) {
	t.Run("error - non-zero value required", func(t *testing.T) {
		dirFS, err := fs.Sub(testdata.Migrations, "default")
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithTargetVersion empty string",
				opt:  godfish.WithTargetVersion(""),
			},
			{
				name: "WithMigrationsTable empty string",
				opt:  godfish.WithMigrationsTable(""),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := makeNoCallDriver(t)
				err := godfish.MigrateWith(t.Context(), driver, dirFS, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if m := err.Error(); !strings.Contains(m, "zero value") {
					t.Errorf("expected for error message (%q) to contain %q", m, "zero value")
				}
			})
		}
	})

	t.Run("called with no options, does not blow up", func(t *testing.T) {
		testBasicOperationWithoutOpts(t, []string{"1234"}, 2, godfish.MigrateWith)
	})
}

func TestRollbackWith(t *testing.T) {
	t.Run("error - non-zero value required", func(t *testing.T) {
		dirFS, err := fs.Sub(testdata.Migrations, "default")
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithTargetVersion empty string",
				opt:  godfish.WithTargetVersion(""),
			},
			{
				name: "WithMigrationsTable empty string",
				opt:  godfish.WithMigrationsTable(""),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := makeNoCallDriver(t)
				err := godfish.RollbackWith(t.Context(), driver, dirFS, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if m := err.Error(); !strings.Contains(m, "zero value") {
					t.Errorf("expected for error message (%q) to contain %q", m, "zero value")
				}
			})
		}
	})

	t.Run("called with no options, does not blow up", func(t *testing.T) {
		testBasicOperationWithoutOpts(t, []string{"1234", "2345", "3456"}, 3, godfish.RollbackWith)
	})
}

func TestApplyMigration(t *testing.T) {
	tests := []struct {
		name string
		up   compat.MigrateFunc
		down compat.RollbackFunc
	}{
		{
			name: "Deprecated APIs",
			up: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.ApplyMigration(ctx, d, fsys, true, v, tbl)
			},
			down: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.ApplyMigration(ctx, d, fsys, false, v, tbl)
			},
		},
		{
			name: "Replacement APIs",
			up: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.ApplyMigrationWith(ctx, d, fsys, opts...)
			},
			down: func(ctx context.Context, d driver.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.ApplyRollbackWith(ctx, d, fsys, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testApplyMigration(t, test.up, test.down)
		})
	}
}

func testApplyMigration(t *testing.T, up compat.MigrateFunc, down compat.RollbackFunc) {
	testUpDown(t, up, down)

	okFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("version empty, not found", func(t *testing.T) {
		driver := stub.Double{
			AppliedVersionsFn: makeScanApplied(t),
		}
		fsys := fstest.MapFS{}
		err := up(t.Context(), &driver, fsys, "", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		if !errors.Is(err, internal.ErrNotFound) {
			t.Errorf("expected for error (%v) to be %v", err, internal.ErrNotFound)
		}
	})

	t.Run("version specified, not found", func(t *testing.T) {
		driver := stub.Double{}
		err := up(t.Context(), &driver, okFS, "1111", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		if !errors.Is(err, internal.ErrNotFound) {
			t.Errorf("expected for error (%v) to be %v", err, internal.ErrNotFound)
		}
	})

	t.Run("version specified, found", func(t *testing.T) {
		var numCallsToAppliedVersions int
		driver := stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				defer func() { numCallsToAppliedVersions++ }()
				var migs []internal.Migration
				switch numCallsToAppliedVersions {
				case 0:
					migs = makeMigrations(t)
				case 1:
					migs = makeMigrations(t, "1234")
				default:
					t.Fatal("too many calls to AppliedVersions")
				}
				return stub.NewAppliedVersions(migs...), nil
			},
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: makeUpdatSchemaMigrationsFn(nil),
		}
		err := up(t.Context(), &driver, okFS, "1234", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = down(t.Context(), &driver, okFS, "1234", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error too many migration files with same version", func(t *testing.T) {
		driver := makeNoCallDriver(t)
		badFS := fstest.MapFS{
			"forward-1234-a": &fstest.MapFile{Mode: 0x640},
			"forward-1234-b": &fstest.MapFile{Mode: 0x640},
		}
		err := up(t.Context(), driver, badFS, "1234", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		const looseExpectation = "too many migration files with matching"
		if m := err.Error(); !strings.Contains(m, looseExpectation) {
			t.Errorf("expected for error (%v) to contain %q", err, looseExpectation)
		}
	})

	t.Run("schema migrations table does not exist", func(t *testing.T) {
		// Check that when the table does not exist, in the happy path, the
		// "database" will handle the error by creating the table and updating it.
		var updateCalls int
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, driver.ErrSchemaMigrationsDoesNotExist
			},
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				updateCalls++
				return nil
			},
		}
		err := up(t.Context(), driver, okFS, "2345", "test")
		if err != nil {
			t.Fatal(err)
		}
		if expNumCalls := 1; updateCalls != expNumCalls {
			t.Errorf("number of calls to UpdateSchemaMigrations; got %d, expected %d", updateCalls, expNumCalls)
		}
	})

	t.Run("rollback - error no migrations found", func(t *testing.T) {
		var calledExec, calledUpdate bool
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return stub.NewAppliedVersions(), nil
			},
			ExecuteFn: func(ctx context.Context, q string, a ...any) error {
				calledExec = true
				return nil
			},
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				calledUpdate = true
				return nil
			},
		}
		err := down(t.Context(), driver, okFS, "", "test")
		expErr := internal.ErrNotFound
		if !errors.Is(err, expErr) {
			t.Fatalf("expected error (%v) to be %v", err, expErr)
		}
		if !strings.Contains(err.Error(), "version") {
			t.Errorf("expected error (%v) to contain %q", err.Error(), "version")
		}
		if calledExec {
			t.Errorf("did not expect to call Execute")
		}
		if calledUpdate {
			t.Errorf("did not expect to call UpdateSchemaMigrations")
		}
	})

	t.Run("handles alternate filename extensions", func(t *testing.T) {
		var appliedVersionsCalls, executeCalls int
		var createSchemaMigrationsCalls, updateSchemaMigrationsCall int
		driver := stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				appliedVersionsCalls++
				switch appliedVersionsCalls {
				case 1:
					return makeScanApplied(t)(ctx, migrationsTable)
				case 2:
					return makeScanApplied(t, "1234")(ctx, migrationsTable)
				default:
					t.Fatalf("unexpected number of calls to AppliedVersions; got %d, expected [1,2]", appliedVersionsCalls)
					return nil, nil
				}
			},
			ExecuteFn: func(ctx context.Context, q string, a ...any) error {
				executeCalls++
				return nil
			},
			CreateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				createSchemaMigrationsCalls++
				return nil
			},
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				updateSchemaMigrationsCall++
				return nil
			},
		}
		dirFS := stub.FS{
			FS: fstest.MapFS{
				"forward-1234-a.cql": &fstest.MapFile{Mode: 0x600},
				"forward-2345-b.cql": &fstest.MapFile{Mode: 0x600},
				"forward-3456-c.cql": &fstest.MapFile{Mode: 0x600},
				"reverse-1234-a.cql": &fstest.MapFile{Mode: 0x600},
				"reverse-2345-b.cql": &fstest.MapFile{Mode: 0x600},
				"reverse-3456-c.cql": &fstest.MapFile{Mode: 0x600},
			},
		}

		if err := up(t.Context(), &driver, dirFS, "", ""); err != nil {
			t.Fatal(err)
		}
		const expNumCallsAfterUp = 1
		if got := executeCalls; got != expNumCallsAfterUp {
			t.Errorf("wrong number of calls to Execute; got %d, expected %d", got, expNumCallsAfterUp)
		}
		if got := createSchemaMigrationsCalls; got != expNumCallsAfterUp {
			t.Errorf("wrong number of calls to CreateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterUp)
		}
		if got := updateSchemaMigrationsCall; got != expNumCallsAfterUp {
			t.Errorf("wrong number of calls to UpdateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterUp)
		}

		if err := down(t.Context(), &driver, dirFS, "", ""); err != nil {
			t.Fatal(err)
		}
		const expNumCallsAfterDown = 2
		if got := executeCalls; got != expNumCallsAfterDown {
			t.Errorf("wrong number of calls to Execute; got %d, expected %d", got, expNumCallsAfterDown)
		}
		if got := createSchemaMigrationsCalls; got != expNumCallsAfterDown {
			t.Errorf("wrong number of calls to CreateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterDown)
		}
		if got := updateSchemaMigrationsCall; got != expNumCallsAfterDown {
			t.Errorf("wrong number of calls to UpdateSchemaMigrations; got %d, expected %d", got, expNumCallsAfterDown)
		}
	})
}

// testUpDown is for testing ApplyMigration, ApplyMigrationWith,
// ApplyRollbackWith, Migrate, MigrateWith, RollbackWith.
func testUpDown(t *testing.T, up compat.MigrateFunc, down compat.RollbackFunc) {
	okFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("all the way up and down", func(t *testing.T) {
		driver := stub.Double{
			AppliedVersionsFn:        makeScanApplied(t, "1234"),
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: makeUpdatSchemaMigrationsFn(nil),
		}
		var err error
		if err = up(t.Context(), &driver, okFS, "", ""); err != nil {
			t.Fatal(err)
		}

		if err = down(t.Context(), &driver, okFS, "", ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("bad version", func(t *testing.T) {
		driver := stub.Double{
			AppliedVersionsFn:        makeScanApplied(t, "1234", "2345", "3456"),
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: makeUpdatSchemaMigrationsFn(nil),
		}
		err := up(t.Context(), &driver, okFS, "bad", "")
		if err == nil {
			t.Fatal("expected error")
		}
		t.Log(err)
		if m := err.Error(); !strings.Contains(m, "version") {
			t.Errorf("expected for error (%v) to mention %q", m, "version")
		}
	})

	t.Run("error reading from FS directory", func(t *testing.T) {
		driver := makeNoCallDriver(t)
		badFS := stub.FS{
			FS: fstest.MapFS{"forward-1234-no_read_permissions": &fstest.MapFile{Mode: 0x000}},
			ReadDirFn: func(n string) ([]fs.DirEntry, error) {
				return nil, errors.New("I've never seen that file in my life")
			},
		}
		err := up(t.Context(), driver, &badFS, "1234", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		const looseExpectation = "reading directory entries"
		if m := err.Error(); !strings.Contains(m, looseExpectation) {
			t.Errorf("expected for error (%v) to contain %q", err, looseExpectation)
		}
	})

	t.Run("error reading from found file", func(t *testing.T) {
		driver := makeNoCallDriver(t)
		// The only allowed Driver method in this case is AppliedVersions.
		// Check that an error reading the file stops before calling Execute, and
		// the other methods.
		driver.AppliedVersionsFn = makeScanApplied(t, "1234")

		badFS := stub.FS{
			FS: fstest.MapFS{
				"forward-1234-a": &fstest.MapFile{Mode: 0x600},
				"forward-2345-b": &fstest.MapFile{Mode: 0x600},
			},
			ReadFileFn: func(name string) ([]byte, error) {
				if name != "forward-2345-b" {
					t.Fatalf("read unexpected file %q, should read %q", name, "forward-2345-b")
				}
				return nil, errors.New("OOF")
			},
		}
		err := up(t.Context(), driver, &badFS, "2345", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		const looseExpectation = "reading file"
		if m := err.Error(); !strings.Contains(m, looseExpectation) {
			t.Errorf("expected for error (%v) to contain %q", err, looseExpectation)
		}
	})

	t.Run("other error scanning for migrations", func(t *testing.T) {
		oof := errors.New("oof")
		var calledUpdate bool
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, oof
			},
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				calledUpdate = true
				return nil
			},
		}

		// TODO: also add test that passes a non-zero version
		err := up(t.Context(), driver, okFS, "", "test")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
		if calledUpdate {
			t.Errorf("did not expect to call UpdateSchemaMigrations")
		}
	})

	t.Run("error executing migration", func(t *testing.T) {
		var calledUpdateFn bool
		driver := &stub.Double{
			AppliedVersionsFn: makeScanApplied(t, "1234"),
			ExecuteFn:         makeExecuteFn(errors.New("OOF")),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				calledUpdateFn = true
				return nil
			},
		}
		err := up(t.Context(), driver, okFS, "2345", "test")
		expErr := internal.ErrExecutingMigration
		if !errors.Is(err, expErr) {
			t.Errorf("expected error (%v) to be %v", err, expErr)
		}
		if calledUpdateFn {
			t.Errorf("did not expect to call the update method")
		}
	})

	t.Run("error creating migrations table", func(t *testing.T) {
		oof := errors.New("oof")
		var calledUpdateFn bool
		driver := &stub.Double{
			AppliedVersionsFn:        makeScanApplied(t),
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(oof),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				calledUpdateFn = true
				return nil
			},
		}
		err := up(t.Context(), driver, okFS, "2345", "test")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
		if calledUpdateFn {
			t.Errorf("did not expect to call the update method")
		}
	})

	t.Run("error updating migrations table", func(t *testing.T) {
		oof := errors.New("oof")
		driver := &stub.Double{
			AppliedVersionsFn:        makeScanApplied(t, "1234"),
			ExecuteFn:                makeExecuteFn(nil),
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: func(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
				return oof
			},
		}
		err := up(t.Context(), driver, okFS, "2345", "test")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
	})

	t.Run("skips over directories", func(t *testing.T) {
		var numExecCalls int
		driver := &stub.Double{
			AppliedVersionsFn: makeScanApplied(t, "1234"),
			ExecuteFn: func(ctx context.Context, q string, a ...any) error {
				numExecCalls++
				return nil
			},
			CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
			UpdateSchemaMigrationsFn: makeUpdatSchemaMigrationsFn(nil),
		}
		dirFS := fstest.MapFS{
			// this "directory" is named like a migrations file, but should be ignored.
			"forward-2345-directory": &fstest.MapFile{Mode: fs.ModeDir},
			"forward-1234-a.sql":     &fstest.MapFile{Mode: 0x640},
			"forward-2345-b.sql":     &fstest.MapFile{Mode: 0x640},
		}
		err := up(t.Context(), driver, dirFS, "2345", "")
		if err != nil {
			t.Fatal(err)
		}
		const expNumCalls = 1
		if numExecCalls != expNumCalls {
			t.Errorf("wrong number of method calls; got %d, expected %d", numExecCalls, expNumCalls)
		}
	})

	t.Run("outputs some logs", func(t *testing.T) {
		t.Setenv(internal.DSNKey, t.Name())

		run := func(h slog.Handler) error {
			originalDefaults := slog.Default()
			t.Cleanup(func() { slog.SetDefault(originalDefaults) })
			slog.SetDefault(slog.New(h))

			driver := stub.Double{
				AppliedVersionsFn:        makeScanApplied(t, "1234"),
				ExecuteFn:                makeExecuteFn(nil),
				CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
				UpdateSchemaMigrationsFn: makeUpdatSchemaMigrationsFn(nil),
			}
			testFS := fstest.MapFS{"forward-0000-a.sql": &fstest.MapFile{}}

			return godfish.ApplyMigration(t.Context(), &driver, testFS, true, "0000", "")
		}
		logRecords, err := st.CaptureRecords(nil, run)
		if err != nil {
			t.Fatal(err)
		}
		if len(logRecords) != 2 {
			t.Fatalf("wrong number of logging records; got %d, expected %d", len(logRecords), 2)
		}

		attrs := st.GetRecordAttrs(logRecords[1])
		check := st.HasKey("duration_ms")
		if err := check(attrs); err != nil {
			t.Error(err)
		}
	})
}

func TestApplyMigrationWith(t *testing.T) {
	t.Run("error - non-zero value required", func(t *testing.T) {
		dirFS, err := fs.Sub(testdata.Migrations, "default")
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithTargetVersion empty string",
				opt:  godfish.WithTargetVersion(""),
			},
			{
				name: "WithMigrationsTable empty string",
				opt:  godfish.WithMigrationsTable(""),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := makeNoCallDriver(t)
				err := godfish.ApplyMigrationWith(t.Context(), driver, dirFS, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if m := err.Error(); !strings.Contains(m, "zero value") {
					t.Errorf("expected for error message (%q) to contain %q", m, "zero value")
				}
			})
		}
	})

	t.Run("called with no options, does not blow up", func(t *testing.T) {
		testBasicOperationWithoutOpts(t, []string{"1234"}, 1, godfish.ApplyRollbackWith)
	})
}

func TestApplyRollbackWith(t *testing.T) {
	t.Run("error - non-zero value required", func(t *testing.T) {
		dirFS, err := fs.Sub(testdata.Migrations, "default")
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithTargetVersion empty string",
				opt:  godfish.WithTargetVersion(""),
			},
			{
				name: "WithMigrationsTable empty string",
				opt:  godfish.WithMigrationsTable(""),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := makeNoCallDriver(t)
				err := godfish.ApplyRollbackWith(t.Context(), driver, dirFS, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if m := err.Error(); !strings.Contains(m, "zero value") {
					t.Errorf("expected for error message (%q) to contain %q", m, "zero value")
				}
			})
		}
	})

	t.Run("called with no options, does not blow up", func(t *testing.T) {
		testBasicOperationWithoutOpts(t, []string{"1234", "2345", "3456"}, 1, godfish.ApplyRollbackWith)
	})
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name string
		info compat.InfoFunc
	}{
		{
			name: "Deprecated APIs",
			info: func(
				ctx context.Context,
				driver driver.Driver,
				dirFS fs.FS,
				fwd bool,
				version string,
				w io.Writer,
				format string,
				table string,
			) error {
				return godfish.Info(ctx, driver, dirFS, fwd, version, w, format, table)
			},
		},
		{
			name: "Replacement APIs",
			info: func(
				ctx context.Context,
				driver driver.Driver,
				dirFS fs.FS,
				fwd bool,
				version string,
				w io.Writer,
				format string,
				table string,
			) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					Format:          format,
					MigrationsTable: table,
					Writer:          w,
				})
				return godfish.InfoWith(ctx, driver, dirFS, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testInfo(t, test.info)
		})
	}
}

func testInfo(t *testing.T, info compat.InfoFunc) {
	okFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unknown format does not error out", func(t *testing.T) {
		driver := stub.Double{
			AppliedVersionsFn: makeScanApplied(t, "1234", "2345", "3456"),
		}
		err := info(t.Context(), &driver, okFS, false, "", t.Output(), "tea_ess_vee", "")
		if err != nil {
			t.Fatalf("unexpected error, %v", err)
		}
	})

	t.Run("scanned migrations have empty label", func(t *testing.T) {
		// Check that applied migrations inserted prior to a schema upgrade can have
		// there Label fields inferred by inspecting matching filenames.

		driver := &stub.Double{
			AppliedVersionsFn: makeScanApplied(t, "1234", "2345", "3456"),
		}

		var buf bytes.Buffer
		err := info(t.Context(), driver, okFS, true, "3456", &buf, "json", "")
		if err != nil {
			t.Fatalf("unexpected error, %v", err)
		}

		expLabels := []string{"alpha", "bravo", "charlie"}
		lines := bytes.TrimSpace(buf.Bytes())
		for i, line := range bytes.Split(lines, []byte{'\n'}) {
			var item struct{ Label string }
			if err = json.Unmarshal(line, &item); err != nil {
				t.Fatal(err)
			}
			expLabel := expLabels[i]
			if got := item.Label; got != expLabel {
				t.Errorf("item %d, got %q, expected %q", i, got, expLabel)
			}
		}
	})

	t.Run("error scanning migraions", func(t *testing.T) {
		oof := errors.New("OOF")
		driver := stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, oof
			},
		}
		err := info(t.Context(), &driver, okFS, true, "", t.Output(), "tsv", "")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
	})
}

func TestInfoWith(t *testing.T) {
	t.Run("error - non-zero value required", func(t *testing.T) {
		dirFS, err := fs.Sub(testdata.Migrations, "default")
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithWriter nil writer",
				opt:  godfish.WithWriter(nil),
			},
			{
				name: "WithFormat empty string",
				opt:  godfish.WithFormat(""),
			},
			{
				name: "WithMigrationsTable empty string",
				opt:  godfish.WithMigrationsTable(""),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := makeNoCallDriver(t)
				err := godfish.InfoWith(t.Context(), driver, dirFS, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if m := err.Error(); !strings.Contains(m, "zero value") {
					t.Errorf("expected for error message (%q) to contain %q", m, "zero value")
				}
			})
		}
	})

	t.Run("called with no options, does not blow up", func(t *testing.T) {
		testBasicOperationWithoutOpts(t, []string{"1234", "2345", "3456"}, 0, godfish.InfoWith)
	})
}

func TestInit(t *testing.T) {
	var err error
	testOutputDir := t.TempDir()

	pathToFile := filepath.Clean(filepath.Join(testOutputDir, "config.json"))

	// setup: file should not exist at first
	if _, err = os.Stat(pathToFile); !os.IsNotExist(err) {
		t.Fatalf("setup error; file at %q should not exist", pathToFile)
	}

	// test 1: file created with this shape
	if err = godfish.Init(pathToFile); err != nil {
		t.Fatalf("something else is wrong with setup; %v", err)
	}
	var conf internal.Config
	if data, err := os.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf); err != nil {
		t.Fatal(err)
	}
	conf.PathToFiles = testOutputDir + "/bar"

	// test2: write data and make sure it's not overwritten after calling Init
	if data, err := json.MarshalIndent(conf, "", "\t"); err != nil {
		t.Fatal(err)
	} else {
		err = os.WriteFile(pathToFile, append(data, byte('\n')), os.FileMode(0640))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := godfish.Init(pathToFile); err != nil {
		t.Fatal(err)
	}
	var conf2 internal.Config
	if data, err := os.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf2); err != nil {
		t.Fatal(err)
	}
	if conf2.PathToFiles != testOutputDir+"/bar" {
		t.Errorf(
			"expected conf.PathToFiles to be %q, got %q",
			"foo", conf2.PathToFiles,
		)
	}
}

func TestUpgradeSchemaMigrations(t *testing.T) {
	tests := []struct {
		name    string
		upgrade compat.UpgradeSchemaFunc
	}{
		{
			name: "Deprecated APIs",
			upgrade: func(ctx context.Context, driver driver.Driver, table string) error {
				return godfish.UpgradeSchemaMigrations(t.Context(), driver, table)
			},
		},
		{
			name: "Replacement APIs",
			upgrade: func(ctx context.Context, driver driver.Driver, table string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{MigrationsTable: table})
				return godfish.UpgradeSchemaMigrationsWith(t.Context(), driver, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testUpgradeSchemaMigrations(t, test.upgrade)
		})
	}
}

func testUpgradeSchemaMigrations(t *testing.T, upgrade compat.UpgradeSchemaFunc) {
	t.Run("error running UpgradeSchemaMigrations", func(t *testing.T) {
		oof := errors.New("OOF")
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, driver.ErrSchemaMigrationsMissingColumns
			},
			UpgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				return oof
			},
		}
		err := upgrade(t.Context(), driver, "test")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
	})

	t.Run("ok", func(t *testing.T) {
		var calledUpgradeFn bool
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, driver.ErrSchemaMigrationsMissingColumns
			},
			UpgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				calledUpgradeFn = true
				return nil
			},
		}
		err := upgrade(t.Context(), driver, "test")
		if err != nil {
			t.Fatal(err)
		}
		if !calledUpgradeFn {
			t.Errorf("expected to call the upgrade method")
		}
	})

	t.Run("ok - upgrade already done", func(t *testing.T) {
		var calledUpgradeFn bool
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				migs := makeMigrations(t, "1234")
				return stub.NewAppliedVersions(migs...), nil
			},
			UpgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				calledUpgradeFn = true
				return nil
			},
		}
		err := upgrade(t.Context(), driver, "test")
		if err != nil {
			t.Fatal(err)
		}
		if calledUpgradeFn {
			t.Errorf("did not expect to call the upgrade method")
		}
	})

	t.Run("ErrSchemaMigrationsDoesNotExist", func(t *testing.T) {
		var calledUpgradeFn bool
		d := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, driver.ErrSchemaMigrationsDoesNotExist
			},
			UpgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				calledUpgradeFn = true
				return nil
			},
		}
		err := upgrade(t.Context(), d, "test")
		expErr := driver.ErrSchemaMigrationsDoesNotExist
		if !errors.Is(err, expErr) {
			t.Errorf("expected error (%v) to be %v", err, expErr)
		}
		if calledUpgradeFn {
			t.Errorf("did not expect to call the upgrade method")
		}
	})

	t.Run("other error while scanning for applied migrations", func(t *testing.T) {
		var calledUpgradeFn bool
		oof := errors.New("OOF")
		driver := &stub.Double{
			AppliedVersionsFn: func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
				return nil, oof
			},
			UpgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				calledUpgradeFn = true
				return nil
			},
		}
		err := upgrade(t.Context(), driver, "test")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
		if calledUpgradeFn {
			t.Errorf("did not expect to call the upgrade method")
		}
	})
}

func TestUpgradeSchemaMigrationsWith(t *testing.T) {
	t.Run("error - non-zero value required", func(t *testing.T) {
		tests := []struct {
			name string
			opt  godfish.Opter
		}{
			{
				name: "WithMigrationsTable empty string",
				opt:  godfish.WithMigrationsTable(""),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := makeNoCallDriver(t)
				err := godfish.UpgradeSchemaMigrationsWith(t.Context(), driver, test.opt)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if m := err.Error(); !strings.Contains(m, "zero value") {
					t.Errorf("expected for error message (%q) to contain %q", m, "zero value")
				}
			})
		}
	})
}

func makeExecuteFn(e error) func(context.Context, string, ...any) error {
	return func(context.Context, string, ...any) error { return e }
}

func makeCreateSchemaMigrationsFn(e error) func(context.Context, string) error {
	return func(context.Context, string) error { return e }
}

func makeUpdatSchemaMigrationsFn(e error) func(context.Context, string, bool, string, string) error {
	return func(context.Context, string, bool, string, string) error {
		return e
	}
}

func makeScanApplied(t *testing.T, versions ...string) func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
	return func(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
		// The migrations in the "database" have empty Label fields.
		migs := makeMigrations(t, versions...)
		return stub.NewAppliedVersions(migs...), nil
	}
}

func makeMigrations(t *testing.T, versions ...string) []internal.Migration {
	migs := make([]internal.Migration, len(versions))
	for i, v := range versions {
		migs[i] = internal.Migration{
			Indirection: internal.Indirection{},
			Version:     mustParseVersion(t, v),
		}
	}
	return migs
}

func mustParseVersion(t *testing.T, v string) internal.Version {
	out, err := internal.ParseVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// makeNoCallDriver constructs a Driver that errors the test whenever a
// [driver.Driver] method is invoked.
func makeNoCallDriver(t *testing.T) *stub.Double {
	t.Helper()

	return &stub.Double{
		NameFn: func() string {
			t.Error("should not call Name")
			return ""
		},
		AppliedVersionsFn: func(_ context.Context, _ string) (driver.AppliedVersions, error) {
			t.Error("should not call AppliedVersions")
			return nil, nil
		},
		CreateSchemaMigrationsFn: func(_ context.Context, _ string) error {
			t.Error("should not call CreateSchemaMigrations")
			return nil
		},
		ExecuteFn: func(_ context.Context, _ string, _ ...any) error {
			t.Error("should not call Execute")
			return nil
		},
		UpdateSchemaMigrationsFn: func(_ context.Context, _ string, _ bool, _, _ string) error {
			t.Error("should not call UpdateSchemaMigrations")
			return nil
		},
		UpgradeSchemaMigrationsFn: func(_ context.Context, _ string) error {
			t.Error("should not call UpgradeSchemaMigrationsFn")
			return nil
		},
	}
}

// testBasicOperationWithoutOpts is a simple test meant to check that fn does
// not blow up when passed the bare minimum of required inputs and most
// importantly, no options. It's only targeted toward migrate and rollback
// functions.
func testBasicOperationWithoutOpts(
	t *testing.T,
	startingVersions []string,
	expectedNumCalls int,
	fn func(context.Context, driver.Driver, fs.FS, ...godfish.Opter) error,
) {
	t.Helper()

	dirFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	var numExecCalls, numUpdateCalls int
	driver := &stub.Double{
		AppliedVersionsFn: makeScanApplied(t, startingVersions...),
		ExecuteFn: func(context.Context, string, ...any) error {
			numExecCalls++
			return nil
		},
		CreateSchemaMigrationsFn: makeCreateSchemaMigrationsFn(nil),
		UpdateSchemaMigrationsFn: func(context.Context, string, bool, string, string) error {
			numUpdateCalls++
			return nil
		},
	}

	// For this test func, make sure not to pass any of the functional options.
	if err := fn(t.Context(), driver, dirFS); err != nil {
		t.Fatal(err)
	}

	if numExecCalls != expectedNumCalls {
		t.Errorf("wrong number of calls to Execute; got %d, expected %d", numExecCalls, expectedNumCalls)
	}
	if numUpdateCalls != expectedNumCalls {
		t.Errorf("wrong number of calls to UpdateSchemaMigrations; got %d, expected %d", numUpdateCalls, expectedNumCalls)
	}
}
