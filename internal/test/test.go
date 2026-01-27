// Package test is a test suite for a godfish.Driver.
package test

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

// DriverConnector is a godfish.Driver with connection management.
type DriverConnector interface {
	godfish.Driver
	Connect(dsn string) error
	Close() error
}

// RunDriverTests tests an implementation of the [godfish.Driver] interface.
// Callers should set the env var, DB_DSN.
func RunDriverTests(t *testing.T, d DriverConnector) {
	dsn := os.Getenv(internal.DSNKey)
	if dsn == "" {
		t.Fatalf("define env var %q for these tests", internal.DSNKey)
	}
	if err := d.Connect(dsn); err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	var q testdataQueries
	q.populateContents(t, d)

	t.Run("Migrate", func(t *testing.T) { testMigrate(t, d, q) })
	t.Run("Info", func(t *testing.T) { testInfo(t, d, q) })
	t.Run("ApplyMigration", func(t *testing.T) { testApplyMigration(t, d, q) })
	t.Run("UpdateSchemaMigrations", func(t *testing.T) { testUpdateSchemaMigrations(t, d) })
	t.Run("UpgradeSchemaMigrations", func(t *testing.T) { testUpgradeSchemaMigrations(t, d, q) })
	t.Run("Context", func(t *testing.T) { testContext(t, d) })
}

// testdataQueries are named DB testdataQueries to use in the tests.
type testdataQueries struct {
	CreateFoos migrationContent
	CreateBars migrationContent
	AlterFoos  migrationContent
}

// populateContents prepares the test suite by looking up testdata for a
// driver and hydrating q.
func (q *testdataQueries) populateContents(t *testing.T, d godfish.Driver) {
	t.Helper()

	// Calculate the absolute path to this file. Needed because this test suite
	// is called from multiple locations in the project, some at different
	// distances relative to the testdata directory.
	_, thisFile, _, _ := runtime.Caller(0)

	testdataSubdir := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", getTestdataSubdir(d))
	testdataRoot, err := os.OpenRoot(testdataSubdir)
	if err != nil {
		t.Fatalf("opening root at path %s: %s", testdataSubdir, err)
	}
	defer func() { _ = testdataRoot.Close() }()
	testdataFS := testdataRoot.FS()

	entries, err := fs.ReadDir(testdataFS, ".")
	if err != nil {
		t.Fatalf("reading directory entries at %s: %s", testdataSubdir, err)
	}
	const minEntriesExpected = 6
	if len(entries) != minEntriesExpected {
		t.Fatalf("too few entries; got %d, expected %d", len(entries), minEntriesExpected)
	}

	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		rawContents, err := fs.ReadFile(testdataFS, name)
		if err != nil {
			t.Fatalf("reading file contents of %s: %s", name, err)
		}
		contents := string(rawContents)
		switch name {
		case "forward-1234-alpha.sql":
			q.CreateFoos.Forward = contents
		case "forward-2345-bravo.sql":
			q.CreateBars.Forward = contents
		case "forward-3456-charlie.sql":
			q.AlterFoos.Forward = contents
		case "reverse-1234-alpha.sql":
			q.CreateFoos.Reverse = contents
		case "reverse-2345-bravo.sql":
			q.CreateBars.Reverse = contents
		case "reverse-3456-charlie.sql":
			q.AlterFoos.Reverse = contents
		default:
			t.Fatalf("unexpected migration filename %q", name)
		}
	}

	for _, query := range []migrationContent{q.CreateFoos, q.CreateBars, q.AlterFoos} {
		if query.Forward == "" || query.Reverse == "" {
			// Should also be valid queries, but the database will decide that.
			t.Fatalf("all %T fields should be non-empty", query)
		}
	}
}

type migrationContent struct{ Forward, Reverse string }

// Magic option values for test setup and teardown.
const (
	skipMigration = "-"
)

// setup prepares state before running a test.
func setup(t *testing.T, driver godfish.Driver, stubs []testDriverStub, migrateTo string, migrationsTable string) (path string) {
	t.Helper()

	path = t.TempDir()

	generateMigrationFiles(t, path, stubs)

	if migrateTo != skipMigration {
		err := godfish.Migrate(t.Context(), driver, os.DirFS(path), true, migrateTo, migrationsTable)
		if err != nil {
			t.Fatalf("Migrate failed during setup: %v", err)
		}
	}

	return
}

