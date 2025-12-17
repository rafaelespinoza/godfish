package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/cmd"
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

			err := cmd.New(stub.NewDriver(), "test").Run(ctx, combinedArgs)
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

			err := cmd.New(stub.NewDriver(), "test").Run(t.Context(), combinedArgs)
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
