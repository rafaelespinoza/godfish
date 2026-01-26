package test

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testMigrate(t *testing.T, driver godfish.Driver, queries testdataQueries) {
	runTest := func(t *testing.T, driver godfish.Driver, dirFS fs.FS, migrationsTable string, expectedVersions []string) {
		err := godfish.Migrate(t.Context(), driver, dirFS, true, "", migrationsTable)
		if err != nil {
			t.Fatalf("could not Migrate in %s Direction; %v", internal.DirForward, err)
		}

		appliedVersions := collectAppliedMigrations(t, driver, migrationsTable)
		testAppliedMigrations(t, appliedVersions, expectedVersions)

		err = godfish.Migrate(t.Context(), driver, dirFS, false, expectedVersions[0], migrationsTable)
		if err != nil {
			t.Fatalf("could not Migrate in %s Direction; %v", internal.DirReverse, err)
		}

		appliedVersions = collectAppliedMigrations(t, driver, migrationsTable)
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
				path := setup(t, driver, stubs, skipMigration, test.migrationsTable)
				// Migrating all the way in reverse should also remove these tables. In case
				// it doesn't, teardown tables anyways to make this test less likely to
				// affect other tests.
				t.Cleanup(func() { teardown(t, driver, path, test.migrationsTable, "foos", "bars") })

				expectedVersions := []string{"12340102030405", "23450102030405", "34560102030405"}
				runTest(t, driver, os.DirFS(path), test.migrationsTable, expectedVersions)
			})
		}
	})

	t.Run("embedded migrations", func(t *testing.T) {
		subdir := getTestdataSubdir(driver)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range okMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				runTest(t, driver, dirFS, test.migrationsTable, []string{"1234", "2345", "3456"})
			})
		}
	})

	t.Run("invalid migrations table", func(t *testing.T) {
		subdir := getTestdataSubdir(driver)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range invalidMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				// Check that there's a clean slate.
				appliedVersions := collectAppliedMigrations(t, driver, internal.DefaultMigrationsTableName)
				testAppliedMigrations(t, appliedVersions, []string{})

				err := godfish.Migrate(t.Context(), driver, dirFS, true, "", test.migrationsTable)
				if !errors.Is(err, internal.ErrDataInvalid) {
					t.Fatalf("expected error (%v) to match %v", err, internal.ErrDataInvalid)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}

				// Check that it didn't try to do something silly, like update another table instead.
				appliedVersions = collectAppliedMigrations(t, driver, internal.DefaultMigrationsTableName)
				testAppliedMigrations(t, appliedVersions, []string{})
			})
		}
	})
}
