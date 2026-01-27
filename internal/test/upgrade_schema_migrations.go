package test

import (
	"bytes"
	"cmp"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func testUpgradeSchemaMigrations(t *testing.T, driver godfish.Driver, queries testdataQueries) {
	// The happy path for the library func, UpgradeSchemaMigrations, is not easy
	// to test from here because it would require an older version of the library
	// to set up the upgradable state and then use a newer library version to
	// perform the upgrade. However there is an integration test for it elsewhere
	// in this project. These tests check for some basic error handling.

	t.Run("table does not exist or already upgraded", func(t *testing.T) {
		for _, test := range okMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				{ // Setup

					// Empty the DB.
					migrationsTable := cmp.Or(test.migrationsTable, internal.DefaultMigrationsTableName)
					teardown(t, driver, t.TempDir(), migrationsTable, "foos", "bars")
					// Go further than the typical teardown and entirely remove the schema
					// migrations table. This positions us to expect an error when attempting
					// to upgrade that table.
					if err := driver.Execute(t.Context(), "DROP TABLE IF EXISTS "+migrationsTable); err != nil {
						t.Fatalf("dropping migrations table: %v", err)
					}
					if driver.Name() == "stub" {
						// This driver only has in-memory data. Like the other drivers, reset everything.
						driver = stub.NewDriver()
					}
				}

				// Expect to be unable to upgrade when the DB table is not there.
				err := godfish.UpgradeSchemaMigrations(t.Context(), driver, test.migrationsTable)
				if !errors.Is(err, godfish.ErrSchemaMigrationsDoesNotExist) {
					t.Fatalf("expected for error (%v) to be %v", err, godfish.ErrSchemaMigrationsDoesNotExist)
				}
				t.Log(err)

				// Now run a regular migration. An expected side effect is that it
				// creates the schema migrations table if it doesn't already exist.
				stubs := []testDriverStub{{content: queries.CreateFoos, version: formattedTime("1234")}}

				path := setup(t, driver, stubs, "1234", test.migrationsTable)
				defer func() { teardown(t, driver, path, test.migrationsTable, "foos", "bars") }()
				appliedVersions := collectAppliedMigrations(t, driver, test.migrationsTable)
				testAppliedMigrations(t, appliedVersions, []string{"1234"})

				// Creating the schema migrations table in the previous step puts
				// it in the expected shape, so there is no need to upgrade here.
				//
				// A log message will mention this, capture it.
				var buf bytes.Buffer
				originalSlogger := slog.Default()
				defer slog.SetDefault(originalSlogger)
				slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

				err = godfish.UpgradeSchemaMigrations(t.Context(), driver, test.migrationsTable)
				if err != nil {
					t.Fatal(err)
				}

				gotLog := buf.String()
				t.Log(gotLog)
				for _, exp := range []string{"no need to upgrade", test.migrationsTable} {
					if !strings.Contains(gotLog, exp) {
						t.Errorf("expected for log messages (%q) to contain %q", gotLog, exp)
					}
				}
			})
		}
	})

	t.Run("invalid migrations table", func(t *testing.T) {
		for _, test := range invalidMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				err := godfish.UpgradeSchemaMigrations(t.Context(), driver, test.migrationsTable)
				if !errors.Is(err, internal.ErrDataInvalid) {
					t.Fatalf("expected error (%v) to match %v", err, internal.ErrDataInvalid)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}
			})
		}
	})
}
