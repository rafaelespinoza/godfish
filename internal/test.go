// Package internal contains code for maintainers.
package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

// RunDriverTests tests an implementation of the godfish.Driver interface.
func RunDriverTests(t *testing.T, driver godfish.Driver) {
	// Tests for creating the schema migrations table are deliberately not
	// included. It should be called as needed by other library functions.

	t.Run("Migrate", func(t *testing.T) {
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
		path, err := setup(driver, t.Name(), stubs, _SkipMigration)
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
	})

	t.Run("Info", func(t *testing.T) {
		path, err := setup(driver, t.Name(), _DefaultTestDriverStubs, "34560102030405")
		if err != nil {
			t.Errorf("could not setup test; %v", err)
			return
		}
		defer teardown(driver, path, "foos", "bars")

		t.Run("forward", func(t *testing.T) {
			err := godfish.Info(driver, path, godfish.DirForward, "")
			if err != nil {
				t.Errorf(
					"could not output info in %s Direction; %v",
					godfish.DirForward, err,
				)
			}
		})
		t.Run("reverse", func(t *testing.T) {
			err := godfish.Info(driver, path, godfish.DirReverse, "")
			if err != nil {
				t.Errorf(
					"could not output info in %s Direction; %v",
					godfish.DirReverse, err,
				)
			}
		})
	})

	t.Run("ApplyMigration", func(t *testing.T) {
		// testSetupState describes the state of the database before calling
		// ApplyMigration.
		type testSetupState struct {
			// migrateTo is the version that the DB should be at.
			migrateTo string
			// stubs is a list of stubbed migration data to populate the DB.
			stubs []testDriverStub
		}
		// testInput is the set of arguments to pass to ApplyMigration.
		type testInput struct {
			// direction is the direction to migrate.
			direction godfish.Direction
			// version is where to go when calling ApplyMigration.
			version string
		}
		// expectedOutput is a set of arguments to describe the expectations.
		type expectedOutput struct {
			// appliedVersions is where you should be after calling
			// ApplyMigration.
			appliedVersions []string
			// err says that we expect some error to happen, and the code should
			// handle it.
			err bool
		}

		// testCase combines setup, inputs and expectations for one test case.
		type testCase struct {
			setup    *testSetupState
			input    *testInput
			expected *expectedOutput
		}

		applyMigrationTests := []testCase{
			// setup state: no migration files
			{
				setup: &testSetupState{
					stubs: []testDriverStub{},
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{},
					err:             true, // no version found
				},
			},
			{
				setup: &testSetupState{
					stubs: []testDriverStub{},
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{},
					err:             true, // no version found
				},
			},
			// setup state: do not migrate
			{
				setup: &testSetupState{
					migrateTo: _SkipMigration,
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: _SkipMigration,
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{},
					err:             true, // no version found
				},
			},
			// setup state: go forward partway or all of the way.
			// test forward
			{
				setup: &testSetupState{
					migrateTo: "12340102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "23450102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no version found
				},
			},
			// test reverse
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "23450102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "34560102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "23450102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "12340102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405"},
					err:             true, // no version found
				},
			},
			// test reverse, available migration at end of range.
			{
				setup: &testSetupState{
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
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "34560102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405"},
				},
			},
			// migration file does not exist
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "43210102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "43210102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					// target migration only exists in forward direction.
					// target > available.
					version: "34560102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs:     _DefaultTestDriverStubs,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					// target migration only exists in forward direction.
					// target < available.
					version: "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // no files found
				},
			},
			// test error during migration execution
			{
				setup: &testSetupState{
					migrateTo: "34560102030405",
					stubs: append(_DefaultTestDriverStubs, testDriverStub{
						content: struct{ forward, reverse string }{
							forward: "invalid SQL",
						},
						version: formattedTime("45670102030405"),
					}),
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "45670102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
					err:             true, // error executing SQL
				},
			},
			// test short version
			{
				setup: &testSetupState{
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
				input: &testInput{
					direction: godfish.DirForward,
					version:   "3456",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"1234", "2345", "3456"},
				},
			},
			// test migration with multiple statements
			{
				setup: &testSetupState{
					migrateTo: _SkipMigration,
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
				input: &testInput{
					direction: godfish.DirForward,
					version:   "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: _SkipMigration,
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
				input: &testInput{
					direction: godfish.DirForward,
					version:   "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{},
					err:             true,
				},
			},
			// test alternative filenames, directions: migrate, rollback
			{
				setup: &testSetupState{
					migrateTo: _SkipMigration,
					stubs:     _StubsWithMigrateRollbackIndirectives,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "12340102030405",
					stubs:     _StubsWithMigrateRollbackIndirectives,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{},
				},
			},
			// test alternative filenames, directions: up, down
			{
				setup: &testSetupState{
					migrateTo: _SkipMigration,
					stubs:     _StubsWithUpDownIndirectives,
				},
				input: &testInput{
					direction: godfish.DirForward,
					version:   "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{"12340102030405"},
				},
			},
			{
				setup: &testSetupState{
					migrateTo: "12340102030405",
					stubs:     _StubsWithUpDownIndirectives,
				},
				input: &testInput{
					direction: godfish.DirReverse,
					version:   "12340102030405",
				},
				expected: &expectedOutput{
					appliedVersions: []string{},
				},
			},
		}

		for i, test := range applyMigrationTests {
			func(i int, test testCase) {
				pathToFiles, err := setup(
					driver,
					t.Name()+"/"+fmt.Sprintf("%02d", i),
					test.setup.stubs,
					test.setup.migrateTo,
				)
				if err != nil {
					t.Errorf("could not setup test; %v", err)
					return
				}
				defer teardown(driver, pathToFiles, "foos", "bars")

				if err := godfish.ApplyMigration(
					driver,
					pathToFiles,
					test.input.direction,
					test.input.version,
				); err != nil && !test.expected.err {
					t.Errorf("test [%d]; could not apply migration; %v", i, err)
					return
				} else if err == nil && test.expected.err {
					t.Errorf("test [%d]; expected error but got none", i)
					return
				}

				actualVersions, err := collectAppliedVersions(driver)
				if err != nil {
					t.Fatalf("test [%d]; %v", i, err)
				}

				err = testAppliedVersions(actualVersions, test.expected.appliedVersions)
				if err != nil {
					t.Errorf("test [%d]; %v", i, err)
				}
			}(i, test)
		}
	})
}

