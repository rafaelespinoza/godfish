// Package internal contains code for maintainers.
package internal

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish"
)

const baseTestOutputDir = "/tmp/godfish_test/driver"

// RunDriverTests tests an implementation of the godfish.Driver interface.
func RunDriverTests(t *testing.T, dsn godfish.DSN) {
	t.Helper()
	connParams := godfish.ConnectionParams{
		Encoding: "UTF8",
		Host:     os.Getenv("DB_HOST"),
		Name:     "godfish_test",
		Pass:     os.Getenv("DB_PASSWORD"),
		Port:     os.Getenv("DB_PORT"),
		User:     "godfish",
	}
	if connParams.Host == "" {
		connParams.Host = "localhost"
	}
	if err := dsn.Boot(connParams); err != nil {
		t.Fatal(err)
	}
	driver, err := godfish.NewDriver(dsn, nil)
	if err != nil {
		t.Fatal(err)
	}

	setup := func(testName string, migrate bool) (path string, stubs []testDriverStub, err error) {
		path = baseTestOutputDir + "/" + driver.Name() + "/" + testName
		if err = os.MkdirAll(path, 0755); err != nil {
			return
		}
		stubs, err = makeTestDriverStubs(
			path,
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
			return
		}
		if migrate {
			err = godfish.Migrate(driver, path, godfish.DirForward, "")
		}
		return
	}

	teardown := func(path string) {
		var err error
		if err = godfish.Migrate(driver, path, godfish.DirReverse, "00000101000000"); err != nil {
			t.Errorf("could not reset migrations; %v", err)
		}

		if _, err = driver.Connect(); err != nil {
			t.Error(err)
		}
		if err = driver.Execute(`TRUNCATE TABLE schema_migrations`); err != nil {
			t.Errorf("could not truncate schema_migrations table; %v", err)
		}
		os.RemoveAll(path)
		driver.Close()
	}

	// Tests for creating the schema migrations table are deliberately not
	// included. It should be called as needed by other library functions.

	t.Run("Migrate", func(t *testing.T) {
		path, _, err := setup(t.Name(), false)
		if err != nil {
			t.Errorf("could not setup test; %v", err)
			return
		}
		defer teardown(path)

		err = godfish.Migrate(driver, path, godfish.DirForward, "")
		if err != nil {
			t.Errorf("could not Migrate in %s Direction; %v", godfish.DirForward, err)
		}

		err = godfish.Migrate(driver, path, godfish.DirReverse, "")
		if err != nil {
			t.Errorf("could not Migrate in %s Direction; %v", godfish.DirReverse, err)
		}
	})

	t.Run("Info", func(t *testing.T) {
		path, _, err := setup(t.Name(), true)
		if err != nil {
			t.Errorf("could not setup test; %v", err)
			return
		}
		defer teardown(path)

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

	t.Run("DumpSchema", func(t *testing.T) {
		path, _, err := setup(t.Name(), true)
		if err != nil {
			t.Errorf("could not setup test; %v", err)
			return
		}
		defer teardown(path)

		err = godfish.DumpSchema(driver)
		if err != nil {
			t.Errorf(
				"test %q %q; could not dump schema; %v",
				driver.Name(), t.Name(), err,
			)
		}
	})

	t.Run("ApplyMigration", func(t *testing.T) {
		path, stubs, err := setup(t.Name(), false)
		if err != nil {
			t.Errorf("could not setup test; %v", err)
			return
		}
		defer teardown(path)

		versions := make([]string, len(stubs))
		for i, stub := range stubs {
			versions[i] = stub.version
		}

		applyMigrationTests := []struct {
			// onlyRollback means to reset the schema migrations state to empty
			// without migrating forward.
			onlyRollback bool
			// setupVersion is where to start before calling ApplyMigration.
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

		for i, test := range applyMigrationTests {
			var err error
			// Prepare state by rolling back all, then migrating forward a bit.
			if err = godfish.Migrate(driver, path, godfish.DirReverse, "00000101000000"); err != nil {
				t.Errorf("test [%d]; could not reset migrations; %v", i, err)
				continue
			}
			if !test.onlyRollback {
				if err = godfish.Migrate(driver, path, godfish.DirForward, test.setupVersion); err != nil {
					t.Errorf("test [%d]; could not setup migrations; %v", i, err)
					continue
				}
			}

			if err = godfish.ApplyMigration(
				driver,
				path,
				test.inputDirection,
				test.inputVersion,
			); err != nil && test.expectedError == nil {
				t.Errorf("test [%d]; could not apply migration; %v", i, err)
				continue
			} else if err == nil && test.expectedError != nil {
				t.Errorf("test [%d]; expected error but got none", i)
				continue
			} else if err != nil && err != test.expectedError {
				t.Errorf("test [%d]; unexpected error; %v", i, err)
				continue
			}

			// Collect output of AppliedVersions.
			// Reconnect here because ApplyMigration closes the connection.
			if _, err = driver.Connect(); err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			var actualVersions []string
			if appliedVersions, ierr := driver.AppliedVersions(); ierr != nil {
				t.Errorf(
					"test [%d]; could not retrieve applied versions; %v",
					i, ierr,
				)
				continue
			} else {
				for appliedVersions.Next() {
					var version string
					if scanErr := appliedVersions.Scan(&version); scanErr != nil {
						t.Errorf(
							"test [%d]; could not scan applied version; %v",
							i, scanErr,
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
					"test [%d]; got wrong output length %d, expected length to be %d",
					i, len(actualVersions), len(test.expectedAppliedVersions),
				)
				continue
			}
			for j, act := range actualVersions {
				if act != test.expectedAppliedVersions[j] {
					t.Errorf(
						"test [%d][%d]; wrong version; got %q, expected %q",
						i, j, act, test.expectedAppliedVersions[j],
					)
				}
			}
		}
	})
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
			if file, err = os.OpenFile(
				pathToTestDir+"/"+filename,
				os.O_RDWR|os.O_CREATE,
				0755,
			); err != nil {
				return nil, err
			}
			// this only works if the slice we're iterating through has
			// migrations where each Direction is in the order:
			// [forward, reverse]
			if j == 0 {
				if _, e := file.WriteString(content.forward); e != nil {
					err = e
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
