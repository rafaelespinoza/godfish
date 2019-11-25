package godfish_test

import (
	"database/sql"
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

func TestMigration(t *testing.T) {
	var testDir *os.File
	var mig *godfish.Migration
	var dbHandler *sql.DB
	var err error
	if testDir, err = os.Open(baseTestOutputDir); err != nil {
		t.Error(err)
		return
	}
	if mig, err = godfish.NewMigration("foo", true, testDir); err != nil {
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
	mutations := []godfish.Mutation{mig.Forward, mig.Reverse}
	for _, mut := range mutations {
		if mut.Name() != "foo" {
			t.Errorf(
				"wrong Name; expected %s, got %s",
				"foo", mut.Name(),
			)
		}
		if mut.Timestamp().IsZero() {
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
	params := godfish.PGParams{
		Encoding: "UTF8",
		Host:     "localhost",
		Name:     testDBName,
		Pass:     os.Getenv("DB_PASSWORD"),
		Port:     "5432",
	}
	if dbHandler, err = godfish.Connect("postgres", params); err != nil {
		t.Error(err)
		return
	}
	for i, mut := range mutations {
		pathToFile, err := godfish.PathToMutationFile(baseTestOutputDir, mut)
		if err != nil {
			t.Errorf("test %d %v\n", i, err)
			return
		}
		if err = godfish.RunMutation(dbHandler, pathToFile); err != nil {
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

func mustMakeMutation(direction godfish.Direction) godfish.Mutation {
	ts, err := time.Parse("20060102150405", "20191118121314")
	if err != nil {
		panic(err)
	}
	mod, err := godfish.NewModification(ts, direction, "test")
	if err != nil {
		panic(err)
	}
	return mod
}

func TestMutation(t *testing.T) {
	tests := []struct {
		filename godfish.Filename
		expected godfish.Mutation
	}{
		{
			filename: godfish.Filename("20191118121314.forward.test.sql"),
			expected: mustMakeMutation(godfish.DirForward),
		},
		{
			filename: godfish.Filename("20191118121314.reverse.test.sql"),
			expected: mustMakeMutation(godfish.DirReverse),
		},
	}

	for i, test := range tests {
		actual, err := godfish.ParseMutation(test.filename)
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
