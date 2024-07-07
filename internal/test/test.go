package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

// RunDriverTests tests an implementation of the godfish.Driver interface.
// Callers should supply a set of valid queries q; most DBs can just use the
// DefaultQueries.
func RunDriverTests(t *testing.T, d godfish.Driver, q Queries) {
	for _, query := range []MigrationContent{q.CreateFoos, q.CreateBars, q.AlterFoos} {
		if query.Forward == "" || query.Reverse == "" {
			// Should also be valid queries, but the database will decide that.
			t.Fatalf("all %T fields should be non-empty", query)
		}
	}

	t.Run("Migrate", func(t *testing.T) { testMigrate(t, d, q) })
	t.Run("Info", func(t *testing.T) { testInfo(t, d, q) })
	t.Run("ApplyMigration", func(t *testing.T) { testApplyMigration(t, d, q) })
}

// Queries are named DB queries to use in the tests.
type Queries struct {
	CreateFoos MigrationContent
	CreateBars MigrationContent
	AlterFoos  MigrationContent
}

// DefaultQueries should be sufficient to use for most DBs in RunDriverTests.
var DefaultQueries = Queries{
	CreateFoos: MigrationContent{
		Forward: "CREATE TABLE foos (id int);",
		Reverse: "DROP TABLE foos;",
	},
	CreateBars: MigrationContent{
		Forward: "CREATE TABLE bars (id int);  ",
		Reverse: "DROP TABLE bars;",
	},
	AlterFoos: MigrationContent{
		Forward: "ALTER TABLE foos ADD COLUMN a varchar(255) ;",
		Reverse: "ALTER TABLE foos DROP COLUMN a;",
	},
}

type MigrationContent struct{ Forward, Reverse string }

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
	skipMigration = "-"
)

// setup prepares state before running a test.
func setup(t *testing.T, driver godfish.Driver, stubs []testDriverStub, migrateTo string) (path string) {
	t.Helper()

	path = t.TempDir()

	generateMigrationFiles(t, path, stubs)

	if migrateTo != skipMigration {
		err := godfish.Migrate(driver, path, true, migrateTo)
		if err != nil {
			t.Fatalf("Migrate failed during setup: %v", err)
		}
	}

	return
}

// teardown clears state after running a test.
func teardown(t *testing.T, driver godfish.Driver, path string, tablesToDrop ...string) {
	t.Helper()

	var err error
	if err = driver.Connect(mustDSN()); err != nil {
		t.Fatalf("error connecting to DB in teardown: %v", err)
	}

	for _, table := range tablesToDrop {
		if err = driver.Execute("DROP TABLE IF EXISTS " + table); err != nil {
			t.Fatalf("error dropping table in teardown: %v", err)
		}
	}

	var truncate string
	switch driver.Name() {
	case "stub":
		stub.Teardown(driver)
		truncate = `TRUNCATE TABLE schema_migrations`
	case "sqlite", "sqlite3":
		truncate = `DELETE FROM schema_migrations`
	default:
		truncate = `TRUNCATE TABLE schema_migrations`
	}
	if err = driver.Execute(truncate); err != nil {
		t.Fatalf("error executing query (%q) in teardown: %v", truncate, err)
	}
	_ = os.RemoveAll(path)
	if err := driver.Close(); err != nil {
		t.Fatalf("error closing driver in teardown: %v", err)
	}
}

func formattedTime(v string) internal.Version {
	out, err := internal.ParseVersion(v)
	if err != nil {
		panic(err)
	}
	return out
}

// testDriverStub encompasses some data to use with interface tests.
type testDriverStub struct {
	migration    internal.Migration
	content      MigrationContent
	indirectives struct{ forward, reverse internal.Indirection }
	version      internal.Version
}

func generateMigrationFiles(t *testing.T, pathToTestDir string, stubs []testDriverStub) {
	t.Helper()

	for i, stub := range stubs {
		var reversible bool
		if stub.content.Forward != "" && stub.content.Reverse != "" {
			reversible = true
		} else if stub.content.Forward == "" {
			t.Fatalf("error in generateMigrationFiles, stubs[%d] should have content in forward direction", i)
		}

		fwd, rev := stub.indirectives.forward, stub.indirectives.reverse
		params, err := internal.NewMigrationParams(strconv.Itoa(i), reversible, pathToTestDir, fwd.Label, rev.Label)
		if err != nil {
			t.Fatalf("error in generateMigrationFiles, stubs[%d] failure from NewMigrationParams: %v", i, err)
		}

		// replace migrations before generating files to maintain control of
		// the timestamps, filenames, and migration content.
		params.Forward = newMigrationStub(params.Forward, stub.version, fwd)
		if params.Reversible {
			params.Reverse = newMigrationStub(params.Reverse, stub.version, rev)
		}
		if err = params.GenerateFiles(); err != nil {
			t.Fatalf("error in generateMigrationFiles, stubs[%d] failure from GenerateFiles: %v", i, err)
		}

		for j, mig := range []internal.Migration{params.Forward, params.Reverse} {
			if j > 0 && !params.Reversible {
				continue
			}

			filename := filepath.Join(pathToTestDir, fmt.Sprintf(
				"%s-%s-%s.sql",
				mig.Indirection().Label, mig.Version().String(), mig.Label(),
			))

			file, err := os.OpenFile(filepath.Clean(filename), os.O_RDWR|os.O_CREATE, 0600)
			if err != nil {
				t.Fatalf("error in generateMigrationFiles, stubs[%d] failure from OpenFile: %v", i, err)
			}
			defer func() { _ = file.Close() }()

			// this only works if the slice we're iterating through has
			// migrations where each Direction is in the order:
			// [forward, reverse]
			if j == 0 {
				if _, err = file.WriteString(stub.content.Forward); err != nil {
					t.Fatalf("error in generateMigrationFiles, stubs[%d] failure writing forward migration: %v", i, err)
				}
				continue
			}
			if _, err = file.WriteString(stub.content.Reverse); err != nil {
				t.Fatalf("error in generateMigrationFiles, stubs[%d] failure writing reverse migration: %v", i, err)
			}
		}
	}
}

func newMigrationStub(mig internal.Migration, version internal.Version, ind internal.Indirection) internal.Migration {
	return stub.NewMigration(mig, version, ind)
}

// collectAppliedVersions uses the Driver's AppliedVersions method to retrieve
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