// teardown clears state after running a test.
func teardown(t *testing.T, driver godfish.Driver, path string, migrationsTable string, tablesToDrop ...string) {
	t.Helper()

	migrationsTable = cmp.Or(migrationsTable, internal.DefaultMigrationsTableName)

	var err error
	// Use a context with its own timeout, rather than directly use the test's
	// context to ensure that the caller's test cleanup does not prevent
	// this function from completing.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	for _, table := range tablesToDrop {
		if err = driver.Execute(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			t.Fatalf("error dropping table in teardown: %v", err)
		}
	}

	var truncate string
	switch driver.Name() {
	case "stub":
		stub.Teardown(driver)
		truncate = `TRUNCATE TABLE ` + migrationsTable
	case "sqlite", "sqlite3":
		truncate = `DELETE FROM ` + migrationsTable
	default:
		truncate = `TRUNCATE TABLE ` + migrationsTable
	}
	if err = driver.Execute(ctx, truncate); err != nil {
		t.Fatalf("error executing query (%q) in teardown: %v", truncate, err)
	}
	_ = os.RemoveAll(path)
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
	content      migrationContent
	indirectives struct{ forward, reverse internal.Indirection }
	version      internal.Version
}

func getTestdataSubdir(driver godfish.Driver) string {
	switch name := driver.Name(); name {
	case "cassandra", "sqlserver":
		return name
	default:
		return "default"
	}
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
				mig.Indirection.Label, mig.Version.String(), mig.Label,
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
	return *stub.NewMigration(mig, version, ind)
}

// collectAppliedMigrations uses the Driver's AppliedVersions method to retrieve
// and scan migration version. It opens a new connection in case the connection
// isn't already on the Driver, but it does close it afterwards.
//
// It uses defer rather than (*testing.T).Cleanup to ensure that any teardown
// functionality within may be called as soon as this function returns, rather
// than when the caller test is complete. This approach helps ensure fewer
// bugs in test support code, especially when it's called multiple times from
// the same test.
func collectAppliedMigrations(t *testing.T, driver godfish.Driver, migrationsTable string) (out []internal.Migration) {
	t.Helper()

	migrationsTable = cmp.Or(migrationsTable, internal.DefaultMigrationsTableName)

	appliedVersions, err := driver.AppliedVersions(t.Context(), migrationsTable)
	if err != nil {
		t.Fatalf("could not retrieve applied versions; %v", err)
	}

	defer func() {
		if cerr := appliedVersions.Close(); cerr != nil {
			slog.Warn("closing appliedVersions from func collectAppliedVersions", slog.Any("error", cerr))
		}
	}()

	for appliedVersions.Next() {
		// pass in the same types to Scan that are passed in the library's scanAppliedVersions function
		var version, label string
		var executedAt int64
		if err = appliedVersions.Scan(&version, &label, &executedAt); err != nil {
			t.Fatalf("could not scan applied versions; %v", err)
		}

		out = append(out, internal.Migration{
			Indirection: internal.Indirection{},
			Label:       label,
			Version:     formattedTime(version),
			ExecutedAt:  time.Unix(executedAt, 0),
		})
	}

	return
}

func testAppliedMigrations(t *testing.T, actual []internal.Migration, expectedVersions []string) {
	t.Helper()

	if len(actual) != len(expectedVersions) {
		t.Fatalf(
			"wrong number of applied migrations; got %d, expected %d",
			len(actual), len(expectedVersions),
		)
	}

	for i, act := range actual {
		version := act.Version.String()
		if version != expectedVersions[i] {
			t.Errorf(
				"index %d; wrong version; got %q, expected %q",
				i, version, expectedVersions[i],
			)
		}

		if act.Label == "" {
			t.Errorf("index %d; label for migration %q should be non-empty", i, version)
		}
		if act.ExecutedAt.IsZero() {
			t.Errorf("index %d; executed_at for migration %q should be non-empty", i, version)
		}
	}
}

type migrationsTableTestCase struct{ name, migrationsTable string }

var (
	okMigrationsTableTestCases = []migrationsTableTestCase{
		{name: "empty", migrationsTable: ""},
		{name: internal.DefaultMigrationsTableName, migrationsTable: internal.DefaultMigrationsTableName},
		{name: "custom", migrationsTable: "custom"},
	}

	invalidMigrationsTableTestCases = []migrationsTableTestCase{
		{name: "too many dots", migrationsTable: `too.many.dots`},
		{name: "injection - comment", migrationsTable: `foobars; --`},
		{name: "injection - query", migrationsTable: `foobars' OR '1'='1`},
		{name: "starts with number", migrationsTable: `123bad`},
		{name: "invalid characters", migrationsTable: "foo\x00_bar"},
	}
)
