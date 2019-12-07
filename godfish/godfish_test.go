package godfish_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"bitbucket.org/rafaelespinoza/godfish/godfish"
)

const (
	testDBName        = "godfish_test"
	baseTestOutputDir = "/tmp/godfish_test"
)

var DriversToTest = []struct {
	name      string
	dsnParams godfish.DSNParams
}{
	{
		name: "postgres",
		dsnParams: godfish.PostgresParams{
			Encoding: "UTF8",
			Host:     "localhost",
			Name:     testDBName,
			Pass:     os.Getenv("DB_PASSWORD"),
			Port:     "5432",
		},
	},
}

func TestMain(m *testing.M) {
	os.MkdirAll(baseTestOutputDir, 0755)
	m.Run()
	os.RemoveAll(baseTestOutputDir)
}

func TestMigrationParams(t *testing.T) {
	var testDir *os.File
	var mig *godfish.MigrationParams
	var err error
	if testDir, err = os.Open(baseTestOutputDir); err != nil {
		t.Error(err)
		return
	}
	if mig, err = godfish.NewMigrationParams("foo", true, testDir); err != nil {
		t.Error(err)
		return
	}
	if mig.Forward.Direction() != godfish.DirForward {
		t.Errorf(
			"wrong Direction; expected %s, got %s",
			godfish.DirForward, mig.Forward.Direction(),
		)
	}
	if mig.Reverse.Direction() != godfish.DirReverse {
		t.Errorf(
			"wrong Direction; expected %s, got %s",
			godfish.DirReverse, mig.Reverse.Direction(),
		)
	}
	migrations := []godfish.Migration{mig.Forward, mig.Reverse}
	for _, mig := range migrations {
		if mig.Name() != "foo" {
			t.Errorf(
				"wrong Name; expected %s, got %s",
				"foo", mig.Name(),
			)
		}
		if mig.Timestamp().IsZero() {
			t.Error("got empty Timestamp")
		}
	}

	var filesBefore, filesAfter []string
	if filesBefore, err = testDir.Readdirnames(0); err != nil {
		t.Error(err)
		return
	}
	if err = mig.GenerateFiles(); err != nil {
		t.Error(err)
		return
	}
	if filesAfter, err = testDir.Readdirnames(0); err != nil {
		t.Error(err)
		return
	}
	if len(filesAfter)-len(filesBefore) != 2 {
		t.Errorf(
			"expected to generate 2 files, got %d",
			len(filesAfter)-len(filesBefore),
		)
		return
	}
	expectedDirections := []string{"reverse", "forward"}
	for i, name := range filesAfter {
		patt := fmt.Sprintf("[0-9]*.%s.foo.sql", expectedDirections[i])
		if match, err := filepath.Match(patt, name); err != nil {
			t.Error(err)
			return
		} else if !match {
			t.Errorf(
				"expected filename %q to match pattern %q",
				name, patt,
			)
		}
	}
}

