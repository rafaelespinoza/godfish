package internal_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
)

func TestFilename(t *testing.T) {
	tests := []struct {
		version   string
		direction internal.Indirection
		label     string
		expOut    internal.Filename
	}{
		{
			version:   "20191118121314",
			direction: internal.Indirection{Value: internal.DirForward, Label: "forward"},
			label:     "test",
			expOut:    internal.Filename("forward-20191118121314-test.sql"),
		},
		{
			version:   "20191118121314",
			direction: internal.Indirection{Value: internal.DirReverse, Label: "reverse"},
			label:     "test",
			expOut:    internal.Filename("reverse-20191118121314-test.sql"),
		},
		// timestamp too long
		{
			version:   "201911181213141516",
			direction: internal.Indirection{Value: internal.DirForward, Label: "forward"},
			label:     "test",
			expOut:    internal.Filename("forward-20191118121314-test.sql"),
		},
		// timestamp too short
		{
			version:   "1234",
			direction: internal.Indirection{Value: internal.DirForward, Label: "forward"},
			label:     "test",
			expOut:    internal.Filename("forward-1234-test.sql"),
		},
		// label has dashes
		{
			version:   "20191118121314",
			direction: internal.Indirection{Value: internal.DirForward, Label: "forward"},
			label:     "foo-bar",
			expOut:    internal.Filename("forward-20191118121314-foo-bar.sql"),
		},
		// alternative names
		{
			direction: internal.Indirection{Value: internal.DirForward, Label: "migrate"},
			version:   "20191118121314",
			label:     "test",
			expOut:    internal.Filename("migrate-20191118121314-test.sql"),
		},
		{
			direction: internal.Indirection{Value: internal.DirForward, Label: "up"},
			version:   "20191118121314",
			label:     "test",
			expOut:    internal.Filename("up-20191118121314-test.sql"),
		},
		{
			direction: internal.Indirection{Value: internal.DirReverse, Label: "rollback"},
			version:   "20191118121314",
			label:     "test",
			expOut:    internal.Filename("rollback-20191118121314-test.sql"),
		},
		{
			direction: internal.Indirection{Value: internal.DirReverse, Label: "down"},
			version:   "20191118121314",
			label:     "test",
			expOut:    internal.Filename("down-20191118121314-test.sql"),
		},
	}
	for i, test := range tests {
		out := internal.MakeFilename(test.version, test.direction, test.label)
		if out != test.expOut {
			t.Errorf(
				"test %d; wrong filename; got %q, expected %q",
				i, out, test.expOut,
			)
		}
	}
}
