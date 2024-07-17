package cmd_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish/internal/cmd"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func TestRoot(t *testing.T) {
	ctx := context.Background()
	t.Setenv("DB_DSN", t.Name())
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

			err := cmd.New(stub.NewDriver()).Run(ctx, combinedArgs)
			t.Log(err)
		})
	}
}