func TestDriver(t *testing.T) {
	for i, test := range DriversToTest {
		pathToTestDir := baseTestOutputDir + "/" + t.Name() + "/" + test.name
		if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
			t.Errorf(
				"test [%d]; error creating test directory %s",
				i, pathToTestDir,
			)
			return
		}
		driver, err := godfish.NewDriver(test.name, test.dsnParams)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			if err := truncateSchemaMigrations(driver); err != nil {
				t.Errorf(
					"test [%d]; could not truncate schema_migrations table for %q; %v",
					i, driver.Name(), err,
				)
			}
		}()
		stubs, err := makeTestDriverStubs(
			pathToTestDir,
			[]contentStub{
				{
					forward: `CREATE TABLE foos (id int);`,
					reverse: `DROP TABLE foos;`,
				},
				{
					forward: `ALTER TABLE foos ADD COLUMN a varchar(255);`,
					reverse: `ALTER TABLE foos DROP COLUMN a;`,
				},
				{
					forward: `CREATE TABLE bars (id int);`,
					reverse: `DROP TABLE bars;`,
				},
			},
			[]string{"12340102030405", "23450102030405", "34560102030405"},
		)
		if err != nil {
			panic(err)
		}
		// test CreateSchemaMigrationsTable
		if err = godfish.CreateSchemaMigrationsTable(driver); err != nil {
			t.Errorf(
				"test [%d]; could not create schema migrations table for driver %q; %v",
				i, driver.Name(), err,
			)
			continue
		}

		// test ApplyMigration
		for j, stub := range stubs {
			err = godfish.ApplyMigration(
				driver,
				pathToTestDir,
				stub.migration.Direction(),
				stub.migration.Timestamp().Format(godfish.TimeFormat),
			)
			if err != nil {
				t.Errorf(
					"test [%d][%d]; driver %q; could not apply migration; %v",
					i, j, driver.Name(), err,
				)
				return
			}
		}

		// test Migrate in forward direction
		if err = godfish.Migrate(driver, pathToTestDir, godfish.DirForward, ""); err != nil {
			t.Errorf(
				"test [%d]; driver %q; could not Migrate in %s Direction; %v",
				i, driver.Name(), godfish.DirForward, err,
			)
			return
		}

		// test Info in forward direction
		fmt.Printf(
			"-- %s test [%d] calling Info %s %s\n",
			t.Name(), i, driver.Name(), godfish.DirForward,
		)
		if err = godfish.Info(driver, pathToTestDir, godfish.DirForward, ""); err != nil {
			t.Errorf(
				"test [%d]; could not output info in %s Direction; %v",
				i, godfish.DirForward, err,
			)
			return
		}

		// test Info in reverse direction
		fmt.Printf(
			"-- %s test [%d] calling Info %s %s\n",
			t.Name(), i, driver.Name(), godfish.DirReverse,
		)
		if err = godfish.Info(driver, pathToTestDir, godfish.DirReverse, ""); err != nil {
			t.Errorf(
				"test [%d]; could not output info in %s Direction; %v",
				i, godfish.DirReverse, err,
			)
			return
		}

		// test DumpSchema
		if err = godfish.DumpSchema(driver); err != nil {
			t.Errorf("test [%d]; could not dump schema; %v", i, err)
			return
		}

		// test Migrate in reverse direction
		if err = godfish.Migrate(driver, pathToTestDir, godfish.DirReverse, ""); err != nil {
			t.Errorf(
				"test [%d]; driver %q; could not Migrate in %s Direction; %v",
				i, driver.Name(), godfish.DirReverse, err,
			)
			return
		}
		versions := make([]string, len(stubs))
		for i, stub := range stubs {
			versions[i] = stub.version
		}

		// test ApplyMigration again, this time by going partway and testing
		// intermediate state.
		applyMigrationTests := []struct {
			// onlyRollback means to reset the schema migrations state to empty
			// without migrating forward.
			onlyRollback bool
			// setupVersion is where to start before calling ApplyMigration.
			// Pass "-1" to not migrate forward at all during setup.
			setupVersion   string
			inputDirection godfish.Direction
			// inputVersion is where to go when calling ApplyMigration.
			inputVersion string
			// expectedAppliedVersions is where you should be after calling
			// ApplyMigration.
			expectedAppliedVersions []string
			// expectedError names a specific error value, if we expect one,
			// while calling ApplyMigration.
			expectedError error
		}{
			// just rollback during set up
			{
				onlyRollback:            true,
				inputDirection:          godfish.DirForward,
				inputVersion:            "",
				expectedAppliedVersions: []string{"12340102030405"},
			},
			{
				onlyRollback:            true,
				inputDirection:          godfish.DirReverse,
				inputVersion:            "",
				expectedAppliedVersions: []string{},
				expectedError:           godfish.ErrNoVersionFound,
			},
			// now setup by going forward part or all of the way
			{
				setupVersion:            versions[0],
				inputDirection:          godfish.DirForward,
				inputVersion:            "",
				expectedAppliedVersions: []string{"12340102030405", "23450102030405"},
			},
			{
				setupVersion:            versions[1],
				inputDirection:          godfish.DirForward,
				inputVersion:            "",
				expectedAppliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
			},
			{
				setupVersion:            versions[2],
				inputDirection:          godfish.DirForward,
				inputVersion:            "",
				expectedAppliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				expectedError:           godfish.ErrNoVersionFound,
			},
			{
				setupVersion:            versions[2],
				inputDirection:          godfish.DirReverse,
				inputVersion:            "",
				expectedAppliedVersions: []string{"12340102030405", "23450102030405"},
			},
			{
				setupVersion:            versions[1],
				inputDirection:          godfish.DirReverse,
				inputVersion:            "",
				expectedAppliedVersions: []string{"12340102030405"},
			},
			{
				setupVersion:            versions[0],
				inputDirection:          godfish.DirReverse,
				inputVersion:            "",
				expectedAppliedVersions: []string{},
			},
			{
				setupVersion:            versions[2],
				inputDirection:          godfish.DirForward,
				inputVersion:            "43210102030405",
				expectedAppliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				expectedError:           godfish.ErrNoFilesFound,
			},
			{
				setupVersion:            versions[2],
				inputDirection:          godfish.DirReverse,
				inputVersion:            "43210102030405",
				expectedAppliedVersions: []string{"12340102030405", "23450102030405", "34560102030405"},
				expectedError:           godfish.ErrNoFilesFound,
			},
		}

		for j, test := range applyMigrationTests {
			var err error
			// Prepare state by rolling back all, then migrating forward a bit.
			if err = godfish.Migrate(driver, pathToTestDir, godfish.DirReverse, "00000101000000"); err != nil {
				t.Errorf(
					"test [%d][%d]; driver %q; could not reset migrations; %v",
					i, j, driver.Name(), err,
				)
				return
			}
			if !test.onlyRollback {
				if err = godfish.Migrate(driver, pathToTestDir, godfish.DirForward, test.setupVersion); err != nil {
					t.Errorf(
						"test [%d][%d]; driver %q; could not setup migrations; %v",
						i, j, driver.Name(), err,
					)
					return
				}
			}

			if err = godfish.ApplyMigration(
				driver,
				pathToTestDir,
				test.inputDirection,
				test.inputVersion,
			); err != nil && test.expectedError == nil {
				t.Errorf(
					"test [%d][%d]; driver %q; could not apply migration; %v",
					i, j, driver.Name(), err,
				)
				return
			} else if err == nil && test.expectedError != nil {
				t.Errorf(
					"test [%d][%d]; driver %q; expected error but got none",
					i, j, driver.Name(),
				)
				continue
			} else if err != nil && err != test.expectedError {
				t.Errorf(
					"test [%d][%d]; driver %q; expected error, got one, but not what we thought it was",
					i, j, driver.Name(),
				)
			}

			// Collect output of AppliedVersions. Need to reconnect here because
			// ApplyMigration closes the connection.
			if _, err = driver.Connect(); err != nil {
				t.Errorf(
					"test [%d][%d]; driver %q; could not connect to DB; %v",
					i, j, driver.Name(), err,
				)
				return
			}
			defer driver.Close()
			var actualVersions []string
			if appliedVersions, ierr := driver.AppliedVersions(); ierr != nil {
				t.Errorf(
					"test [%d][%d]; driver %q; could not retrieve applied versions; %v",
					i, j, driver.Name(), ierr,
				)
				return
			} else {
				for appliedVersions.Next() {
					var version string
					if scanErr := appliedVersions.Scan(&version); scanErr != nil {
						t.Errorf(
							"test [%d][%d]; driver %q; could not scan applied version; %v",
							i, j, driver.Name(), scanErr,
						)
						err = scanErr
						return
					}
					actualVersions = append(actualVersions, version)
				}
			}
			// Finally, test the output.
			if len(actualVersions) != len(test.expectedAppliedVersions) {
				t.Errorf(
					"test [%d][%d]; got wrong output length %d, expected length to be %d",
					i, j, len(actualVersions), len(test.expectedAppliedVersions),
				)
				continue
			}
			for k, act := range actualVersions {
				if act != test.expectedAppliedVersions[k] {
					t.Errorf(
						"test [%d][%d][%d]; wrong version; got %q, expected %q",
						i, j, k, act, test.expectedAppliedVersions[k],
					)
				}
			}
		}
	}
}

