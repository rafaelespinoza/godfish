package test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/compat"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testContext(t *testing.T, driver godfish.Driver) {
	tests := []struct {
		name     string
		migrate  compat.MigrateFunc
		rollback compat.RollbackFunc
	}{
		{
			name: "Deprecated APIs",
			migrate: func(ctx context.Context, d godfish.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.ApplyMigration(ctx, d, fsys, true, v, tbl)
			},
			rollback: func(ctx context.Context, d godfish.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.Migrate(ctx, d, fsys, false, v, tbl)
			},
		},
		{
			name: "Replacement APIs",
			migrate: func(ctx context.Context, d godfish.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.ApplyMigrationWith(ctx, d, fsys, opts...)
			},
			rollback: func(ctx context.Context, d godfish.Driver, fsys fs.FS, v, tbl string) error {
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
			runContextTests(t, driver, test.migrate, test.rollback)
		})
	}
}

func runContextTests(t *testing.T, driver godfish.Driver, migrate compat.MigrateFunc, rollback compat.RollbackFunc) {
	subdir := getTestdataSubdir(driver)
	dirFS, err := fs.Sub(testdata.Migrations, subdir)
	if err != nil {
		t.Fatal(err)
	}
	const table = internal.DefaultMigrationsTableName

	{ // Setup
		defer func() {
			err := rollback(t.Context(), driver, dirFS, "1234", table)
			if err != nil {
				t.Fatalf("resetting DB, could not rollback: %v", err)
			}
			appliedVersions := collectAppliedMigrations(t, driver, table)
			testAppliedMigrations(t, appliedVersions, []string{})
		}()

		if err = driver.CreateSchemaMigrationsTable(t.Context(), table); err != nil {
			t.Fatal(err)
		}

		// Ensure a clean slate, then set expected state.
		appliedVersions := collectAppliedMigrations(t, driver, table)
		testAppliedMigrations(t, appliedVersions, []string{})

		if err = migrate(t.Context(), driver, dirFS, "1234", table); err != nil {
			t.Fatal(err)
		}
		appliedVersions = collectAppliedMigrations(t, driver, table)
		testAppliedMigrations(t, appliedVersions, []string{"1234"})
	}

	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 0)
		defer cancel()

		err := migrate(ctx, driver, dirFS, "2345", table)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
		}
		appliedVersions := collectAppliedMigrations(t, driver, table)
		testAppliedMigrations(t, appliedVersions, []string{"1234"})
	})

	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		cancel()

		err := migrate(ctx, driver, dirFS, "2345", table)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected error %v, got %v", context.Canceled, err)
		}
		appliedVersions := collectAppliedMigrations(t, driver, table)
		testAppliedMigrations(t, appliedVersions, []string{"1234"})
	})
}
