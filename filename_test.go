package godfish

import (
	"testing"
	"time"
)

func TestFilename(t *testing.T) {
	tests := []struct {
		version   string
		direction Direction
		name      string
		expOut    filename
		expErr    bool
	}{
		{
			version:   "20191118121314",
			direction: DirForward,
			name:      "test",
			expOut:    filename("forward-20191118121314-test.sql"),
		},
		{
			version:   "20191118121314",
			direction: DirReverse,
			name:      "test",
			expOut:    filename("reverse-20191118121314-test.sql"),
		},
		// timestamp too long
		{
			version:   "201911181213141516",
			direction: DirForward,
			name:      "test",
			expOut:    filename("forward-20191118121314-test.sql"),
		},
		// timestamp too short
		{
			version:   "201911181213",
			direction: DirForward,
			name:      "test",
			expErr:    true,
		},
		// unknown direction
		{
			version: "20191118121314",
			name:    "test",
			expErr:  true,
		},
		// name has dashes
		{
			version:   "20191118121314",
			direction: DirForward,
			name:      "foo-bar",
			expOut:    filename("forward-20191118121314-foo-bar.sql"),
		},
		// just bad
		{
			name:   "test",
			expErr: true,
		},
	}
	for i, test := range tests {
		out, err := makeFilename(test.version, test.direction, test.name)
		if !test.expErr && err != nil {
			t.Errorf("test %d; unexpected error; %v", i, err)
		} else if test.expErr && err == nil {
			t.Errorf("test %d; expected error but did not get one", i)
		}
		if out != test.expOut {
			t.Errorf(
				"test %d; wrong filename; got %q, expected %q",
				i, out, test.expOut,
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
		expOut   Migration
		expErr   bool
	}{
		{
			filename: filename("forward-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", DirForward, "test"),
		},
		{
			filename: filename("reverse-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", DirReverse, "test"),
		},
		// no extension
		{
			filename: filename("forward-20191118121314-test"),
			expOut:   mustMakeMigration("20191118121314", DirForward, "test"),
		},
		// timestamp too long
		{
			filename: filename("forward-201911181213141516-test.sql"),
			expOut:   mustMakeMigration("20191118121314", DirForward, "516-test"),
		},
		// timestamp too short
		{filename: filename("forward-201911181213-bar.sql"), expErr: true},
		// unknown direction
		{filename: filename("foo-20191118121314-bar.sql"), expErr: true},
		// just bad
		{filename: filename("foo-bar"), expErr: true},
		// name has a delimiter
		{
			filename: filename("forward-20191118121314-foo-bar.sql"),
			expOut:   mustMakeMigration("20191118121314", DirForward, "foo-bar"),
		},
	}

	for i, test := range tests {
		actual, err := parseMigration(test.filename)
		if !test.expErr && err != nil {
			t.Errorf("test %d; %v", i, err)
			continue
		} else if test.expErr && err == nil {
			t.Errorf("test %d; expected error but did not get one", i)
			continue
		} else if test.expErr && err != nil {
			continue // ok
		}
		if actual.Direction() != test.expOut.Direction() {
			t.Errorf(
				"test %d; wrong Direction; expected %s, got %s",
				i, test.expOut.Direction(), actual.Direction(),
			)
		}
		if actual.Name() != test.expOut.Name() {
			t.Errorf(
				"test %d; wrong Name; expected %s, got %s",
				i, test.expOut.Name(), actual.Name(),
			)
		}
		if !actual.Timestamp().Equal(test.expOut.Timestamp()) {
			t.Errorf(
				"test %d; wrong Timestamp; expected %s, got %s",
				i, test.expOut.Timestamp(), actual.Timestamp(),
			)
		}
	}
}
