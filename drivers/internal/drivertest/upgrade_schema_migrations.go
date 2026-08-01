package drivertest

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/compat"
)

func testUpgradeSchemaMigrations(t *testing.T, d driver.Driver, queries testdataQueries) {
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
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					MigrationsTable: table,
				})
				return godfish.UpgradeSchemaMigrationsWith(t.Context(), driver, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runUpgradeSchemaMigrationsTests(t, d, queries, test.upgrade)
		})
	}
}

func runUpgradeSchemaMigrationsTests(t *testing.T, d driver.Driver, queries testdataQueries, upgrade compat.UpgradeSchemaFunc) {
	// The happy path for the library func, UpgradeSchemaMigrations*, is not easy
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
					teardown(t, d, t.TempDir(), migrationsTable, "foos", "bars")
					// Go further than the typical teardown and entirely remove the schema
					// migrations table. This positions us to expect an error when attempting
					// to upgrade that table.
					if err := d.Execute(t.Context(), "DROP TABLE IF EXISTS "+migrationsTable); err != nil {
						t.Fatalf("dropping migrations table: %v", err)
					}
				}

				// Expect to be unable to upgrade when the DB table is not there.
				err := upgrade(t.Context(), d, test.migrationsTable)
				if !errors.Is(err, driver.ErrSchemaMigrationsDoesNotExist) {
					t.Fatalf("expected for error (%v) to be %v", err, driver.ErrSchemaMigrationsDoesNotExist)
				}
				t.Log(err)

				// Now run a regular migration. An expected side effect is that it
				// creates the schema migrations table if it doesn't already exist.
				stubs := []testDriverStub{{content: queries.CreateFoos, version: formattedTime("1234")}}

				path := setup(t, d, stubs, "1234", test.migrationsTable)
				defer func() { teardown(t, d, path, test.migrationsTable, "foos", "bars") }()
				appliedVersions := collectAppliedMigrations(t, d, test.migrationsTable)
				testAppliedMigrations(t, appliedVersions, []string{"1234"})

				// Creating the schema migrations table in the previous step puts
				// it in the expected shape, so there is no need to upgrade here.
				//
				// A log message will mention this, capture it.
				var buf bytes.Buffer
				originalSlogger := slog.Default()
				defer slog.SetDefault(originalSlogger)
				slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

				err = upgrade(t.Context(), d, test.migrationsTable)
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
				err := upgrade(t.Context(), d, test.migrationsTable)
				if !internal.IsInvalidDataError(err) {
					t.Fatalf("expected error (%v) to be an invalid data error", err)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}
			})
		}
	})
}
