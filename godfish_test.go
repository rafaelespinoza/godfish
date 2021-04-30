package godfish_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/rafaelespinoza/godfish"
)

const baseTestOutputDir = "/tmp/godfish_test"

func TestInit(t *testing.T) {
	var err error
	testOutputDir := baseTestOutputDir + "/" + t.Name()
	if err = os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatal(err)
	}

	var pathToFile string
	if outdirPath, err := os.MkdirTemp(testOutputDir, ""); err != nil {
		t.Fatal(err)
	} else {
		pathToFile = outdirPath + "/config.json"
	}

	// setup: file should not exist at first
	if _, err = os.Stat(pathToFile); !os.IsNotExist(err) {
		t.Fatalf("setup error; file at %q should not exist", pathToFile)
	}

	// test 1: file created with this shape
	if err = godfish.Init(pathToFile); err != nil {
		t.Fatalf("something else is wrong with setup; %v", err)
	}
	var conf godfish.Config
	if data, err := os.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf); err != nil {
		t.Fatal(err)
	}
	conf.PathToFiles = testOutputDir + "/bar"

	// test2: write data and make sure it's not overwritten after calling Init
	if data, err := json.MarshalIndent(conf, "", "\t"); err != nil {
		t.Fatal(err)
	} else {
		os.WriteFile(
			pathToFile,
			append(data, byte('\n')),
			os.FileMode(0644),
		)
	}
	if err := godfish.Init(pathToFile); err != nil {
		t.Fatal(err)
	}
	var conf2 godfish.Config
	if data, err := os.ReadFile(pathToFile); err != nil {
		t.Fatal(err)
	} else if err = json.Unmarshal(data, &conf2); err != nil {
		t.Fatal(err)
	}
	if conf2.PathToFiles != testOutputDir+"/bar" {
		t.Errorf(
			"expected conf.PathToFiles to be %q, got %q",
			"foo", conf2.PathToFiles,
		)
	}
}

func TestAppliedVersions(t *testing.T) {
	// Regression test on the API. It's supposed to wrap this type from the
	// standard library for the most common cases.
	var thing interface{} = new(sql.Rows)
	if _, ok := thing.(godfish.AppliedVersions); !ok {
		t.Fatalf("expected %T to implement godfish.AppliedVersions", thing)
	}
}
