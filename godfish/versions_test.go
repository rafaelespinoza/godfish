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
	err             error
	errorOnExecute  error
}

var _ Driver = (*StubDB)(nil)

func (d *StubDB) Name() string              { return "stub" }
func (d *StubDB) Connect() (*sql.DB, error) { return d.connection, d.err }
func (d *StubDB) Close() error              { return d.err }
func (d *StubDB) DSNParams() DSNParams      { return d.dsn }
func (d *StubDB) CreateSchemaMigrationsTable() error {
	if d.appliedVersions == nil {
		d.appliedVersions = makeStubAppliedVersions()
	}
	return d.err
}
func (d *StubDB) DumpSchema() error                        { return d.err }
func (d *StubDB) Execute(q string, a ...interface{}) error { return d.errorOnExecute }
func (d *StubDB) UpdateSchemaMigrations(direction Direction, version string) error {
	var stubbedAV *StubAppliedVersions
	av, err := d.AppliedVersions()
	if err != nil {
		return err
	}
	switch val := av.(type) {
	case *StubAppliedVersions:
		stubbedAV = val
	case nil:
		return ErrSchemaMigrationsDoesNotExist
	default:
		return fmt.Errorf(
			"if you assign anything to this field, make it a %T", stubbedAV,
		)
	}
	if direction == DirForward {
		stubbedAV.versions = append(stubbedAV.versions, version)
	} else {
		stubbedAV.versions = stubbedAV.versions[:len(stubbedAV.versions)-1]
	}
	d.appliedVersions = stubbedAV
	return nil
}
func (d *StubDB) AppliedVersions() (AppliedVersions, error) {
	if d.appliedVersions == nil {
		return nil, ErrSchemaMigrationsDoesNotExist
	}
	return d.appliedVersions, d.err
}

type StubDSN struct{}

func (d StubDSN) NewDriver(migConf *MigrationsConf) (Driver, error) {
	return &StubDB{dsn: d}, nil
}
func (d StubDSN) String() string { return "this://is.a/test" }

var _ DSNParams = (*StubDSN)(nil)

type StubAppliedVersions struct {
	counter  int
	versions []string
}

var _ AppliedVersions = (*StubAppliedVersions)(nil)

func makeStubAppliedVersions(migrations ...Migration) AppliedVersions {
	out := StubAppliedVersions{
		versions: make([]string, len(migrations)),
	}
	for i, mig := range migrations {
		out.versions[i] = mig.Timestamp().Format(TimeFormat)
	}
	return &out
}

