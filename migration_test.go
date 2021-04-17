package godfish_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/rafaelespinoza/godfish"
)

func TestMigrationParams(t *testing.T) {
	testOutputDir := baseTestOutputDir + "/" + t.Name()
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("New", func(t *testing.T) {
		tests := []struct {
			label       string
			reversible  bool
			dirpath     string
			expectError bool
		}{
			{label: "foo", reversible: true, dirpath: testOutputDir},
			{label: "bar", reversible: false, dirpath: testOutputDir},
			{
				label:       "bad",
				reversible:  false,
				dirpath:     testOutputDir + "/this_should_not_exist",
				expectError: true,
			},
		}

		for i, test := range tests {
			var err error
			var migParams *godfish.MigrationParams

			migParams, err = godfish.NewMigrationParams(test.label, test.reversible, test.dirpath)
			if err != nil && !test.expectError {
				t.Fatalf("test %d; %v", i, err)
			} else if err == nil && test.expectError {
				t.Fatalf("test %d; expected an error but got %v", i, err)
			} else if err != nil && test.expectError {
				continue // test passes, no more things to check.
			}

			if migParams.Forward.Indirection().Value != godfish.DirForward {
				t.Errorf(
					"test %d; wrong Direction; expected %s, got %s",
					i, godfish.DirForward, migParams.Forward.Indirection().Value,
				)
			}
			if migParams.Reverse.Indirection().Value != godfish.DirReverse {
				t.Errorf(
					"test %d; wrong Direction; expected %s, got %s",
					i, godfish.DirReverse, migParams.Reverse.Indirection().Value,
				)
			}
			for j, mig := range []godfish.Migration{migParams.Forward, migParams.Reverse} {
				if j > 0 && !test.reversible {
					continue
				}
				if mig.Label() != test.label {
					t.Errorf("test [%d][%d]; Name should be unchanged", i, j)
				}
				if mig.Version().String() == "" {
					t.Errorf("test [%d][%d]; got empty Timestamp", i, j)
				}
			}
		}
	})

	t.Run("GenerateFiles", func(t *testing.T) {
		tests := []struct {
			label      string
			reversible bool
		}{
			{label: "foo", reversible: true},
			{label: "bar", reversible: false},
			{label: "foo-bar", reversible: false}, // delimiter is part of the label
		}

		for i, test := range tests {
			var err error
			var outdirPath string
			var migParams *godfish.MigrationParams

			if outdirPath, err = os.MkdirTemp(testOutputDir, ""); err != nil {
				t.Fatal(err)
			}

			migParams, err = godfish.NewMigrationParams(test.label, test.reversible, outdirPath)
			if err != nil {
				t.Fatalf("test %d; %v", i, err)
			}

			var filesBefore, filesAfter []os.DirEntry
			if filesBefore, err = os.ReadDir(outdirPath); err != nil {
				t.Fatalf("test %d; %v", err, i)
			}
			if err = migParams.GenerateFiles(); err != nil {
				t.Errorf("test %d; %v", i, err)
			}
			if filesAfter, err = os.ReadDir(outdirPath); err != nil {
				t.Fatalf("test %d; %v", i, err)
			}
			// Tests much more predictable if sorting after reading entries. Some OS
			// implementations seem to order the output differently.
			sort.Slice(filesBefore, func(i, j int) bool { return filesBefore[i].Name() < filesBefore[j].Name() })
			sort.Slice(filesAfter, func(i, j int) bool { return filesAfter[i].Name() < filesAfter[j].Name() })

			actualNumFiles := len(filesAfter) - len(filesBefore)
			expectedDirections := []string{"forward"}
			if test.reversible {
				expectedDirections = append(expectedDirections, "reverse")
			}
			if actualNumFiles != len(expectedDirections) {
				t.Errorf(
					"test %d; expected to generate %d files, got %d",
					i, len(expectedDirections), actualNumFiles,
				)
				continue
			}

			for j, dirEntry := range filesAfter {
				patt := fmt.Sprintf("%s-[0-9]*-%s.sql", expectedDirections[j], test.label)
				name := dirEntry.Name()
				if match, err := filepath.Match(patt, name); err != nil {
					t.Fatalf("test [%d][%d]; %v", i, j, err)
				} else if !match {
					t.Errorf(
						"test [%d][%d]; expected filename %q to match pattern %q",
						i, j, name, patt,
					)
				}
			}
		}
	})
}
