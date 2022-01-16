package test

import (
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func testMigrate(t *testing.T, driver godfish.Driver, queries Queries) {
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
	path, err := setup(driver, t.Name(), stubs, skipMigration)
	if err != nil {
		t.Errorf("could not setup test; %v", err)
		return
	}
	// Migrating all the way in reverse should also remove these tables
	// teardown. In case it doesn't, teardown tables anyways so it doesn't
	// affect other tests.
	defer teardown(driver, path, "foos", "bars")

	err = godfish.Migrate(driver, path, true, "")
	if err != nil {
		t.Errorf("could not Migrate in %s Direction; %v", internal.DirForward, err)
	}

	appliedVersions, err := collectAppliedVersions(driver)
	if err != nil {
		t.Fatal(err)
	}
	expectedVersions := []string{"12340102030405", "23450102030405", "34560102030405"}
	err = testAppliedVersions(appliedVersions, expectedVersions)
	if err != nil {
		t.Error(err)
	}

	err = godfish.Migrate(driver, path, false, "12340102030405")
	if err != nil {
		t.Errorf("could not Migrate in %s Direction; %v", internal.DirReverse, err)
	}

	appliedVersions, err = collectAppliedVersions(driver)
	if err != nil {
		t.Fatal(err)
	}
	expectedVersions = []string{}
	err = testAppliedVersions(appliedVersions, expectedVersions)
	if err != nil {
		t.Error(err)
	}
}