func (r *StubAppliedVersions) Close() error {
	r.counter = 0
	return nil
}
func (r *StubAppliedVersions) Next() bool { return r.counter < len(r.versions) }
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
		makeTestFilesOrDie(t, pathToTestDir, test.available...)
		stubbedAppliedVersions := makeStubAppliedVersions(test.applied...)
		// teardown
		defer os.RemoveAll(pathToTestDir + "/" + string(i))

		// finally run the test case
		driver := StubDB{
			appliedVersions: stubbedAppliedVersions,
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

func TestApplyMigration(t *testing.T) {
	availableMigrations := []Migration{
		mustMakeMigration("12340102030405", DirForward, "alpha"),
		mustMakeMigration("23450102030405", DirForward, "bravo"),
		mustMakeMigration("34560102030405", DirForward, "charlie"),
	}
	t.Run("ErrNoFilesFound", func(t *testing.T) {
		pathToTestDir := baseTestOutputDir + "/" + t.Name()
		if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
			t.Fatalf("error creating test directory %s", pathToTestDir)
			return
		}
		defer os.RemoveAll(pathToTestDir)
		driver := StubDB{
			dsn: StubDSN{},
		}
		if err := ApplyMigration(
			&driver,
			pathToTestDir,
			DirForward,
			availableMigrations[0].Timestamp().Format(TimeFormat),
		); err != ErrNoFilesFound {
			t.Errorf("got %v, but expected %v", err, ErrNoFilesFound)
			return
		}
	})
	t.Run("works", func(t *testing.T) {
		pathToTestDir := baseTestOutputDir + "/" + t.Name()
		if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
			t.Fatalf("error creating test directory %s", pathToTestDir)
			return
		}
		defer os.RemoveAll(pathToTestDir)
		driver := StubDB{
			dsn: StubDSN{},
		}
		makeTestFilesOrDie(t, pathToTestDir, availableMigrations...)
		if av, err := driver.AppliedVersions(); err != nil && err != ErrSchemaMigrationsDoesNotExist {
			t.Errorf(
				"got %v, but expected %v",
				err, ErrSchemaMigrationsDoesNotExist,
			)
			return
		} else if av != nil {
			t.Fatal("test setup messed up, driver applied versions should be nil at this point")
		}

		if err := ApplyMigration(
			&driver,
			pathToTestDir,
			DirForward,
			availableMigrations[0].Timestamp().Format(TimeFormat),
		); err != nil {
			t.Errorf("did not expect error, but got %v", err)
			return
		}
		if av, err := driver.AppliedVersions(); err != nil {
			t.Errorf("got unexpected error %v", err)
			return
		} else if av == nil {
			t.Error(
				"driver AppliedVersions is empty, but expected something",
			)
			return
		}
	})
	t.Run("does not update schema_migrations after failed execution", func(t *testing.T) {
		pathToTestDir := baseTestOutputDir + "/" + t.Name()
		if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
			t.Fatalf("error creating test directory %s", pathToTestDir)
			return
		}
		defer os.RemoveAll(pathToTestDir)
		driver := StubDB{
			dsn: StubDSN{},
		}
		makeTestFilesOrDie(t, pathToTestDir, availableMigrations...)
		const numOK = 2
		for i, mig := range availableMigrations[:numOK] {
			if err := ApplyMigration(
				&driver,
				pathToTestDir,
				DirForward,
				mig.Timestamp().Format(TimeFormat),
			); err != nil {
				t.Errorf("migration [%d] did not expect error, but got %v", i, err)
				return
			}
		}
		// we should have `numOK` AppliedVersions at this point
		versionsBefore := collectVersionsOrDie(t, &driver)
		if len(versionsBefore) != numOK {
			t.Errorf(
				"wrong number of applied versions; got %d, expected %d",
				len(versionsBefore), numOK,
			)
			return
		}
		for i, version := range versionsBefore {
			expected := availableMigrations[i].Timestamp().Format(TimeFormat)
			if version != expected {
				t.Errorf(
					"versions[%d]; wrong version; got %q, expected %q",
					i, version, expected,
				)
			}
		}
		// set up last migration to fail, this should have no effect on the
		// schema migrations table (represented as applied versions).
		driver.errorOnExecute = fmt.Errorf("OOF")
		// reset so we can read versions again and compare
		if err := driver.appliedVersions.Close(); err != nil {
			t.Fatal(err)
		}

		if err := ApplyMigration(
			&driver,
			pathToTestDir,
			DirForward,
			availableMigrations[numOK].Timestamp().Format(TimeFormat),
		); err == nil {
			t.Errorf("expected an error but got %v", err)
			return
		}
		versionsAfter := collectVersionsOrDie(t, &driver)
		if len(versionsAfter) != len(versionsBefore) {
			t.Errorf(
				"wrong number of applied versions; got %d, expected %d",
				len(versionsAfter), len(versionsBefore),
			)
			return
		}
		for i, version := range versionsAfter {
			if version != versionsBefore[i] {
				t.Errorf(
					"versions[%d]; wrong version; got %q, expected %q",
					i, version, versionsBefore[i],
				)
			}
		}
	})
}

func makeTestFilesOrDie(t *testing.T, dirPath string, migrations ...Migration) {
	t.Helper()
	for _, mig := range migrations {
		if fn, err := pathToMigrationFile(dirPath, mig); err != nil {
			t.Fatal(err)
		} else if _, err = os.Create(fn); err != nil {
			t.Fatal(err)
		}
	}
}

func collectVersionsOrDie(t *testing.T, driver Driver) (out []string) {
	t.Helper()
	err := scanAppliedVersions(driver, func(rows AppliedVersions) error {
		var version string
		if ierr := rows.Scan(&version); ierr != nil {
			return ierr
		}
		out = append(out, version)
		return nil
	})
	if err != nil {
		t.Fatalf(
			"driver %q; could not scan applied version; %v",
			driver.Name(), err,
		)
	}
	return
}
