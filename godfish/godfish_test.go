package godfish_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"bitbucket.org/rafaelespinoza/godfish/godfish"
)

const baseTestOutputDir = "/tmp/godfish_test"

func TestMain(m *testing.M) {
	os.MkdirAll(baseTestOutputDir, 0755)
	m.Run()
	os.RemoveAll(baseTestOutputDir)
}

func TestMigrationParams(t *testing.T) {
	var testDir *os.File
	var mig *godfish.MigrationParams
	var err error
	pathToTestDir := baseTestOutputDir + "/" + t.Name()
	if err := os.MkdirAll(pathToTestDir, 0755); err != nil {
		t.Fatalf("error creating test directory %s", pathToTestDir)
		return
	}
	if testDir, err = os.Open(pathToTestDir); err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(pathToTestDir)
	if mig, err = godfish.NewMigrationParams("foo", true, testDir); err != nil {
		t.Error(err)
		return
	}
	if mig.Forward.Direction() != godfish.DirForward {
		t.Errorf(
			"wrong Direction; expected %s, got %s",
			godfish.DirForward, mig.Forward.Direction(),
		)
	}
	if mig.Reverse.Direction() != godfish.DirReverse {
		t.Errorf(
			"wrong Direction; expected %s, got %s",
			godfish.DirReverse, mig.Reverse.Direction(),
		)
	}
	migrations := []godfish.Migration{mig.Forward, mig.Reverse}
	for _, mig := range migrations {
		if mig.Name() != "foo" {
			t.Errorf(
				"wrong Name; expected %s, got %s",
				"foo", mig.Name(),
			)
		}
		if mig.Timestamp().IsZero() {
			t.Error("got empty Timestamp")
		}
	}

	var filesBefore, filesAfter []string
	if filesBefore, err = testDir.Readdirnames(0); err != nil {
		t.Error(err)
		return
	}
	if err = mig.GenerateFiles(); err != nil {
		t.Error(err)
		return
	}
	if filesAfter, err = testDir.Readdirnames(0); err != nil {
		t.Error(err)
		return
	}
	if len(filesAfter)-len(filesBefore) != 2 {
		t.Errorf(
			"expected to generate 2 files, got %d",
			len(filesAfter)-len(filesBefore),
		)
		return
	}
	expectedDirections := []string{"reverse", "forward"}
	for i, name := range filesAfter {
		patt := fmt.Sprintf("%s-[0-9]*-foo.sql", expectedDirections[i])
		if match, err := filepath.Match(patt, name); err != nil {
			t.Error(err)
			return
		} else if !match {
			t.Errorf(
				"expected filename %q to match pattern %q",
				name, patt,
			)
		}
	}
}

func TestInit(t *testing.T) {
	var err error
	const pathToFile = baseTestOutputDir + "/config.json"
	// setup: file should not exist at first
	if _, err = os.Stat(pathToFile); !os.IsNotExist(err) {
		t.Fatalf("setup error; file at %q should not exist", pathToFile)
		return
	}

	// test 1: file created with this shape
	if err = godfish.Init(pathToFile); err != nil {
		t.Fatalf("something else is wrong with setup; %v", err)
		return
	}
	var conf godfish.MigrationsConf
	if data, err := ioutil.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf); err != nil {
		t.Fatal(err)
	}
	conf.DriverName = "foo"
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
	if conf2.DriverName != "foo" {
		t.Errorf(
			"expected conf.DriverName to be %q, got %q",
			"foo", conf2.DriverName,
		)
	}
	if conf2.PathToFiles != baseTestOutputDir+"/bar" {
		t.Errorf(
			"expected conf.PathToFiles to be %q, got %q",
			"foo", conf2.PathToFiles,
		)
	}
}
