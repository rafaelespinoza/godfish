package internal_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
)

const baseTestOutputDir = "/tmp/godfish_test"

func TestParseMigration(t *testing.T) {
	type testCase struct {
		filename       internal.Filename
		expErr         bool
		expIndirection internal.Indirection
		expLabel       string
		// expVersionInput should be an input to ParseVersion, the
		// implementation used in the test assertions is not exported, so you
		// cannot directly construct it.
		expVersionInput string
	}

	runTest := func(t *testing.T, test testCase) {
		actual, err := internal.ParseMigration(test.filename)
		if !test.expErr && err != nil {
			t.Fatal(err)
		} else if test.expErr && err == nil {
			t.Fatal("expected error but did not get one")
		} else if test.expErr && err != nil {
			// some unexported funcs in the godfish package have behavior that
			// depend on this wrapped error.
			if !errors.Is(err, internal.ErrDataInvalid) {
				t.Fatalf("expected error %v to wrap %v", err, internal.ErrDataInvalid)
			}
			return // ok, nothing more to test.
		}

		if actual.Indirection().Value != test.expIndirection.Value {
			t.Errorf(
				"wrong Direction; got %s, expected %s",
				actual.Indirection().Value, test.expIndirection.Value,
			)
		}

		if actual.Label() != test.expLabel {
			t.Errorf(
				"wrong Name; got %s, expected %s",
				actual.Label(), test.expLabel,
			)
		}

		expVersion, err := internal.ParseVersion(test.expVersionInput)
		if err != nil {
			t.Fatal(err)
		}
		gotVersion := actual.Version()
		if gotVersion.Before(expVersion) || expVersion.Before(gotVersion) {
			t.Errorf(
				"wrong Version timestamp; got %s, expected %s",
				gotVersion.String(), expVersion.String(),
			)
		}
	}

	t.Run("ok", func(t *testing.T) {
		runTest(t, testCase{
			filename:        internal.Filename("forward-20191118121314-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirForward, Label: "forward"},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})

		runTest(t, testCase{
			filename:        internal.Filename("reverse-20191118121314-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirReverse},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})
	})

	t.Run("no extension", func(t *testing.T) {
		runTest(t, testCase{
			filename:        internal.Filename("forward-20191118121314-test"),
			expIndirection:  internal.Indirection{Value: internal.DirForward},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})
	})

	t.Run("timestamp is too long", func(t *testing.T) {
		runTest(t, testCase{
			filename:        internal.Filename("forward-201911181213141516-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirForward},
			expLabel:        "516-test",
			expVersionInput: "20191118121314",
		})
	})

	t.Run("timestamp is short", func(t *testing.T) {
		// short, but ok
		runTest(t, testCase{
			filename:        internal.Filename("forward-1234-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirForward},
			expLabel:        "test",
			expVersionInput: "1234",
		})

		// too short
		runTest(t, testCase{filename: internal.Filename("forward-123-test.sql"), expErr: true})
	})

	t.Run("unknown direction", func(t *testing.T) {
		runTest(t, testCase{filename: internal.Filename("foo-20191118121314-bar.sql"), expErr: true})
	})

	t.Run("just bad", func(t *testing.T) {
		runTest(t, testCase{filename: internal.Filename("foo-bar"), expErr: true})
	})

	t.Run("label", func(t *testing.T) {
		t.Run("has a delimiter", func(t *testing.T) {
			runTest(t, testCase{
				filename:        internal.Filename("forward-20191118121314-foo-bar.sql"),
				expIndirection:  internal.Indirection{Value: internal.DirForward},
				expLabel:        "foo-bar",
				expVersionInput: "20191118121314",
			})
		})

		t.Run("empty", func(t *testing.T) {
			runTest(t, testCase{
				filename:        internal.Filename("forward-20191118121314-.sql"),
				expIndirection:  internal.Indirection{Value: internal.DirForward},
				expLabel:        "",
				expVersionInput: "20191118121314",
			})
		})

		// These cases describe outcomes where the user omits the label portion
		// from the filename. The feature here is that the code doesn't blow up.
		t.Run("omitted entirely", func(t *testing.T) {
			runTest(t, testCase{
				filename:        internal.Filename("forward-20191118121314.sql"),
				expIndirection:  internal.Indirection{Value: internal.DirForward},
				expLabel:        "sql",
				expVersionInput: "20191118121314",
			})

			// No filename extension
			runTest(t, testCase{
				filename:        internal.Filename("forward-20191118121314"),
				expIndirection:  internal.Indirection{Value: internal.DirForward},
				expLabel:        "",
				expVersionInput: "20191118121314",
			})
		})
	})

	t.Run("alternative names for directions", func(t *testing.T) {
		runTest(t, testCase{
			filename:        internal.Filename("migrate-20191118121314-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirForward, Label: "migration"},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})

		runTest(t, testCase{
			filename:        internal.Filename("up-20191118121314-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirForward, Label: "up"},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})

		runTest(t, testCase{
			filename:        internal.Filename("rollback-20191118121314-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirReverse, Label: "rollback"},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})

		runTest(t, testCase{
			filename:        internal.Filename("down-20191118121314-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirReverse, Label: "down"},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})
	})

	t.Run("unix timestamps as version", func(t *testing.T) {
		runTest(t, testCase{
			filename:        internal.Filename("forward-1574079194-test.sql"),
			expIndirection:  internal.Indirection{Value: internal.DirForward, Label: "forward"},
			expLabel:        "test",
			expVersionInput: "20191118121314",
		})
	})
}

func TestMigrationParams(t *testing.T) {
	type testCase struct {
		name               string
		reversible         bool
		dirpath            string
		fwdLabel, revLabel string
		expectedDirections []string
		expectError        bool
	}

	runTest := func(t *testing.T, test testCase) {
		var (
			migParams  *internal.MigrationParams
			err        error
			filesAfter []os.DirEntry
		)

		// construct params and test the fields.
		migParams, err = internal.NewMigrationParams(test.name, test.reversible, test.dirpath, test.fwdLabel, test.revLabel)
		if err != nil {
			t.Fatal(err)
		}
		if migParams.Forward.Indirection().Value != internal.DirForward {
			t.Errorf(
				"wrong Direction; expected %s, got %s",
				internal.DirForward, migParams.Forward.Indirection().Value,
			)
		}
		if migParams.Reverse.Indirection().Value != internal.DirReverse {
			t.Errorf(
				"wrong Direction; expected %s, got %s",
				internal.DirReverse, migParams.Reverse.Indirection().Value,
			)
		}
		for i, mig := range []internal.Migration{migParams.Forward, migParams.Reverse} {
			if i > 0 && !test.reversible {
				continue
			}
			if mig.Label() != test.name {
				t.Errorf("test [%d]; Name should be unchanged", i)
			}
			if mig.Version().String() == "" {
				t.Errorf("test [%d]; got empty Timestamp", i)
			}
		}

		// generate files and test effects.
		err = migParams.GenerateFiles()
		if err != nil && !test.expectError {
			t.Fatal(err)
		} else if err == nil && test.expectError {
			t.Fatalf("expected an error but got %v", err)
		} else if err != nil && test.expectError {
			return // test passes, no more things to check.
		}

		if filesAfter, err = os.ReadDir(test.dirpath); err != nil {
			t.Fatal(err)
		}

		if len(filesAfter) != len(test.expectedDirections) {
			t.Fatalf(
				"expected to generate %d files, got %d",
				len(test.expectedDirections), len(filesAfter),
			)
		}

		// Some golang platforms seem to have differing output orders for
		// reading filenames in a directory. So, just sort them first.
		sort.Slice(filesAfter, func(i, j int) bool { return filesAfter[i].Name() < filesAfter[j].Name() })

		// Sorting this makes sense here as long as the generated filenames
		// begin with the direction label.
		sort.Strings(test.expectedDirections)

		for i, dirEntry := range filesAfter {
			patt := fmt.Sprintf("%s-[0-9]*-%s.sql", test.expectedDirections[i], test.name)
			name := dirEntry.Name()
			if match, err := filepath.Match(patt, name); err != nil {
				t.Fatalf("test [%d]; %v", i, err)
			} else if !match {
				t.Errorf(
					"test [%d]; expected filename %q to match pattern %q",
					i, name, patt,
				)
			}
		}
	}

	t.Run("reversible", func(t *testing.T) {
		runTest(t, testCase{
			name:               "foo",
			reversible:         true,
			dirpath:            t.TempDir(),
			fwdLabel:           "forward",
			revLabel:           "reverse",
			expectedDirections: []string{"forward", "reverse"},
		})
	})

	t.Run("forward only", func(t *testing.T) {
		runTest(t, testCase{
			name:               "bar",
			reversible:         false,
			dirpath:            t.TempDir(),
			fwdLabel:           "forward",
			expectedDirections: []string{"forward"},
		})
	})

	t.Run("delimiter in the name", func(t *testing.T) {
		runTest(t, testCase{
			name:               "delimiter-in-the-name",
			reversible:         false,
			dirpath:            t.TempDir(),
			fwdLabel:           "forward",
			revLabel:           "reverse",
			expectedDirections: []string{"forward"},
		})
	})

	t.Run("alternative direction names", func(t *testing.T) {
		runTest(t, testCase{
			name:               "alternatives",
			reversible:         true,
			dirpath:            t.TempDir(),
			fwdLabel:           "up",
			revLabel:           "down",
			expectedDirections: []string{"up", "down"},
		})
	})

	t.Run("err", func(t *testing.T) {
		runTest(t, testCase{
			name:        "bad",
			reversible:  false,
			dirpath:     filepath.Join(t.TempDir(), "this_should_not_exist"),
			fwdLabel:    "forward",
			revLabel:    "reverse",
			expectError: true,
		})
	})
}
