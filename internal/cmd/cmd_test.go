package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func TestRoot(t *testing.T) {
	ctx := context.Background()
	t.Setenv(internal.DSNKey, t.Name())
	testdir := t.TempDir()

	args := [][]string{
		{"help"},
		{"create-migration"},
		{"create-migration", "-h"},
		{"create-migration", "-fwdlabel", "up"},
		{"create-migration", "-revlabel", "down"},
		{"info"},
		{"info", "-h"},
		{"info", "-format", "json"},
		{"info", "-direction", "reverse"},
		{"init", "-conf", filepath.Join(testdir, "test.json")},
		{"init", "-h"},
		{"migrate"},
		{"migrate", "-h"},
		{"remigrate"},
		{"remigrate", "-h"},
		{"rollback"},
		{"rollback", "-h"},
		{"version"},
		{"version", "-json"},
		{"version", "-h"},
	}
	for _, cmdAndArgs := range args {
		t.Run(strings.Join(cmdAndArgs, " "), func(t *testing.T) {
			godfishFlags := []string{
				"-conf", filepath.Join(testdir, ".godfish.json"),
				"-files", testdir,
			}
			combinedArgs := append(godfishFlags, cmdAndArgs...)

			err := New(stub.NewDriver(), "test").Run(ctx, combinedArgs)
			t.Log(err)
		})
	}
}

func TestDBDSN(t *testing.T) {
	testdir := t.TempDir()

	tests := []struct {
		name    string
		envVal  string
		flagVal string
		exp     string
		expErr  bool
	}{
		{
			name:   "env var already set, no flag value",
			envVal: "env_val",
			exp:    "env_val",
		},
		{
			name:    "env var already set, flag value overrides env val",
			envVal:  "env_val",
			flagVal: "flag_val",
			exp:     "flag_val",
		},
		{
			name:    "env var not set, flag value set",
			flagVal: "flag_val",
			exp:     "flag_val",
		},
		{
			name:   "env var not set, flag value not set",
			expErr: true,
		},
		{
			name:    "bad flag value set",
			flagVal: "bad\x00",
			expErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(internal.DSNKey, test.envVal)

			godfishFlags := []string{"-files", testdir}
			if test.flagVal != "" {
				godfishFlags = append(godfishFlags, "-dsn", test.flagVal)
			}
			combinedArgs := append(godfishFlags, "info")

			err := New(stub.NewDriver(), "test").Run(t.Context(), combinedArgs)
			if !test.expErr && err != nil {
				t.Fatal(err)
			} else if test.expErr && err == nil {
				t.Fatal("expected an error but got nil")
			} else if test.expErr && err != nil {
				if !strings.Contains(err.Error(), internal.DSNKey) {
					t.Fatalf("expected for error message (%s) to mention %s", err.Error(), internal.DSNKey)
				}
				return // OK
			}

			got, defined := os.LookupEnv(internal.DSNKey)
			if !defined {
				t.Fatalf("expected for env var %s to be defined", internal.DSNKey)
			}
			if got != test.exp {
				t.Errorf("wrong value; got %q, expected %q", got, test.exp)
			}
		})
	}
}

