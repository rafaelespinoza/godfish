package test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testMigrate(t *testing.T, driver godfish.Driver, queries testdataQueries) {
	runTest := func(t *testing.T, driver godfish.Driver, dirFS fs.FS, expectedVersions []string) {
		err := godfish.Migrate(driver, dirFS, true, "")
		if err != nil {
			t.Fatalf("could not Migrate in %s Direction; %v", internal.DirForward, err)
		}

		appliedVersions := collectAppliedVersions(t, driver)
		testAppliedVersions(t, appliedVersions, expectedVersions)

		err = godfish.Migrate(driver, dirFS, false, expectedVersions[0])
		if err != nil {
			t.Fatalf("could not Migrate in %s Direction; %v", internal.DirReverse, err)
		}

		appliedVersions = collectAppliedVersions(t, driver)
		expectedVersions = []string{}
		testAppliedVersions(t, appliedVersions, expectedVersions)
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

		path := setup(t, driver, stubs, skipMigration)
		// Migrating all the way in reverse should also remove these tables
		// teardown. In case it doesn't, teardown tables anyways so it's less likely
		// to affect other tests.
		t.Cleanup(func() { teardown(t, driver, path, "foos", "bars") })

		expectedVersions := []string{"12340102030405", "23450102030405", "34560102030405"}
		runTest(t, driver, os.DirFS(path), expectedVersions)
	})

	t.Run("embedded migrations", func(t *testing.T) {
		subdir := getTestdataSubdir(driver)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, driver, dirFS, []string{"1234", "2345", "3456"})
	})
}