// testDriverStub encompasses some data to use with interface tests.
type testDriverStub struct {
	migration godfish.Migration
	content   contentStub
	version   string
}

// migrationStub is a godfish.Migration implementation that's used to override
// the timestamp field, so that the generated filename is "unique".
type migrationStub struct {
	direction godfish.Direction
	name      string
	timestamp time.Time
}

var _ godfish.Migration = (*migrationStub)(nil)

func newMigrationStub(mig godfish.Migration, timestamp time.Time) godfish.Migration {
	return &migrationStub{
		direction: mig.Direction(),
		name:      mig.Name(),
		timestamp: timestamp,
	}
}

func (m *migrationStub) Direction() godfish.Direction { return m.direction }
func (m *migrationStub) Name() string                 { return m.name }
func (m *migrationStub) Timestamp() time.Time         { return m.timestamp }

type contentStub struct {
	forward string
	reverse string
}

// makeTestDriverStubs generates some stub migrations in both directions,
// generates the files and populates each with some dummy content for
// contentStub input. The versions input should be a list of formattable
// timestamps so we can control the versions of each output.
func makeTestDriverStubs(pathToTestDir string, contents []contentStub, versions []string) ([]testDriverStub, error) {
	testDir, err := os.Open(pathToTestDir)
	if err != nil {
		return nil, err
	}
	out := make([]testDriverStub, len(contents))
	for i, content := range contents {
		var filename string
		var file *os.File
		var err error
		var params *godfish.MigrationParams
		var timestamp time.Time
		defer func() {
			if file != nil {
				file.Close()
			}
		}()
		name := fmt.Sprintf("%d", i)
		if params, err = godfish.NewMigrationParams(
			name,
			true,
			testDir,
		); err != nil {
			return nil, err
		}
		// replace migrations before generating files, so we can control the
		// timestamps, filenames, and migration content.
		if timestamp, err = time.Parse(godfish.TimeFormat, versions[i]); err != nil {
			return nil, fmt.Errorf("could not parse stubbed timestamp, %v", err)
		}
		params.Forward = newMigrationStub(params.Forward, timestamp)
		params.Reverse = newMigrationStub(params.Reverse, timestamp)
		if err = params.GenerateFiles(); err != nil {
			return nil, err
		}

		for j, mig := range []godfish.Migration{params.Forward, params.Reverse} {
			if filename, err = godfish.Basename(mig); err != nil {
				return nil, err
			}
			if file, err = os.Open(pathToTestDir + "/" + filename); err != nil {
				return nil, err
			}
			// this only works if the slice we're iterating through has
			// migrations where each Direction is in the order:
			// [forward, reverse]
			if j == 0 {
				if file.WriteString(content.forward); err != nil {
					return nil, err
				}
				out[i] = testDriverStub{
					migration: params.Forward,
					content:   content,
					version:   timestamp.Format(godfish.TimeFormat),
				}
				continue
			}
			if file.WriteString(content.reverse); err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}

func truncateSchemaMigrations(driver godfish.Driver) (err error) {
	switch driver.Name() {
	case "postgres":
		cmd := exec.Command(
			"psql",
			testDBName, "-e", "-c", "TRUNCATE TABLE schema_migrations CASCADE",
		)
		_, err = cmd.Output()
		if val, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(val.Stderr))
			err = val
		}
	default:
		err = fmt.Errorf("unknown Driver %q", driver.Name())
	}
	return
}
