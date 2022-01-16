package internal

import (
	"testing"
)

func TestFilename(t *testing.T) {
	tests := []struct {
		version   string
		direction Indirection
		label     string
		expOut    Filename
	}{
		{
			version:   "20191118121314",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "test",
			expOut:    Filename("forward-20191118121314-test.sql"),
		},
		{
			version:   "20191118121314",
			direction: Indirection{Value: DirReverse, Label: "reverse"},
			label:     "test",
			expOut:    Filename("reverse-20191118121314-test.sql"),
		},
		// timestamp too long
		{
			version:   "201911181213141516",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "test",
			expOut:    Filename("forward-20191118121314-test.sql"),
		},
		// timestamp too short
		{
			version:   "1234",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "test",
			expOut:    Filename("forward-1234-test.sql"),
		},
		// label has dashes
		{
			version:   "20191118121314",
			direction: Indirection{Value: DirForward, Label: "forward"},
			label:     "foo-bar",
			expOut:    Filename("forward-20191118121314-foo-bar.sql"),
		},
		// alternative names
		{
			direction: Indirection{Value: DirForward, Label: "migrate"},
			version:   "20191118121314",
			label:     "test",
			expOut:    Filename("migrate-20191118121314-test.sql"),
		},
		{
			direction: Indirection{Value: DirForward, Label: "up"},
			version:   "20191118121314",
			label:     "test",
			expOut:    Filename("up-20191118121314-test.sql"),
		},
		{
			direction: Indirection{Value: DirReverse, Label: "rollback"},
			version:   "20191118121314",
			label:     "test",
			expOut:    Filename("rollback-20191118121314-test.sql"),
		},
		{
			direction: Indirection{Value: DirReverse, Label: "down"},
			version:   "20191118121314",
			label:     "test",
			expOut:    Filename("down-20191118121314-test.sql"),
		},
	}
	for i, test := range tests {
		out := MakeFilename(test.version, test.direction, test.label)
		if out != string(test.expOut) {
			t.Errorf(
				"test %d; wrong filename; got %q, expected %q",
				i, out, test.expOut,
			)
		}
	}
}

func mustMakeMigration(version string, indirection Indirection, label string) Migration {
	ver, err := ParseVersion(version)
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
		filename Filename
		expOut   Migration
		expErr   bool
	}{
		{
			filename: Filename("forward-20191118121314-test.sql"),
			expOut: mustMakeMigration(
				"20191118121314",
				Indirection{Value: DirForward, Label: "forward"},
				"test",
			),
		},
		{
			filename: Filename("reverse-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirReverse}, "test"),
		},
		// no extension
		{
			filename: Filename("forward-20191118121314-test"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward}, "test"),
		},
		// timestamp too long
		{
			filename: Filename("forward-201911181213141516-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward}, "516-test"),
		},
		// timestamp short
		{
			filename: Filename("forward-1234-test.sql"),
			expOut:   mustMakeMigration("1234", Indirection{Value: DirForward}, "test"),
		},
		// unknown direction
		{filename: Filename("foo-20191118121314-bar.sql"), expErr: true},
		// just bad
		{filename: Filename("foo-bar"), expErr: true},
		// label has a delimiter
		{
			filename: Filename("forward-20191118121314-foo-bar.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward}, "foo-bar"),
		},
		// alternative names for directions
		{
			filename: Filename("migrate-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward, Label: "migration"}, "test"),
		},
		{
			filename: Filename("up-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward, Label: "up"}, "test"),
		},
		{
			filename: Filename("rollback-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirReverse, Label: "rollback"}, "test"),
		},
		{
			filename: Filename("down-20191118121314-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirReverse, Label: "down"}, "test"),
		},
		// unix timestamp (seconds) as version
		{
			filename: Filename("forward-1574079194-test.sql"),
			expOut:   mustMakeMigration("20191118121314", Indirection{Value: DirForward, Label: "forward"}, "test"),
		},
	}

	for i, test := range tests {
		actual, err := ParseMigration(test.filename)
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