const dsnKey = "DB_DSN"

func mustDSN() string {
	dsn := os.Getenv(dsnKey)
	if dsn == "" {
		panic("empty environment variable " + dsnKey)
	}
	return dsn
}

// Magic option values for test setup and teardown.
const (
	_SkipMigration = "-"
)

// setup prepares state before running a test.
func setup(driver godfish.Driver, testName string, stubs []testDriverStub, migrateTo string) (path string, err error) {
	path = "/tmp/godfish_test/drivers/" + driver.Name() + "/" + testName
	if err = os.MkdirAll(path, 0755); err != nil {
		return
	}
	if err = generateMigrationFiles(path, stubs); err != nil {
		return
	}
	if migrateTo != _SkipMigration {
		err = godfish.Migrate(driver, path, godfish.DirForward, migrateTo)
	}
	return
}

// teardown clears state after running a test.
func teardown(driver godfish.Driver, path string, tablesToDrop ...string) {
	var err error
	if err = driver.Connect(mustDSN()); err != nil {
		panic(err)
	}

	for _, table := range tablesToDrop {
		if err = driver.Execute("DROP TABLE IF EXISTS " + table); err != nil {
			panic(err)
		}
	}
	// keep the stub test driver simple, just reset it.
	if d, ok := driver.(*stub.Driver); ok {
		d.Teardown()
	}
	if err = driver.Execute(`TRUNCATE TABLE schema_migrations`); err != nil {
		panic(err)
	}
	os.RemoveAll(path)
	driver.Close()
}

var (
	_DefaultTestDriverStubs = []testDriverStub{
		{
			content: struct{ forward, reverse string }{
				forward: `CREATE TABLE foos (id int);`,
			},
			version: formattedTime("12340102030405"),
		},
		{
			content: struct{ forward, reverse string }{
				forward: `ALTER TABLE foos ADD COLUMN a varchar(255);`,
				reverse: `ALTER TABLE foos DROP COLUMN a;`,
			},
			version: formattedTime("23450102030405"),
		},
		{
			content: struct{ forward, reverse string }{
				forward: `CREATE TABLE bars (id int);`,
			},
			version: formattedTime("34560102030405"),
		},
	}
	_StubsWithMigrateRollbackIndirectives = []testDriverStub{
		{
			content: struct{ forward, reverse string }{
				forward: "CREATE TABLE foos (id int);",
				reverse: "DROP TABLE foos;",
			},
			indirectives: struct{ forward, reverse godfish.Indirection }{
				forward: godfish.Indirection{Label: "migrate"},
				reverse: godfish.Indirection{Label: "rollback"},
			},
			version: formattedTime("12340102030405"),
		},
	}
	_StubsWithUpDownIndirectives = []testDriverStub{
		{
			content: struct{ forward, reverse string }{
				forward: "CREATE TABLE foos (id int);",
				reverse: "DROP TABLE foos;",
			},
			indirectives: struct{ forward, reverse godfish.Indirection }{
				forward: godfish.Indirection{Label: "up"},
				reverse: godfish.Indirection{Label: "down"},
			},
			version: formattedTime("12340102030405"),
		},
	}
)

