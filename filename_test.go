package godfish

import (
	"testing"
)

func TestFilename(t *testing.T) {
	tests := []struct {
		version   string
		direction Indirection
		label     string
		expOut    filename
		expErr    bool
	}{
		{
			version:   "20191118121314",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "test",
			expOut:    filename("forward-20191118121314-test.sql"),
		},
		{
			version:   "20191118121314",
			direction: Indirection{Value: DirReverse, Label: "reverse"},
			label:     "test",
			expOut:    filename("reverse-20191118121314-test.sql"),
		},
		// timestamp too long
		{
			version:   "201911181213141516",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "test",
			expOut:    filename("forward-20191118121314-test.sql"),
		},
		// timestamp too short
		{
			version:   "1234",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "test",
			expOut:    filename("forward-1234-test.sql"),
		},
		// label has dashes
		{
			version:   "20191118121314",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "foo-bar",
			expOut:    filename("forward-20191118121314-foo-bar.sql"),
		},
		// alternative names
		{
			direction: Indirection{Value: DirForward, Label: "migrate"},
			version:   "20191118121314",
			label:     "test",
			expOut:    filename("migrate-20191118121314-test.sql"),
		},
		{
			direction: Indirection{Value: DirForward, Label: "up"},
			version:   "20191118121314",
			label:     "test",
			expOut:    filename("up-20191118121314-test.sql"),
		},
		{
			direction: Indirection{Value: DirReverse, Label: "rollback"},
			version:   "20191118121314",
			label:     "test",
			expOut:    filename("rollback-20191118121314-test.sql"),
		},
		{
			direction: Indirection{Value: DirReverse, Label: "down"},
			version:   "20191118121314",
			label:     "test",
			expOut:    filename("down-20191118121314-test.sql"),
		},
	}
	for i, test := range tests {
		out, err := makeFilename(test.version, test.direction, test.label)
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

func mustMakeMigration(version string, indirection Indirection, label string) Migration {
	ver, err := parseVersion(version)
	if err != nil {
		panic(err)
	}
	return &mutation{
		indirection: indirection,
		label:       label,
		version:     ver,
	}
}

func TestParseMigration(t *testing.T) {
	tests := []struct {
		filename filename
		expOut   Migration
		expErr   bool
	}{
		{
			filename: filename("forward-20191118121314-test.sql"),
			expOut: mustMakeMigration(
				"20191118121314",
				Indirection{Value: DirForward, Label: "forward"},
				"test",
			),
		},
		{
			filename: filename("reverse-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirReverse}, "test"),
		},
		// no extension
		{
			filename: filename("forward-20191118121314-test"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward}, "test"),
		},
		// timestamp too long
		{
			filename: filename("forward-201911181213141516-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward}, "516-test"),
		},
		// timestamp short
		{
			filename: filename("forward-1234-test.sql"),
			expOut:   mustMakeMigration("1234", Indirection{Value: DirForward}, "test"),
		},
		// unknown direction
		{filename: filename("foo-20191118121314-bar.sql"), expErr: true},
		// just bad
		{filename: filename("foo-bar"), expErr: true},
		// label has a delimiter
		{
			filename: filename("forward-20191118121314-foo-bar.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward}, "foo-bar"),
		},
		// alternative names for directions
		{
			filename: filename("migrate-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward, Label: "migration"}, "test"),
		},
		{
			filename: filename("up-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward, Label: "up"}, "test"),
		},
		{
			filename: filename("rollback-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirReverse, Label: "rollback"}, "test"),
		},
		{
			filename: filename("down-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirReverse, Label: "down"}, "test"),
		},
		// unix timestamp (seconds) as version
		{
			filename: filename("forward-1574079194-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward, Label: "forward"}, "test"),
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
		if actual.Indirection().Value != test.expOut.Indirection().Value {
			t.Errorf(
				"test %d; wrong Direction; expected %s, got %s",
				i, test.expOut.Indirection().Value, actual.Indirection().Value,
			)
		}
		if actual.Label() != test.expOut.Label() {
			t.Errorf(
				"test %d; wrong Name; expected %s, got %s",
				i, test.expOut.Label(), actual.Label(),
			)
		}
		act := actual.Version()
		exp := test.expOut.Version()
		if act.Before(exp) || exp.Before(act) {
			t.Errorf(
				"test %d; wrong Timestamp; expected %s, got %s",
				i, test.expOut.Version(), actual.Version(),
			)
		}
	}
}
