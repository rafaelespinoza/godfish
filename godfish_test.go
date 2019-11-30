package godfish_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bitbucket.org/rafaelespinoza/godfish"
)

const (
	baseTestOutputDir = "/tmp/godfish"
	testDBName        = "godfish_test"
)

func TestMain(m *testing.M) {
	os.MkdirAll(baseTestOutputDir, 0755)
	m.Run()
	os.RemoveAll(baseTestOutputDir)
}

func TestMigrationParams(t *testing.T) {
	var testDir *os.File
	var mig *godfish.MigrationParams
	var driver godfish.Driver
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

	driver, err = godfish.NewDriver("postgres", godfish.PGParams{
		Encoding: "UTF8",
		Host:     "localhost",
		Name:     testDBName,
		Pass:     os.Getenv("DB_PASSWORD"),
		Port:     "5432",
	})
	if err != nil {
		t.Error(err)
		return
	}
	for i, mig := range migrations {
		if err != nil {
			t.Errorf("test %d %v\n", i, err)
			return
		}
		timestamp := mig.Timestamp()
		err = godfish.ApplyMigration(
			driver,
			mig.Direction(),
			baseTestOutputDir,
			timestamp.Format(godfish.TimeFormat),
		)
		if err != nil {
			t.Errorf("test %d %v\n", i, err)
			return
		}
	}
}

func mustMakeFilename(ver string, dir godfish.Direction, name string) godfish.Filename {
	out, err := godfish.MakeFilename(ver, dir, name)
	if err != nil {
		panic(err)
	}
	return out
}

func TestFilename(t *testing.T) {
	const (
		version = "20191118121314"
		name    = "test"
	)
	actual := []godfish.Filename{
		mustMakeFilename(version, godfish.DirForward, name),
		mustMakeFilename(version, godfish.DirReverse, name),
	}
	expected := []godfish.Filename{
		godfish.Filename("20191118121314.forward.test.sql"),
		godfish.Filename("20191118121314.reverse.test.sql"),
	}
	for i, name := range actual {
		if name != expected[i] {
			t.Errorf(
				"wrong filename; got %q, expected %q",
				name, expected[i],
			)
		}
	}
}

func mustMakeMigration(direction godfish.Direction) godfish.Migration {
	ts, err := time.Parse("20060102150405", "20191118121314")
	if err != nil {
		panic(err)
	}
	mut, err := godfish.NewMutation(ts, direction, "test")
	if err != nil {
		panic(err)
	}
	return mut
}

func TestMigration(t *testing.T) {
	tests := []struct {
		filename godfish.Filename
		expected godfish.Migration
	}{
		{
			filename: godfish.Filename("20191118121314.forward.test.sql"),
			expected: mustMakeMigration(godfish.DirForward),
		},
		{
			filename: godfish.Filename("20191118121314.reverse.test.sql"),
			expected: mustMakeMigration(godfish.DirReverse),
		},
	}

	for i, test := range tests {
		actual, err := godfish.ParseMigration(test.filename)
		if err != nil {
			t.Error(err)
			return
		}
		if actual.Direction() != test.expected.Direction() {
			t.Errorf(
				"test %d; wrong Direction; expected %s, got %s",
				i, test.expected.Direction(), actual.Direction(),
			)
		}
		if actual.Name() != test.expected.Name() {
			t.Errorf(
				"test %d; wrong Name; expected %s, got %s",
				i, test.expected.Name(), actual.Name(),
			)
		}
		if !actual.Timestamp().Equal(test.expected.Timestamp()) {
			t.Errorf(
				"test %d; wrong Timestamp; expected %s, got %s",
				i, test.expected.Timestamp(), actual.Timestamp(),
			)
		}
	}
}

func TestListVersionsToApply(t *testing.T) {
	tests := []struct {
		direction   godfish.Direction
		applied     []string
		available   []string
		expectedOut []string
		expectError bool
	}{
		{
			direction:   godfish.DirForward,
			applied:     []string{"1234", "5678"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{},
		},
		{
			direction:   godfish.DirForward,
			applied:     []string{"1234"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"5678"},
		},
		{
			direction:   godfish.DirForward,
			applied:     []string{},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"1234", "5678"},
		},
		{
			direction:   godfish.DirReverse,
			applied:     []string{"1234", "5678"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"1234", "5678"},
		},
		{
			direction:   godfish.DirReverse,
			applied:     []string{"1234"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"1234"},
		},
		{
			direction:   godfish.DirReverse,
			applied:     []string{},
			available:   []string{"1234", "5678"},
			expectedOut: []string{},
		},
		{
			direction:   godfish.DirForward,
			applied:     []string{},
			available:   []string{},
			expectedOut: []string{},
		},
		{
			direction:   godfish.DirReverse,
			applied:     []string{},
			available:   []string{},
			expectedOut: []string{},
		},
		{
			applied:     []string{},
			available:   []string{},
			expectError: true,
		},
	}

	for i, test := range tests {
		actual, err := godfish.ListVersionsToApply(
			test.direction,
			test.applied,
			test.available,
		)
		gotError := err != nil
		if gotError && !test.expectError {
			t.Errorf("test %d; got error %v but did not expect one", i, err)
			continue
		} else if !gotError && test.expectError {
			t.Errorf("test %d; did not get error but did expect one", i)
			continue
		}
		if len(actual) != len(test.expectedOut) {
			t.Errorf(
				"test %d; got wrong output length %d, expected length to be %d",
				i, len(actual), len(test.expectedOut),
			)
			continue
		}
		for j, version := range actual {
			if version != test.expectedOut[j] {
				t.Errorf(
					"test [%d][%d]; got version %q but expected %q",
					i, j, version, test.expectedOut[j],
				)
			}
		}
	}
}
