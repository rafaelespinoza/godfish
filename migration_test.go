package godfish_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/rafaelespinoza/godfish"
)

func TestNewMigrationParams(t *testing.T) {
	testOutputDir := baseTestOutputDir + "/" + t.Name()
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		name        string
		reversible  bool
		dirpath     string
		expectError bool
	}

	runTest := func(t *testing.T, test testCase) {
		t.Helper()

		migParams, err := godfish.NewMigrationParams(test.name, test.reversible, test.dirpath, "", "")
		if err != nil && !test.expectError {
			t.Fatal(err)
		} else if err == nil && test.expectError {
			t.Fatalf("expected an error but got %v", err)
		} else if err != nil && test.expectError {
			return // test passes, no more things to check.
		}

		if migParams.Forward.Indirection().Value != godfish.DirForward {
			t.Errorf(
				"wrong Direction; expected %s, got %s",
				godfish.DirForward, migParams.Forward.Indirection().Value,
			)
		}
		if migParams.Reverse.Indirection().Value != godfish.DirReverse {
			t.Errorf(
				"wrong Direction; expected %s, got %s",
				godfish.DirReverse, migParams.Reverse.Indirection().Value,
			)
		}
		for i, mig := range []godfish.Migration{migParams.Forward, migParams.Reverse} {
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
	}

	runTest(t, testCase{name: "foo", reversible: true, dirpath: testOutputDir})
	runTest(t, testCase{name: "bar", reversible: false, dirpath: testOutputDir})
	runTest(t, testCase{
		name:        "bad",
		reversible:  false,
		dirpath:     testOutputDir + "/this_should_not_exist",
		expectError: true,
	})
}

func TestMigrationParamsGenerateFiles(t *testing.T) {
	testOutputDir := baseTestOutputDir + "/" + t.Name()
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		name               string
		reversible         bool
		fwdLabel, revLabel string
		expectedDirections []string
	}

	runTest := func(t *testing.T, test testCase) {
		t.Helper()

		var err error
		var outdirPath string
		var migParams *godfish.MigrationParams

		if outdirPath, err = os.MkdirTemp(testOutputDir, ""); err != nil {
			t.Fatal(err)
		}

		migParams, err = godfish.NewMigrationParams(test.name, test.reversible, outdirPath, test.fwdLabel, test.revLabel)
		if err != nil {
			t.Fatal(err)
		}

		var filesBefore, filesAfter []os.DirEntry
		if filesBefore, err = os.ReadDir(outdirPath); err != nil {
			t.Fatal(err)
		}
		if err = migParams.GenerateFiles(); err != nil {
			t.Error(err)
		}
		if filesAfter, err = os.ReadDir(outdirPath); err != nil {
			t.Fatal(err)
		}

		// Some golang platforms seem to have differing output orders for
		// reading filenames in a directory. So, just sort them first.
		sort.Slice(filesBefore, func(i, j int) bool { return filesBefore[i].Name() < filesBefore[j].Name() })
		sort.Slice(filesAfter, func(i, j int) bool { return filesAfter[i].Name() < filesAfter[j].Name() })

		// Sorting this makes sense here as long as the generated filenames
		// begin with the direction label.
		sort.Strings(test.expectedDirections)

		actualNumFiles := len(filesAfter) - len(filesBefore)
		if actualNumFiles != len(test.expectedDirections) {
			t.Fatalf(
				"expected to generate %d files, got %d",
				len(test.expectedDirections), actualNumFiles,
			)
		}

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

	runTest(t, testCase{
		name:               "foo",
		reversible:         true,
		expectedDirections: []string{"forward", "reverse"},
	})
	runTest(t, testCase{
		name:               "bar",
		reversible:         false,
		expectedDirections: []string{"forward"},
	})

	runTest(t, testCase{
		name:               "delimiter-in-the-name",
		reversible:         false,
		expectedDirections: []string{"forward"},
	})

	runTest(t, testCase{
		name:               "alternatives",
		reversible:         true,
		fwdLabel:           "up",
		revLabel:           "down",
		expectedDirections: []string{"up", "down"},
	})
}
