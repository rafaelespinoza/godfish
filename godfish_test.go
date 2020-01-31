package godfish_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rafaelespinoza/godfish"
)

const baseTestOutputDir = "/tmp/godfish_test"

func TestMain(m *testing.M) {
	os.MkdirAll(baseTestOutputDir, 0755)
	m.Run()
	os.RemoveAll(baseTestOutputDir)
}

func TestMigrationParams(t *testing.T) {
	type testCase struct {
		name       string
		reversible bool
	}

	tests := []testCase{
		{
			name:       "foo",
			reversible: true,
		},
		{
			name:       "bar",
			reversible: false,
		},
		{
			name:       "foo-bar",
			reversible: false,
		},
	}

	for i, test := range tests {
		var directory *os.File
		var err error
		pathToTestDir := baseTestOutputDir + "/" + t.Name() + "/" + strconv.Itoa(i)
		if err = os.MkdirAll(pathToTestDir, 0755); err != nil {
			t.Fatal(err)
		}
		if directory, err = os.Open(pathToTestDir); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(directory.Name())

		mig, err := godfish.NewMigrationParams(test.name, test.reversible, directory)
		if err != nil {
			t.Error(err)
		}
		if mig.Forward.Direction() != godfish.DirForward {
			t.Errorf(
				"test %d; wrong Direction; expected %s, got %s",
				i, godfish.DirForward, mig.Forward.Direction(),
			)
		}
		if test.reversible {
			if mig.Reverse.Direction() != godfish.DirReverse {
				t.Errorf(
					"test %d; wrong Direction; expected %s, got %s",
					i, godfish.DirReverse, mig.Reverse.Direction(),
				)
			}
		}
		migrations := []godfish.Migration{mig.Forward, mig.Reverse}
		for j, mig := range migrations {
			if j > 0 && !test.reversible {
				continue
			}
			if mig.Name() != test.name {
				t.Errorf("test [%d][%d]; Name should be unchanged", i, j)
			}
			if mig.Timestamp().IsZero() {
				t.Errorf("test [%d][%d]; got empty Timestamp", i, j)
			}
		}

		var filesBefore, filesAfter []string
		if filesBefore, err = directory.Readdirnames(0); err != nil {
			t.Fatalf("test %d; %v", err, i)
		}
		if err = mig.GenerateFiles(); err != nil {
			t.Errorf("test %d; %v", i, err)
		}
		if filesAfter, err = directory.Readdirnames(0); err != nil {
			t.Fatalf("test %d; %v", i, err)
		}

		actualNumFiles := len(filesAfter) - len(filesBefore)
		numExpectedFiles := 2
		expectedDirections := []string{"reverse", "forward"}
		if !test.reversible {
			numExpectedFiles--
			// the list of filenames seem to be returned in ctime desc order...
			expectedDirections = expectedDirections[1:]
		}
		if actualNumFiles != numExpectedFiles {
			t.Errorf(
				"test %d; expected to generate %d files, got %d",
				i, numExpectedFiles,
				actualNumFiles,
			)
			continue
		}
		for j, name := range filesAfter {
			if j > 0 && !test.reversible {
				continue
			}
			patt := fmt.Sprintf("%s-[0-9]*-%s.sql", expectedDirections[j], test.name)
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
}

func TestInit(t *testing.T) {
	var err error
	const pathToFile = baseTestOutputDir + "/config.json"
	// setup: file should not exist at first
	if _, err = os.Stat(pathToFile); !os.IsNotExist(err) {
		t.Fatalf("setup error; file at %q should not exist", pathToFile)
	}

	// test 1: file created with this shape
	if err = godfish.Init(pathToFile); err != nil {
		t.Fatalf("something else is wrong with setup; %v", err)
	}
	var conf godfish.MigrationsConf
	if data, err := ioutil.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf); err != nil {
		t.Fatal(err)
	}
	conf.PathToFiles = baseTestOutputDir + "/bar"

	// test2: write data and make sure it's not overwritten after calling Init
	if data, err := json.MarshalIndent(conf, "", "\t"); err != nil {
		t.Fatal(err)
	} else {
		ioutil.WriteFile(
			pathToFile,
			append(data, byte('\n')),
			os.FileMode(0644),
		)
	}
	if err := godfish.Init(pathToFile); err != nil {
		t.Fatal(err)
	}
	var conf2 godfish.MigrationsConf
	if data, err := ioutil.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf2); err != nil {
		t.Fatal(err)
	}
	if conf2.PathToFiles != baseTestOutputDir+"/bar" {
		t.Errorf(
			"expected conf.PathToFiles to be %q, got %q",
			"foo", conf2.PathToFiles,
		)
	}
}
