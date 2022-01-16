package internal_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
)

const baseTestOutputDir = "/tmp/godfish_test"

func TestMigrationParams(t *testing.T) {
	testOutputDir := baseTestOutputDir + "/" + t.Name()
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatal(err)
	}

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
		dirpath, err := os.MkdirTemp(testOutputDir, "")
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, testCase{
			name:               "foo",
			reversible:         true,
			dirpath:            dirpath,
			fwdLabel:           "forward",
			revLabel:           "reverse",
			expectedDirections: []string{"forward", "reverse"},
		})
	})

	t.Run("forward only", func(t *testing.T) {
		dirpath, err := os.MkdirTemp(testOutputDir, "")
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, testCase{
			name:               "bar",
			reversible:         false,
			dirpath:            dirpath,
			fwdLabel:           "forward",
			expectedDirections: []string{"forward"},
		})
	})

	t.Run("delimiter in the name", func(t *testing.T) {
		dirpath, err := os.MkdirTemp(testOutputDir, "")
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, testCase{
			name:               "delimiter-in-the-name",
			reversible:         false,
			dirpath:            dirpath,
			fwdLabel:           "forward",
			revLabel:           "reverse",
			expectedDirections: []string{"forward"},
		})
	})

	t.Run("alternative direction names", func(t *testing.T) {
		dirpath, err := os.MkdirTemp(testOutputDir, "")
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, testCase{
			name:               "alternatives",
			reversible:         true,
			dirpath:            dirpath,
			fwdLabel:           "up",
			revLabel:           "down",
			expectedDirections: []string{"up", "down"},
		})
	})

	t.Run("err", func(t *testing.T) {
		dirpath, err := os.MkdirTemp(testOutputDir, "")
		if err != nil {
			t.Fatal(err)
		}

		runTest(t, testCase{
			name:        "bad",
			reversible:  false,
			dirpath:     dirpath + "/this_should_not_exist",
			fwdLabel:    "forward",
			revLabel:    "reverse",
			expectError: true,
		})
	})
}
