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
		filename("20191118121314.forward.test.sql"),
		filename("20191118121314.reverse.test.sql"),
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

func mustMakeMigration(direction Direction) Migration {
	ts, err := time.Parse("20060102150405", "20191118121314")
	if err != nil {
		panic(err)
	}
	mut, err := newMutation(ts, direction, "test")
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
			filename: filename("20191118121314.forward.test.sql"),
			expected: mustMakeMigration(DirForward),
		},
		{
			filename: filename("20191118121314.reverse.test.sql"),
			expected: mustMakeMigration(DirReverse),
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