func TestConfiguration(t *testing.T) {
	type testCase struct {
		name               string
		conf               *internal.Config
		args               []string
		expPathToFiles     string
		expMigrationsTable string
		expError           error
	}

	runTest := func(t *testing.T, test testCase) {
		// Set up
		if test.conf != nil {
			configFile := filepath.Join(t.TempDir(), ".godfish.json")
			if err := godfish.Init(configFile); err != nil {
				t.Fatal(err)
			}
			rawConfigData, err := json.Marshal(test.conf)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configFile, rawConfigData, 0600); err != nil {
				t.Fatal(err)
			}

			test.args = append(test.args, "-conf", configFile)
		}
		// Need to pass some valid command here so that we can exercise the config
		// resolution code. If nothing is passed at this point, then a usage message
		// is printed, and we'd circumvent the code to test.
		test.args = append(test.args, "version", "-json")

		// Run code to test
		err := New(stub.NewDriver(), "test").Run(t.Context(), test.args)

		// Check results
		if test.expError != nil && err == nil {
			t.Fatal("expected an error but got nil")
		} else if test.expError == nil && err != nil {
			t.Fatalf("unexpected error %v", err)
		} else if test.expError != nil && err != nil {
			if !errors.Is(err, test.expError) {
				t.Fatalf("expected error (%v) to match %v", err, test.expError)
			}
			return // OK
		}

		if commonArgs.Files != test.expPathToFiles {
			t.Errorf("wrong Files; got %q, expected %q", commonArgs.Files, test.expPathToFiles)
		}

		if commonArgs.MigrationsTable != test.expMigrationsTable {
			t.Errorf("wrong MigrationsTable; got %q, expected %q", commonArgs.MigrationsTable, test.expMigrationsTable)
		}
	}

	t.Run("files", func(t *testing.T) {
		tests := []testCase{
			{
				name:           "config unspecified, flag unspecified",
				conf:           nil,
				args:           []string{},
				expPathToFiles: "",
			},
			{
				name:           "config unspecified, flag specified",
				conf:           nil,
				args:           []string{"-files", "flag-val"},
				expPathToFiles: "flag-val",
			},
			{
				name:           "config specified, flag unspecified",
				conf:           &internal.Config{PathToFiles: "config-file-val"},
				args:           []string{},
				expPathToFiles: "config-file-val",
			},
			{
				name:           "config specified, flag specified",
				conf:           &internal.Config{PathToFiles: "config=file-val"},
				args:           []string{"-files", "flag-val"},
				expPathToFiles: "flag-val",
			},
			{
				name:           "config specified, flag specified but empty",
				conf:           &internal.Config{PathToFiles: "config=file-val"},
				args:           []string{"-files", ""},
				expPathToFiles: "",
			},
		}

		for _, test := range tests {
			test.expMigrationsTable = internal.DefaultMigrationsTableName
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})

	t.Run("migrations-table", func(t *testing.T) {
		tests := []testCase{
			{
				name:               "config unspecified, flag unspecified",
				conf:               nil,
				args:               []string{},
				expMigrationsTable: internal.DefaultMigrationsTableName,
			},
			{
				name:               "config unspecified, flag specified",
				conf:               nil,
				args:               []string{"-migrations-table", "flag-val"},
				expMigrationsTable: "flag-val",
			},
			{
				name:               "config specified, flag unspecified",
				conf:               &internal.Config{MigrationsTable: "config-file-val"},
				args:               []string{},
				expMigrationsTable: "config-file-val",
			},
			{
				name:               "config specified, flag specified",
				conf:               &internal.Config{MigrationsTable: "config=file-val"},
				args:               []string{"-migrations-table", "flag-val"},
				expMigrationsTable: "flag-val",
			},
			{
				name:               "config specified, flag specified but empty",
				conf:               &internal.Config{MigrationsTable: "config=file-val"},
				args:               []string{"-migrations-table", ""},
				expMigrationsTable: "",
			},
		}

		for _, test := range tests {
			test.args = append([]string{"-files", os.TempDir()}, test.args...)
			test.expPathToFiles = os.TempDir()
			t.Run(test.name, func(t *testing.T) { runTest(t, test) })
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("loading bad config data", func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), ".godfish.json")
			if err := godfish.Init(configFile); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configFile, []byte(`BAD`), 0600); err != nil {
				t.Fatal(err)
			}

			runTest(t, testCase{
				args:     []string{"-conf", configFile},
				expError: internal.ErrDataInvalid,
			})
		})

		t.Run("other error", func(t *testing.T) {
			runTest(t, testCase{
				args:     []string{"-conf", t.TempDir()},
				expError: errReadConfig,
			})
		})
	})
}
