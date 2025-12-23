package test

import (
	"os"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func testApplyMigration(t *testing.T, driver godfish.Driver, queries testdataQueries) {
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
		direction internal.Direction
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
		pathToFiles := setup(t, driver, setupState.stubs, setupState.migrateTo)
		defer teardown(t, driver, pathToFiles, "foos", "bars")

		err := godfish.ApplyMigration(driver, os.DirFS(pathToFiles), input.direction == internal.DirForward, input.version)
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

	var defaultStubs = []testDriverStub{
		{
			content: queries.CreateFoos,
			version: formattedTime("12340102030405"),
		},
		{
			content: queries.AlterFoos,
			version: formattedTime("23450102030405"),
		},
		{
			content: queries.CreateBars,
			version: formattedTime("34560102030405"),
		},
	}

	t.Run("setup state: no migration files", func(t *testing.T) {
		runTest(
			t,
			testSetupState{
				stubs: []testDriverStub{},
			},
			testInput{
				direction: internal.DirForward,
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
				direction: internal.DirReverse,
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
				stubs:     defaultStubs,
			},
			testInput{
				direction: internal.DirForward,
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
				stubs:     defaultStubs,
			},
			testInput{
				direction: internal.DirReverse,
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
					stubs:     defaultStubs,
				},
				testInput{
					direction: internal.DirForward,
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
					stubs:     defaultStubs,
				},
				testInput{
					direction: internal.DirForward,
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
					stubs:     defaultStubs,
				},
				testInput{
					direction: internal.DirForward,
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
					stubs:     defaultStubs,
				},
				testInput{
					direction: internal.DirReverse,
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
					stubs:     defaultStubs,
				},
				testInput{
					direction: internal.DirReverse,
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
					stubs: []testDriverStub{
						{
							content: migrationContent{Forward: queries.CreateFoos.Forward},
							version: formattedTime("12340102030405"),
						},
						{
							content: migrationContent{
								Forward: queries.AlterFoos.Forward,
								Reverse: queries.AlterFoos.Reverse,
							},
							version: formattedTime("23450102030405"),
						},
						{
							content: migrationContent{Forward: queries.CreateBars.Forward},
							version: formattedTime("34560102030405"),
						},
					},
				},
				testInput{
					direction: internal.DirReverse,
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
				},
			},
			testInput{
				direction: internal.DirReverse,
				version:   "34560102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405"},
			},
		)
	})

	t.Run("migration file does not exist", func(t *testing.T) {
		var stubs = []testDriverStub{
			{
				content: migrationContent{Forward: queries.CreateFoos.Forward},
				version: formattedTime("12340102030405"),
			},
			{
				content: migrationContent{
					Forward: queries.AlterFoos.Forward,
					Reverse: queries.AlterFoos.Reverse,
				},
				version: formattedTime("23450102030405"),
			},
			{
				content: migrationContent{Forward: queries.CreateBars.Forward},
				version: formattedTime("34560102030405"),
			},
		}

		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     stubs,
			},
			testInput{
				direction: internal.DirForward,
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
				stubs:     stubs,
			},
			testInput{
				direction: internal.DirReverse,
				version:   "43210102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)

		// targeted migration only exists in forward direction.
		// target > available.
		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     stubs,
			},
			testInput{
				direction: internal.DirReverse,
				version:   "34560102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)

		// targeted migration only exists in forward direction.
		// target < available.
		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs:     stubs,
			},
			testInput{
				direction: internal.DirReverse,
				version:   "12340102030405",
			},
			expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // no files found
			},
		)
	})

	t.Run("Error during migration execution", func(t *testing.T) {
		stubs := make([]testDriverStub, len(defaultStubs))
		copy(stubs, defaultStubs)

		runTest(
			t,
			testSetupState{
				migrateTo: "34560102030405",
				stubs: append(stubs, testDriverStub{
					content: migrationContent{
						Forward: "invalid SQL",
					},
					version: formattedTime("45670102030405"),
				}),
			},
			testInput{
				direction: internal.DirForward,
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
						content: queries.CreateFoos,
						version: formattedTime("1234"),
					},
					{
						content: queries.CreateBars,
						version: formattedTime("2345"),
					},
					{
						content: queries.AlterFoos,
						version: formattedTime("3456"),
					},
				},
			},
			testInput{
				direction: internal.DirForward,
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
						content: migrationContent{
							Forward: strings.Join([]string{
								queries.CreateFoos.Forward,
								"",
								queries.CreateBars.Forward,
								"",
								queries.AlterFoos.Forward,
								"  ",
								"",
							}, "\n"),
						},
						version: formattedTime("12340102030405"),
					},
				},
			},
			testInput{
				direction: internal.DirForward,
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
						content: migrationContent{
							Forward: strings.Join([]string{
								queries.CreateFoos.Forward,
								"invalid SQL;",
								queries.AlterFoos.Forward,
							}, "\n"),
						},
						version: formattedTime("12340102030405"),
					},
				},
			},
			testInput{
				direction: internal.DirForward,
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
			var stubs = []testDriverStub{
				{
					content: queries.CreateFoos,
					indirectives: struct{ forward, reverse internal.Indirection }{
						forward: internal.Indirection{Label: "migrate"},
						reverse: internal.Indirection{Label: "rollback"},
					},
					version: formattedTime("12340102030405"),
				},
			}

			runTest(
				t,
				testSetupState{
					migrateTo: skipMigration,
					stubs:     stubs,
				},
				testInput{
					direction: internal.DirForward,
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
					stubs:     stubs,
				},
				testInput{
					direction: internal.DirReverse,
					version:   "12340102030405",
				},
				expectedOutput{
					appliedVersions: []string{},
				},
			)
		})

		t.Run("up down", func(t *testing.T) {
			var stubs = []testDriverStub{
				{
					content: queries.CreateFoos,
					indirectives: struct{ forward, reverse internal.Indirection }{
						forward: internal.Indirection{Label: "up"},
						reverse: internal.Indirection{Label: "down"},
					},
					version: formattedTime("12340102030405"),
				},
			}

			runTest(
				t,
				testSetupState{
					migrateTo: skipMigration,
					stubs:     stubs,
				},
				testInput{
					direction: internal.DirForward,
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
					stubs:     stubs,
				},
				testInput{
					direction: internal.DirReverse,
					version:   "12340102030405",
				},
				expectedOutput{
					appliedVersions: []string{},
				},
			)
		})
	})
}
