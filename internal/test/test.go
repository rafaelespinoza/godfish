package test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/rafaelespinoza/godfish"
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
func setup(driver godfish.Driver, testName string, stubs []testDriverStub, migrateTo string) (path string, err error) {
	path = "/tmp/godfish_test/drivers/" + driver.Name() + "/" + testName
	if err = os.MkdirAll(path, 0755); err != nil {
		return
	}
	if err = generateMigrationFiles(path, stubs); err != nil {
		return
	}
	if migrateTo != skipMigration {
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

	var truncate string
	switch driver.Name() {
	case "sqlite3":
		truncate = `DELETE FROM schema_migrations`
	default:
		truncate = `TRUNCATE TABLE schema_migrations`
	}
	if err = driver.Execute(truncate); err != nil {
		panic(err)
	}
	os.RemoveAll(path)
	driver.Close()
}

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
	content      MigrationContent
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
		var err error
		var params *godfish.MigrationParams

		var reversible bool
		if stub.content.Forward != "" && stub.content.Reverse != "" {
			reversible = true
		} else if stub.content.Forward == "" {
			panic(fmt.Errorf("test setup should have content in forward direction"))
		}

		if params, err = godfish.NewMigrationParams(strconv.Itoa(i), reversible, pathToTestDir); err != nil {
			return err
		}

		// replace migrations before generating files, to maintain control of
		// the timestamps, filenames, and migration content.
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

			filename := fmt.Sprintf(
				"%s/%s-%s-%s.sql",
				pathToTestDir, mig.Indirection().Label, mig.Version().String(), mig.Label(),
			)
			file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
			if err != nil {
				return err
			}
			defer file.Close()

			// this only works if the slice we're iterating through has
			// migrations where each Direction is in the order:
			// [forward, reverse]
			if j == 0 {
				if _, err = file.WriteString(stub.content.Forward); err != nil {
					return err
				}
				continue
			}
			if _, err = file.WriteString(stub.content.Reverse); err != nil {
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
