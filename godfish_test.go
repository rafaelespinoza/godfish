package godfish_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
	"github.com/rafaelespinoza/godfish/internal/test"
)

const (
	baseTestOutputDir = "/tmp/godfish_test"
	dsnKey            = "DB_DSN"
)

func makeTestDir(t *testing.T, basedir string) (outpath string) {
	if basedir == "" {
		basedir = os.TempDir()
	}
	err := os.MkdirAll(basedir, 0750)
	if err != nil {
		t.Fatal(err)
	}
	outpath, err = os.MkdirTemp(basedir, strings.Replace(t.Name(), "/", "_", -1))
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestCreateMigrationFiles(t *testing.T) {
	t.Run("err", func(t *testing.T) {
		testdir := makeTestDir(t, baseTestOutputDir)
		err := godfish.CreateMigrationFiles("err_test", true, testdir, "bad", "bad2")
		if err == nil {
			t.Fatal(err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		testdir := makeTestDir(t, "")
		err := godfish.CreateMigrationFiles("err_test", true, testdir, "", "")
		if err != nil {
			t.Fatal(err)
		}

		entries, err := os.ReadDir(testdir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("wrong number of entries, got %d, expected %d", len(entries), 2)
		}

		for i, direction := range []string{"forward", "reverse"} {
			got := entries[i].Name()
			if !strings.HasPrefix(got, direction) {
				t.Errorf("expected filename, %q, to have prefix %q", got, direction)
			}
			if !strings.HasSuffix(got, "err_test.sql") {
				t.Errorf("expected filename, %q, to have suffix %q", got, "err_test.sql")
			}
		}
	})
}

func TestMigrate(t *testing.T) {
	t.Run("missing DB_DSN", func(t *testing.T) {
		t.Setenv(dsnKey, "")

		testdir := makeTestDir(t, baseTestOutputDir)
		err := godfish.Migrate(stub.NewDriver(), testdir, false, "")
		if err == nil {
			t.Fatalf("expected an error, got %v", err)
		}
		got := err.Error()
		if !strings.Contains(got, dsnKey) {
			t.Errorf("expected error message %q to mention %q", got, dsnKey)
		}
	})
}

func TestApplyMigration(t *testing.T) {
	t.Run("missing DB_DSN", func(t *testing.T) {
		t.Setenv(dsnKey, "")

		testdir := makeTestDir(t, baseTestOutputDir)
		err := godfish.ApplyMigration(stub.NewDriver(), testdir, false, "")
		if err == nil {
			t.Fatalf("expected an error, got %v", err)
		}
		got := err.Error()
		if !strings.Contains(got, dsnKey) {
			t.Errorf("expected error message %q to mention %q", got, dsnKey)
		}
	})
}

func TestInfo(t *testing.T) {
	t.Run("missing DB_DSN", func(t *testing.T) {
		t.Setenv(dsnKey, "")

		testdir := makeTestDir(t, baseTestOutputDir)
		err := godfish.Info(stub.NewDriver(), testdir, false, "", os.Stderr, "")
		if err == nil {
			t.Fatalf("expected an error, got %v", err)
		}
		got := err.Error()
		if !strings.Contains(got, dsnKey) {
			t.Errorf("expected error message %q to mention %q", got, dsnKey)
		}
	})

	t.Run("unknown format does not error out", func(t *testing.T) {
		t.Setenv(dsnKey, "test")

		testdir := makeTestDir(t, baseTestOutputDir)
		err := godfish.Info(stub.NewDriver(), testdir, false, "", os.Stderr, "tea_ess_vee")
		if err != nil {
			t.Fatalf("unexpected error, %v", err)
		}
	})
}

func TestInit(t *testing.T) {
	var err error
	testOutputDir := makeTestDir(t, "")

	pathToFile := testOutputDir + "/config.json"

	// setup: file should not exist at first
	if _, err = os.Stat(pathToFile); !os.IsNotExist(err) {
		t.Fatalf("setup error; file at %q should not exist", pathToFile)
	}

	// test 1: file created with this shape
	if err = godfish.Init(pathToFile); err != nil {
		t.Fatalf("something else is wrong with setup; %v", err)
	}
	var conf internal.Config
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
	var conf2 internal.Config
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

func TestDriver(t *testing.T) {
	// These tests also run in the stub package. They are duplicated here to
	// make test coverage tool consider the tests in godfish.go as covered.
	t.Setenv("DB_DSN", "stub_dsn")
	test.RunDriverTests(t, stub.NewDriver(), test.DefaultQueries)
}