type formattedTime string

func (v formattedTime) Before(u godfish.Version) bool {
	w := u.(formattedTime) // potential panic intended, keep tests simple
	return string(v) < string(w)
}
func (v formattedTime) String() string { return string(v) }
func (v formattedTime) Value() int64 {
	i, e := strconv.ParseInt(v.String()[:4], 10, 64)
	if e != nil {
		panic(e)
	}
	return i
}

var _ godfish.Version = (*formattedTime)(nil)

// testDriverStub encompasses some data to use with interface tests.
type testDriverStub struct {
	migration    godfish.Migration
	content      struct{ forward, reverse string }
	indirectives struct{ forward, reverse godfish.Indirection }
	version      godfish.Version
}

// migrationStub is a godfish.Migration implementation that's used to override
// the version field so that the generated filename is "unique".
type migrationStub struct {
	indirection godfish.Indirection
	label       string
	version     godfish.Version
}

var _ godfish.Migration = (*migrationStub)(nil)

func newMigrationStub(mig godfish.Migration, version godfish.Version, ind godfish.Indirection) godfish.Migration {
	stub := migrationStub{
		indirection: mig.Indirection(),
		label:       mig.Label(),
		version:     version,
	}
	if ind.Label != "" {
		stub.indirection.Label = ind.Label
	}
	return &stub
}

func (m *migrationStub) Indirection() godfish.Indirection { return m.indirection }
func (m *migrationStub) Label() string                    { return m.label }
func (m *migrationStub) Version() godfish.Version         { return m.version }

func generateMigrationFiles(pathToTestDir string, stubs []testDriverStub) error {
	for i, stub := range stubs {
		var file *os.File
		var err error
		var params *godfish.MigrationParams
		defer func() {
			if file != nil {
				file.Close()
			}
		}()
		var reversible bool
		if stub.content.forward != "" && stub.content.reverse != "" {
			reversible = true
		} else if stub.content.forward == "" {
			panic(fmt.Errorf("test setup should have content in forward direction"))
		}
		name := fmt.Sprintf("%d", i)
		if params, err = godfish.NewMigrationParams(
			name,
			reversible,
			pathToTestDir,
		); err != nil {
			return err
		}
		// replace migrations before generating files, so we can control the
		// timestamps, filenames, and migration content.
		params.Forward = newMigrationStub(params.Forward, stub.version, stub.indirectives.forward)
		if params.Reversible {
			params.Reverse = newMigrationStub(params.Reverse, stub.version, stub.indirectives.reverse)
		}
		if err = params.GenerateFiles(); err != nil {
			return err
		}

		for j, mig := range []godfish.Migration{params.Forward, params.Reverse} {
			if j > 0 && !params.Reversible {
				continue
			}

			if file, err = os.OpenFile(
				fmt.Sprintf(
					"%s/%s-%s-%s.sql",
					pathToTestDir, mig.Indirection().Label, mig.Version().String(), mig.Label(),
				),
				os.O_RDWR|os.O_CREATE,
				0755,
			); err != nil {
				return err
			}
			defer file.Close()
			// this only works if the slice we're iterating through has
			// migrations where each Direction is in the order:
			// [forward, reverse]
			if j == 0 {
				if _, e := file.WriteString(stub.content.forward); e != nil {
					err = e
					return err
				}
				continue
			}
			if file.WriteString(stub.content.reverse); err != nil {
				return err
			}
		}
	}

	return nil
}

// collectAppliedVersions uses the Driver's AppliedVersions method to retreive
// and scan migration version. It opens a new connection in case the connection
// isn't already on the Driver, but it does close it afterwards.
func collectAppliedVersions(driver godfish.Driver) (out []string, err error) {
	// Collect output of AppliedVersions.
	// Reconnect here because ApplyMigration closes the connection.
	if err = driver.Connect(mustDSN()); err != nil {
		return
	}
	defer driver.Close()

	appliedVersions, err := driver.AppliedVersions()
	if err != nil {
		err = fmt.Errorf("could not retrieve applied versions; %v", err)
		return
	}
	defer appliedVersions.Close()

	for appliedVersions.Next() {
		var version string
		if err = appliedVersions.Scan(&version); err != nil {
			err = fmt.Errorf("could not scan applied versions; %v", err)
			return
		}
		out = append(out, version)
	}

	return
}

func testAppliedVersions(actual, expected []string) error {
	if len(actual) != len(expected) {
		return fmt.Errorf(
			"wrong output length; got %d, expected %d",
			len(actual), len(expected),
		)
	}
	for i, version := range actual {
		if version != expected[i] {
			return fmt.Errorf(
				"index %d; wrong version; got %q, expected %q",
				i, version, expected[i],
			)
		}
	}
	return nil
}
