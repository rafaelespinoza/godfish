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

	type testCase struct {
		name       string
		setupState testSetupState
		input      testInput
		expected   expectedOutput
	}

	runTest := func(t *testing.T, test testCase) {
		t.Helper()

		setupState, input, expected := test.setupState, test.input, test.expected

		pathToFiles := setup(t, driver, setupState.stubs, setupState.migrateTo)
		t.Cleanup(func() { teardown(t, driver, pathToFiles, "foos", "bars") })

		err := godfish.ApplyMigration(driver, os.DirFS(pathToFiles), input.direction == internal.DirForward, input.version)
		if err != nil && !expected.err {
			t.Errorf("could not apply migration; %v", err)
			return
		} else if err == nil && expected.err {
			t.Error("expected error but got none")
			return
		}

		actualVersions := collectAppliedVersions(t, driver)
		testAppliedVersions(t, actualVersions, expected.appliedVersions)
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
		tests := []testCase{
			{
				name:       "Forward",
				setupState: testSetupState{stubs: []testDriverStub{}},
				input:      testInput{direction: internal.DirForward, version: ""},
				expected: expectedOutput{
					appliedVersions: []string{},
					err:             true, // no version found
				},
			},
			{
				name:       "Reverse",
				setupState: testSetupState{stubs: []testDriverStub{}},
				input:      testInput{direction: internal.DirReverse, version: ""},
				expected: expectedOutput{
					appliedVersions: []string{},
					err:             true, // no version found
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})

	t.Run("setup state: do not migrate", func(t *testing.T) {
		tests := []testCase{
			{
				name:       "Forward",
				setupState: testSetupState{migrateTo: skipMigration, stubs: defaultStubs},
				input:      testInput{direction: internal.DirForward, version: ""},
				expected:   expectedOutput{appliedVersions: []string{"12340102030405"}},
			},
			{
				name:       "Reverse",
				setupState: testSetupState{migrateTo: skipMigration, stubs: defaultStubs},
				input:      testInput{direction: internal.DirReverse, version: ""},
				expected: expectedOutput{
					appliedVersions: []string{},
					err:             true, // no version found
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})

	t.Run("setup state: go forward partway or all of the way", func(t *testing.T) {
		t.Run("Forward", func(t *testing.T) {
			tests := []testCase{
				{
					name:       "start at 1234...",
					setupState: testSetupState{migrateTo: "12340102030405", stubs: defaultStubs},
					input:      testInput{direction: internal.DirForward, version: ""},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405", "23450102030405"},
					},
				},
				{
					name:       "start at 2345...",
					setupState: testSetupState{migrateTo: "23450102030405", stubs: defaultStubs},
					input:      testInput{direction: internal.DirForward, version: ""},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					},
				},
				{
					name:       "start at 3456...",
					setupState: testSetupState{migrateTo: "34560102030405", stubs: defaultStubs},
					input:      testInput{direction: internal.DirForward, version: ""},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
						err:             true, // no version found
					},
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) { runTest(t, test) })
			}
		})

		t.Run("Reverse", func(t *testing.T) {
			tests := []testCase{
				{
					name:       "start at 3456...",
					setupState: testSetupState{migrateTo: "34560102030405", stubs: defaultStubs},
					input:      testInput{direction: internal.DirReverse, version: "23450102030405"},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405", "34560102030405"},
					},
				},
				{
					name:       "start at 2345...",
					setupState: testSetupState{migrateTo: "23450102030405", stubs: defaultStubs},
					input:      testInput{direction: internal.DirReverse, version: ""},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405"},
					},
				},
				{
					name: "start at 1234...",
					setupState: testSetupState{
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
					input: testInput{
						direction: internal.DirReverse,
						version:   "",
					},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405"},
						err:             true, // no version found
					},
				},
				{
					name: "start at 3456 with available reverse migration at end of range",
					setupState: testSetupState{
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
					input: testInput{
						direction: internal.DirReverse,
						version:   "34560102030405",
					},
					expected: expectedOutput{
						appliedVersions: []string{"12340102030405", "23450102030405"},
					},
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) { runTest(t, test) })
			}
		})
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

		tests := []testCase{
			{
				name:       "Forward",
				setupState: testSetupState{migrateTo: "34560102030405", stubs: stubs},
				input:      testInput{direction: internal.DirForward, version: "43210102030405"},
				expected: expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			{
				name:       "Reverse",
				setupState: testSetupState{migrateTo: "34560102030405", stubs: stubs},
				input:      testInput{direction: internal.DirReverse, version: "43210102030405"},
				expected: expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			{
				name:       "Reverse, targeted migration only exists in forward direction, target > available",
				setupState: testSetupState{migrateTo: "34560102030405", stubs: stubs},
				input:      testInput{direction: internal.DirReverse, version: "34560102030405"},
				expected: expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			{
				name:       "Reverse, targeted migration only exists in forward direction, target < available",
				setupState: testSetupState{migrateTo: "34560102030405", stubs: stubs},
				input:      testInput{direction: internal.DirReverse, version: "12340102030405"},
				expected: expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})

	t.Run("Error during migration execution", func(t *testing.T) {
		stubs := make([]testDriverStub, len(defaultStubs))
		copy(stubs, defaultStubs)

		runTest(t, testCase{
			setupState: testSetupState{
				migrateTo: "34560102030405",
				stubs: append(stubs, testDriverStub{
					content: migrationContent{
						Forward: "invalid SQL",
					},
					version: formattedTime("45670102030405"),
				}),
			},
			input: testInput{
				direction: internal.DirForward,
				version:   "45670102030405",
			},
			expected: expectedOutput{
				appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				err:             true, // error executing SQL
			},
		})
	})

	t.Run("Short version", func(t *testing.T) {
		runTest(t, testCase{
			setupState: testSetupState{
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
			input:    testInput{direction: internal.DirForward, version: "3456"},
			expected: expectedOutput{appliedVersions: []string{"1234", "2345", "3456"}},
		})
	})

	t.Run("Multiple statements", func(t *testing.T) {
		tests := []testCase{
			{
				name: "ok",
				setupState: testSetupState{
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
				input:    testInput{direction: internal.DirForward, version: "12340102030405"},
				expected: expectedOutput{appliedVersions: []string{"12340102030405"}},
			},
			{
				name: "should handle an error in the middle",
				setupState: testSetupState{
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
				input: testInput{direction: internal.DirForward, version: "12340102030405"},
				expected: expectedOutput{
					appliedVersions: []string{},
					err:             true,
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})

	t.Run("Alternate filenames", func(t *testing.T) {
		var migrateRollbackStubs = []testDriverStub{
			{
				content: queries.CreateFoos,
				indirectives: struct{ forward, reverse internal.Indirection }{
					forward: internal.Indirection{Label: "migrate"},
					reverse: internal.Indirection{Label: "rollback"},
				},
				version: formattedTime("12340102030405"),
			},
		}

		var upDownStubs = []testDriverStub{
			{
				content: queries.CreateFoos,
				indirectives: struct{ forward, reverse internal.Indirection }{
					forward: internal.Indirection{Label: "up"},
					reverse: internal.Indirection{Label: "down"},
				},
				version: formattedTime("12340102030405"),
			},
		}

		tests := []testCase{
			{
				name:       "migrate",
				setupState: testSetupState{migrateTo: skipMigration, stubs: migrateRollbackStubs},
				input:      testInput{direction: internal.DirForward, version: "12340102030405"},
				expected:   expectedOutput{appliedVersions: []string{"12340102030405"}},
			},
			{
				name:       "rollback",
				setupState: testSetupState{migrateTo: "12340102030405", stubs: migrateRollbackStubs},
				input:      testInput{direction: internal.DirReverse, version: "12340102030405"},
				expected:   expectedOutput{appliedVersions: []string{}},
			},
			{
				name:       "up",
				setupState: testSetupState{migrateTo: skipMigration, stubs: upDownStubs},
				input:      testInput{direction: internal.DirForward, version: "12340102030405"},
				expected: expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			},
			{
				name:       "down",
				setupState: testSetupState{migrateTo: "12340102030405", stubs: upDownStubs},
				input:      testInput{direction: internal.DirReverse, version: "12340102030405"},
				expected: expectedOutput{
					appliedVersions: []string{},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})
}
