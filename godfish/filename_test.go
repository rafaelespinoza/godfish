package godfish

import (
	"testing"
	"time"
)

func mustMakeFilename(ver string, dir Direction, name string) filename {
	out, err := makeFilename(ver, dir, name)
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
	actual := []filename{
		mustMakeFilename(version, DirForward, name),
		mustMakeFilename(version, DirReverse, name),
	}
	expected := []filename{
		filename("forward-20191118121314-test.sql"),
		filename("reverse-20191118121314-test.sql"),
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

func mustMakeMigration(version string, direction Direction, name string) Migration {
	ts, err := time.Parse(TimeFormat, version)
	if err != nil {
		panic(err)
	}
	mut, err := newMutation(ts, direction, name)
	if err != nil {
		panic(err)
	}
	return mut
}

func TestParseMigration(t *testing.T) {
	tests := []struct {
		filename filename
		expected Migration
	}{
		{
			filename: filename("forward-20191118121314-test.sql"),
			expected: mustMakeMigration("20191118121314", DirForward, "test"),
		},
		{
			filename: filename("reverse-20191118121314-test.sql"),
			expected: mustMakeMigration("20191118121314", DirReverse, "test"),
		},
	}

	for i, test := range tests {
		actual, err := parseMigration(test.filename)
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
