package test

import (
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
)

func testApplyMigration(t *testing.T, driver godfish.Driver) {
	// testSetupState is the database state before calling ApplyMigration.
	type testSetupState struct {
		// migrateTo is the version that the DB should be at.
		migrateTo string
		// stubs is a list of stubbed migration data to populate the DB.
		stubs []testDriverStub
	}

	// testInput is passed to ApplyMigration.
	type testInput struct {
		// direction is the direction to migrate.
		direction godfish.Direction
		// version is where to go when calling ApplyMigration.
		version string
	}

	type expectedOutput struct {
		// appliedVersions is where you should be after calling
		// ApplyMigration.
		appliedVersions []string
		// err says that we expect some error to happen, and the code should
		// handle it.
		err bool
	}

	runTest := func(t *testing.T, setupState testSetupState, input testInput, expected expectedOutput) {
		t.Helper()
		pathToFiles, err := setup(driver, t.Name(), setupState.stubs, setupState.migrateTo)
		if err != nil {
			t.Errorf("could not setup test; %v", err)
			return
		}
		defer teardown(driver, pathToFiles, "foos", "bars")

		err = godfish.ApplyMigration(driver, pathToFiles, input.direction, input.version)
		if err != nil && !expected.err {
			t.Errorf("could not apply migration; %v", err)
			return
		} else if err == nil && expected.err {
			t.Error("expected error but got none")
			return
		}

		actualVersions, err := collectAppliedVersions(driver)
		if err != nil {
			t.Fatal(err)
		}

		err = testAppliedVersions(actualVersions, expected.appliedVersions)
		if err != nil {
			t.Error(err)
		}
	}

	t.Run("setup state: no migration files", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				stubs: []testDriverStub{},
			},
			testInput{
				direction: godfish.DirForward,
				version:   "",
			},
			expectedOutput{
				appliedVersions: []string{},
				err:             true, // no version found
			},
		)

		runTest(
			t,
			testSetupState{
				stubs: []testDriverStub{},
			},
			testInput{
				direction: godfish.DirReverse,
				version:   "",
			},
			expectedOutput{
				appliedVersions: []string{},
				err:             true, // no version found
			},
		)
	})

	t.Run("setup state: do not migrate", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				migrateTo: skipMigration,
				stubs:     _DefaultTestDriverStubs,
			},
			testInput{
				direction: godfish.DirForward,
				version:   "",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405"},
			},
		)

		runTest(
			t,
			testSetupState{
				migrateTo: skipMigration,
				stubs:     _DefaultTestDriverStubs,
			},
			testInput{
				direction: godfish.DirReverse,
				version:   "",
			},
			expectedOutput{
				appliedVersions: []string{},
				err:             true, // no version found
			},
		)
	})

	t.Run("setup state: go forward partway or all of the way.", func(t *testing.T) {
		t.Run("Forward", func(t *testing.T) {
			runTest(
				t,
				testSetupState{
					migrateTo: "12340102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405"},
				},
			)

			runTest(
				t,
				testSetupState{
					migrateTo: "23450102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				},
			)

			runTest(
				t,
				testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no version found
				},
			)
		})

		t.Run("Reverse", func(t *testing.T) {
			runTest(
				t,
				testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				testInput{
					direction: godfish.DirReverse,
					version:   "23450102030405",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405", "34560102030405"},
				},
			)

			runTest(
				t,
				testSetupState{
					migrateTo: "23450102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				testInput{
					direction: godfish.DirReverse,
					version:   "",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			)

			runTest(
				t,
				testSetupState{
					migrateTo: "12340102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				testInput{
					direction: godfish.DirReverse,
					version:   "",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405"},
					err:             true, // no version found
				},
			)
		})
	})

	t.Run("Reverse", func(t *testing.T) {
		// test reverse, available migration at end of range.
		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs: []testDriverStub{
					{
						content: struct{ forward, reverse string }{
							forward: `CREATE TABLE foos (id int);`,
						},
						version: formattedTime("12340102030405"),
					},
					{
						content: struct{ forward, reverse string }{
							forward: `CREATE TABLE bars (id int);`,
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
				},
			},
			testInput{
				direction: godfish.DirReverse,
				version:   "34560102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405"},
			},
		)
	})

	t.Run("migration file does not exist", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     _DefaultTestDriverStubs,
			},
			testInput{
				direction: godfish.DirForward,
				version:   "43210102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)

		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     _DefaultTestDriverStubs,
			},
			testInput{
				direction: godfish.DirReverse,
				version:   "43210102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)

		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     _DefaultTestDriverStubs,
			},
			testInput{
				direction: godfish.DirReverse,
				// target migration only exists in forward direction.
				// target > available.
				version: "34560102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)

		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     _DefaultTestDriverStubs,
			},
			testInput{
				direction: godfish.DirReverse,
				// target migration only exists in forward direction.
				// target < available.
				version: "12340102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)
	})

	t.Run("Error during migration execution", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs: append(_DefaultTestDriverStubs, testDriverStub{
					content: struct{ forward, reverse string }{
						forward: "invalid SQL",
					},
					version: formattedTime("45670102030405"),
				}),
			},
			testInput{
				direction: godfish.DirForward,
				version:   "45670102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // error executing SQL
			},
		)
	})

	t.Run("Short version", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				migrateTo: "2345",
				stubs: []testDriverStub{
					{
						content: struct{ forward, reverse string }{
							forward: `CREATE TABLE foos (id int);`,
						},
						version: formattedTime("1234"),
					},
					{
						content: struct{ forward, reverse string }{
							forward: `CREATE TABLE bars (id int);`,
						},
						version: formattedTime("2345"),
					},
					{
						content: struct{ forward, reverse string }{
							forward: `ALTER TABLE foos ADD COLUMN a varchar(255);`,
							reverse: `ALTER TABLE foos DROP COLUMN a;`,
						},
						version: formattedTime("3456"),
					},
				},
			},
			testInput{
				direction: godfish.DirForward,
				version:   "3456",
			},
			expectedOutput{
				appliedVersions: []string{"1234", "2345", "3456"},
			},
		)
	})

	t.Run("Multiple statements", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				migrateTo: skipMigration,
				stubs: []testDriverStub{
					{
						content: struct{ forward, reverse string }{
							forward: strings.Join([]string{
								"CREATE TABLE foos (id int);",
								"",
								"CREATE TABLE bars (id int);  ",
								"",
								"ALTER TABLE foos ADD COLUMN a varchar(255) ;",
								"  ",
								"",
							}, "\n"),
						},
						version: formattedTime("12340102030405"),
					},
				},
			},
			testInput{
				direction: godfish.DirForward,
				version:   "12340102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405"},
			},
		)

		// should handle an error in the middle.
		runTest(
			t,
			testSetupState{
				migrateTo: skipMigration,
				stubs: []testDriverStub{
					{
						content: struct{ forward, reverse string }{
							forward: strings.Join([]string{
								"CREATE TABLE foos (id int);",
								"invalid SQL;",
								"ALTER TABLE foos ADD COLUMN a varchar(255);",
							}, "\n"),
						},
						version: formattedTime("12340102030405"),
					},
				},
			},
			testInput{
				direction: godfish.DirForward,
				version:   "12340102030405",
			},
			expectedOutput{
				appliedVersions: []string{},
				err:             true,
			},
		)
	})

	t.Run("Alternate filenames", func(t *testing.T) {
		t.Run("migrate rollback", func(t *testing.T) {
			runTest(
				t,
				testSetupState{
					migrateTo: skipMigration,
					stubs:     _StubsWithMigrateRollbackIndirectives,
				},
				testInput{
					direction: godfish.DirForward,
					version:   "12340102030405",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			)

			runTest(
				t,
				testSetupState{
					migrateTo: "12340102030405",
					stubs:     _StubsWithMigrateRollbackIndirectives,
				},
				testInput{
					direction: godfish.DirReverse,
					version:   "12340102030405",
				},
				expectedOutput{
					appliedVersions: []string{},
				},
			)
		})

		t.Run("up down", func(t *testing.T) {
			runTest(
				t,
				testSetupState{
					migrateTo: skipMigration,
					stubs:     _StubsWithUpDownIndirectives,
				},
				testInput{
					direction: godfish.DirForward,
					version:   "12340102030405",
				},
				expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			)

			runTest(
				t,
				testSetupState{
					migrateTo: "12340102030405",
					stubs:     _StubsWithUpDownIndirectives,
				},
				testInput{
					direction: godfish.DirReverse,
					version:   "12340102030405",
				},
				expectedOutput{
					appliedVersions: []string{},
				},
			)
		})
	})
}
