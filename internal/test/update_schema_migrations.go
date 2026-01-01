package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func testUpdateSchemaMigrations(t *testing.T, driver godfish.Driver) {
	t.Run("invalid migrations table", func(t *testing.T) {
		for _, test := range invalidMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				// Check that there's a clean slate.
				appliedVersions := collectAppliedVersions(t, driver, internal.DefaultMigrationsTableName)
				testAppliedVersions(t, appliedVersions, []string{})

				err := driver.UpdateSchemaMigrations(test.migrationsTable, true, "1234")
				if !errors.Is(err, internal.ErrDataInvalid) {
					t.Fatalf("expected error (%v) to match %v", err, internal.ErrDataInvalid)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}

				// Check that it didn't try to do something silly, like update another table instead.
				appliedVersions = collectAppliedVersions(t, driver, internal.DefaultMigrationsTableName)
				testAppliedVersions(t, appliedVersions, []string{})
			})
		}
	})
}
