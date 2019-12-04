package godfish

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
)

type StubDB struct {
	dsn             StubDSN
	connection      *sql.DB
	appliedVersions AppliedVersions
}

var _ Driver = (*StubDB)(nil)

func (d *StubDB) Name() string                                       { return "stub" }
func (d *StubDB) Connect() (*sql.DB, error)                          { return d.connection, nil }
func (d *StubDB) Close() error                                       { return nil }
func (d *StubDB) DSNParams() DSNParams                               { return d.dsn }
func (d *StubDB) CreateSchemaMigrationsTable() error                 { return nil }
func (d *StubDB) DumpSchema() error                                  { return nil }
func (d *StubDB) Execute(q string, a ...interface{}) error           { return nil }
func (d *StubDB) UpdateSchemaMigrations(u Direction, v string) error { return nil }
func (d *StubDB) AppliedVersions() (AppliedVersions, error)          { return d.appliedVersions, nil }

type StubDSN struct{}

func (d StubDSN) String() string { return "this://is.a/test" }

var _ DSNParams = (*StubDSN)(nil)

type StubAppliedVersions struct {
	counter  int
	versions []string
}

var _ AppliedVersions = (*StubAppliedVersions)(nil)

func (r *StubAppliedVersions) Close() error { return nil }
func (r *StubAppliedVersions) Next() bool   { return r.counter < len(r.versions) }
func (r *StubAppliedVersions) Scan(dest ...interface{}) error {
	var out *string
	if s, ok := dest[0].(*string); !ok {
		return fmt.Errorf("pass in *string; got %T", s)
	} else if !r.Next() {
		return fmt.Errorf("no more results")
	} else {
		out = s
	}
	*out = r.versions[r.counter]
	r.counter++
	return nil
}

const baseTestOutputDir = "/tmp/godfish_test"

func TestListMigrationsToApply(t *testing.T) {
	pathToTestDir := baseTestOutputDir + "/" + t.Name()
	if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
		t.Errorf("error creating test directory %s", pathToTestDir)
		return
	}
	defer os.RemoveAll(pathToTestDir)

	type testCase struct {
		direction       Direction
		applied         []Migration
		available       []Migration
		finishAtVersion string
		expectedOut     []Migration
		expectError     bool
	}

	tests := []testCase{
		{
			direction: DirForward,
			applied: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
				mustMakeMigration("23450102030405", DirForward, "bravo"),
			},
			available: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
				mustMakeMigration("23450102030405", DirForward, "bravo"),
			},
			expectedOut: []Migration{},
		},
		{
			direction: DirForward,
			applied: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
			},
			available: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
				mustMakeMigration("23450102030405", DirForward, "bravo"),
			},
			expectedOut: []Migration{
				mustMakeMigration("23450102030405", DirForward, "bravo"),
			},
		},
		{
			direction: DirForward,
			applied:   []Migration{},
			available: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
				mustMakeMigration("23450102030405", DirForward, "bravo"),
			},
			expectedOut: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
				mustMakeMigration("23450102030405", DirForward, "bravo"),
			},
		},
		{
			direction: DirReverse,
			applied: []Migration{
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
				mustMakeMigration("23450102030405", DirReverse, "bravo"),
			},
			available: []Migration{
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
				mustMakeMigration("23450102030405", DirReverse, "bravo"),
			},
			expectedOut: []Migration{
				mustMakeMigration("23450102030405", DirReverse, "bravo"),
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
			},
		},
		{
			direction: DirReverse,
			applied: []Migration{
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
			},
			available: []Migration{
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
				mustMakeMigration("23450102030405", DirReverse, "bravo"),
			},
			expectedOut: []Migration{
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
			},
		},
		{
			direction: DirReverse,
			applied:   []Migration{},
			available: []Migration{
				mustMakeMigration("12340102030405", DirReverse, "alpha"),
				mustMakeMigration("23450102030405", DirReverse, "bravo"),
			},
			expectedOut: []Migration{},
		},
		{
			direction:   DirForward,
			applied:     []Migration{},
			available:   []Migration{},
			expectedOut: []Migration{},
		},
		{
			direction:   DirReverse,
			applied:     []Migration{},
			available:   []Migration{},
			expectedOut: []Migration{},
		},
		{
			applied:     []Migration{},
			available:   []Migration{},
			expectError: true,
		},
		{
			direction: DirForward,
			applied:   []Migration{},
			available: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
				mustMakeMigration("23450102030405", DirForward, "bravo"),
				mustMakeMigration("34560102030405", DirForward, "charlie"),
			},
			finishAtVersion: "12340102030405",
			expectedOut: []Migration{
				mustMakeMigration("12340102030405", DirForward, "alpha"),
			},
		},
	}

	// runTest is the body of the tests loop. It exists because there is some
	// setup and teardown we want to do w/o complicating the control flow.
	runTest := func(i int, test testCase) error {
		pathToTestDir := fmt.Sprintf("%s/%d", pathToTestDir, i)
		// setup
		if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
			panic(err)
		}
		for _, mig := range test.available {
			var fn string
			var err error
			if fn, err = pathToMigrationFile(pathToTestDir, mig); err != nil {
				panic(err)
			}
			if _, err = os.Create(fn); err != nil {
				panic(err)
			}
		}
		stubbedAppliedVersions := StubAppliedVersions{
			versions: make([]string, len(test.applied)),
		}
		for i, mig := range test.applied {
			stubbedAppliedVersions.versions[i] = mig.Timestamp().Format(TimeFormat)
		}
		// teardown
		defer os.RemoveAll(pathToTestDir + "/" + string(i))

		// finally run the test case
		driver := StubDB{
			appliedVersions: &stubbedAppliedVersions,
			dsn:             StubDSN{},
		}
		output, err := listMigrationsToApply(
			&driver,
			pathToTestDir,
			test.direction,
			test.finishAtVersion,
			false,
		)
		gotError := err != nil
		if gotError && !test.expectError {
			t.Errorf("test %d; got error %v but did not expect one", i, err)
			return err
		} else if !gotError && test.expectError {
			t.Errorf("test %d; did not get error but did expect one", i)
			return err
		}
		if len(output) != len(test.expectedOut) {
			t.Errorf(
				"test %d; got wrong output length %d, expected length to be %d",
				i, len(output), len(test.expectedOut),
			)
			return err
		}
		for j, mig := range output {
			if mig.Timestamp() != test.expectedOut[j].Timestamp() {
				t.Errorf(
					"test [%d][%d]; wrong Timestamp; got %q, expected %q",
					i, j, mig.Timestamp(), test.expectedOut[j].Timestamp(),
				)
			}
			if mig.Direction() != test.expectedOut[j].Direction() {
				t.Errorf(
					"test [%d][%d]; wrong Direction; got %q, expected %q",
					i, j, mig.Direction(), test.expectedOut[j].Direction(),
				)
			}
			if mig.Name() != test.expectedOut[j].Name() {
				t.Errorf(
					"test [%d][%d]; wrong Name; got %q, expected %q",
					i, j, mig.Name(), test.expectedOut[j].Name(),
				)
			}
		}
		return nil
	}

	for i, test := range tests {
		if err := runTest(i, test); err != nil {
			t.Errorf("test [%d] failed; %v", i, err)
		}
	}
}
