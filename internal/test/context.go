package test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testContext(t *testing.T, driver godfish.Driver) {
	subdir := getTestdataSubdir(driver)
	dirFS, err := fs.Sub(testdata.Migrations, subdir)
	if err != nil {
		t.Fatal(err)
	}
	const table = internal.DefaultMigrationsTableName

	{ // Setup
		if err := driver.Connect(mustDSN()); err != nil {
			t.Fatal(err)
		}
		defer driver.Close()
		defer func() {
			// Reset
			err := godfish.Migrate(t.Context(), driver, dirFS, false, "1234", table)
			if err != nil {
				t.Fatalf("resetting DB, could not Migrate in %s Direction; %v", internal.DirReverse, err)
			}
			appliedVersions := collectAppliedVersions(t, driver, table)
			testAppliedVersions(t, appliedVersions, []string{})
		}()
		if err = driver.CreateSchemaMigrationsTable(t.Context(), table); err != nil {
			t.Fatal(err)
		}

		// Ensure a clean slate, then set expected state.
		appliedVersions := collectAppliedVersions(t, driver, table)
		testAppliedVersions(t, appliedVersions, []string{})
		if err = godfish.ApplyMigration(t.Context(), driver, dirFS, true, "1234", table); err != nil {
			t.Fatal(err)
		}
		appliedVersions = collectAppliedVersions(t, driver, table)
		testAppliedVersions(t, appliedVersions, []string{"1234"})
	}

	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 0)
		defer cancel()

		err := godfish.ApplyMigration(ctx, driver, dirFS, true, "2345", table)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected for error (%v) to match %v", err, context.DeadlineExceeded)
		}
		appliedVersions := collectAppliedVersions(t, driver, table)
		testAppliedVersions(t, appliedVersions, []string{"1234"})
	})

	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		cancel()

		err := godfish.ApplyMigration(ctx, driver, dirFS, true, "2345", table)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected for error (%v) to match %v", err, context.Canceled)
		}
		appliedVersions := collectAppliedVersions(t, driver, table)
		testAppliedVersions(t, appliedVersions, []string{"1234"})
	})
}
