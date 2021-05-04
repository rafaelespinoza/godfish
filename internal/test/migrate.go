package test

import (
	"testing"

	"github.com/rafaelespinoza/godfish"
)

func testMigrate(t *testing.T, driver godfish.Driver) {
	stubs := []testDriverStub{
		{
			content: struct{ forward, reverse string }{
				forward: `CREATE TABLE foos (id int);`,
				reverse: `DROP TABLE foos;`,
			},
			version: formattedTime("12340102030405"),
		},
		{
			content: struct{ forward, reverse string }{
				forward: `CREATE TABLE bars (id int);`,
				reverse: `DROP TABLE bars;`,
			},
			version: formattedTime("23450102030405"),
		},
		{
			content: struct{ forward, reverse string }{
				forward: `ALTER TABLE foos ADD COLUMN a varchar(255);`,
				reverse: `ALTER TABLE foos DROP COLUMN a;`,
			},
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

	err = godfish.Migrate(driver, path, godfish.DirForward, "")
	if err != nil {
		t.Errorf("could not Migrate in %s Direction; %v", godfish.DirForward, err)
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

	err = godfish.Migrate(driver, path, godfish.DirReverse, "12340102030405")
	if err != nil {
		t.Errorf("could not Migrate in %s Direction; %v", godfish.DirReverse, err)
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
