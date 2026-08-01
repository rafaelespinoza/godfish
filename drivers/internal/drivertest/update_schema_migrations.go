package drivertest

import (
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/internal"
)

func testUpdateSchemaMigrations(t *testing.T, driver driver.Driver) {
	t.Run("invalid migrations table", func(t *testing.T) {
		for _, test := range invalidMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				// Check that there's a clean slate.
				appliedVersions := collectAppliedMigrations(t, driver, internal.DefaultMigrationsTableName)
				testAppliedMigrations(t, appliedVersions, []string{})

				err := driver.UpdateSchemaMigrations(t.Context(), test.migrationsTable, true, "1234", test.migrationsTable)
				if !internal.IsInvalidDataError(err) {
					t.Fatalf("expected error (%v) to be an invalid data error", err)
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
