package drivertest

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/compat"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testMigrate(t *testing.T, d driver.Driver, queries testdataQueries) {
	tests := []struct {
		name     string
		migrate  compat.MigrateFunc
		rollback compat.RollbackFunc
	}{
		{
			name: "Deprecated APIs",
			migrate: func(ctx context.Context, driver driver.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.Migrate(ctx, driver, fsys, true, v, tbl)
			},
			rollback: func(ctx context.Context, driver driver.Driver, fsys fs.FS, v, tbl string) error {
				return godfish.Migrate(ctx, driver, fsys, false, v, tbl)
			},
		},
		{
			name: "Replacement APIs",
			migrate: func(ctx context.Context, driver driver.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.MigrateWith(ctx, driver, fsys, opts...)
			},
			rollback: func(ctx context.Context, driver driver.Driver, fsys fs.FS, v, tbl string) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					TargetVersion:   v,
					MigrationsTable: tbl,
				})
				return godfish.RollbackWith(ctx, driver, fsys, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runMigrateTests(t, d, queries, test.migrate, test.rollback)
		})
	}
}

func runMigrateTests(t *testing.T, d driver.Driver, queries testdataQueries, migrate compat.MigrateFunc, rollback compat.RollbackFunc) {
	runTest := func(t *testing.T, driver driver.Driver, dirFS fs.FS, migrationsTable string, expectedVersions []string) {
		err := migrate(t.Context(), driver, dirFS, "", migrationsTable)
		if err != nil {
			t.Fatalf("could not Migrate in %s Direction; %v", internal.DirForward, err)
		}

		appliedVersions := collectAppliedMigrations(t, d, migrationsTable)
		testAppliedMigrations(t, appliedVersions, expectedVersions)

		err = rollback(t.Context(), d, dirFS, expectedVersions[0], migrationsTable)
		if err != nil {
			t.Fatalf("could not Migrate in %s Direction; %v", internal.DirReverse, err)
		}

		appliedVersions = collectAppliedMigrations(t, d, migrationsTable)
		expectedVersions = []string{}
		testAppliedMigrations(t, appliedVersions, expectedVersions)
	}

	t.Run("migrations on filesystem", func(t *testing.T) {
		stubs := []testDriverStub{
			{
				content: queries.CreateFoos,
				version: formattedTime("12340102030405"),
			},
			{
				content: queries.CreateBars,
				version: formattedTime("23450102030405"),
			},
			{
				content: queries.AlterFoos,
				version: formattedTime("34560102030405"),
			},
		}

		for _, test := range okMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				path := setup(t, d, stubs, skipMigration, test.migrationsTable)
				// Migrating all the way in reverse should also remove these tables. In case
				// it doesn't, teardown tables anyways to make this test less likely to
				// affect other tests.
				t.Cleanup(func() { teardown(t, d, path, test.migrationsTable, "foos", "bars") })

				expectedVersions := []string{"12340102030405", "23450102030405", "34560102030405"}
				runTest(t, d, os.DirFS(path), test.migrationsTable, expectedVersions)
			})
		}
	})

	t.Run("embedded migrations", func(t *testing.T) {
		subdir := getTestdataSubdir(d)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range okMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				runTest(t, d, dirFS, test.migrationsTable, []string{"1234", "2345", "3456"})
			})
		}
	})

	t.Run("invalid migrations table", func(t *testing.T) {
		subdir := getTestdataSubdir(d)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range invalidMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				// Check that there's a clean slate.
				appliedVersions := collectAppliedMigrations(t, d, internal.DefaultMigrationsTableName)
				testAppliedMigrations(t, appliedVersions, []string{})

				err := migrate(t.Context(), d, dirFS, "", test.migrationsTable)
				if !internal.IsInvalidDataError(err) {
					t.Fatalf("expected error (%v) to be an invalid data error", err)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}

				// Check that it didn't try to do something silly, like update another table instead.
				appliedVersions = collectAppliedMigrations(t, d, internal.DefaultMigrationsTableName)
				testAppliedMigrations(t, appliedVersions, []string{})
			})
		}
	})
}
