package godfish_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
	"github.com/rafaelespinoza/godfish/internal/test"
	"github.com/rafaelespinoza/godfish/testdata"
)

func TestCreateMigrationFiles(t *testing.T) {
	t.Run("err", func(t *testing.T) {
		err := godfish.CreateMigrationFiles("err_test", true, t.TempDir(), "bad", "bad2")
		if err == nil {
			t.Fatal(err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		testdir := t.TempDir()
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
	// There are more detailed tests in the internal/test package.
	dirFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("all the way up and down", func(t *testing.T) {
		driver := stub.NewDriver()
		var err error
		if err = godfish.Migrate(t.Context(), driver, dirFS, true, "", ""); err != nil {
			t.Fatal(err)
		}

		if err = godfish.Migrate(t.Context(), driver, dirFS, false, "", ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("bad version", func(t *testing.T) {
		driver := stub.NewDriver()
		err := godfish.Migrate(t.Context(), driver, dirFS, true, "bad", "")
		if err == nil {
			t.Fatal("expected error")
		}
		t.Log(err)
		if m := err.Error(); !strings.Contains(m, "version") {
			t.Errorf("expected for error (%v) to mention %q", m, "version")
		}
	})
}

func TestApplyMigration(t *testing.T) {
	// There are more detailed tests in the internal/test package.
	okFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("all the way up and down", func(t *testing.T) {
		driver := stub.NewDriver()
		var err error
		if err = godfish.ApplyMigration(t.Context(), driver, okFS, true, "", ""); err != nil {
			t.Fatal(err)
		}

		if err = godfish.ApplyMigration(t.Context(), driver, okFS, false, "", ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("version empty, not found", func(t *testing.T) {
		driver := stub.NewDriver()
		fsys := fstest.MapFS{}
		err := godfish.ApplyMigration(t.Context(), driver, fsys, true, "", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		if !errors.Is(err, internal.ErrNotFound) {
			t.Errorf("expected for error (%v) to be %v", err, internal.ErrNotFound)
		}
	})

	t.Run("version specified, not found", func(t *testing.T) {
		driver := stub.NewDriver()
		err := godfish.ApplyMigration(t.Context(), driver, okFS, true, "1111", "")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		t.Log(err)
		if !errors.Is(err, internal.ErrNotFound) {
			t.Errorf("expected for error (%v) to be %v", err, internal.ErrNotFound)
		}
	})
}

func TestInfo(t *testing.T) {
	okFS, err := fs.Sub(testdata.Migrations, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unknown format does not error out", func(t *testing.T) {
		driver := stub.NewDriver()
		err := godfish.Info(t.Context(), driver, okFS, false, "", os.Stderr, "tea_ess_vee", "")
		if err != nil {
			t.Fatalf("unexpected error, %v", err)
		}
	})

	t.Run("scanned migrations have empty label", func(t *testing.T) {
		// Check that applied migrations inserted prior to a schema upgrade can have
		// there Label fields inferred by inspecting matching filenames.

		driver := &Driver{
			appliedVersionsFn: func(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error) {
				// The migrations in the "database" have empty Label fields.
				applied := stub.NewAppliedVersions(
					internal.Migration{Indirection: internal.Indirection{}, Version: mustParseVersion(t, "1234")},
					internal.Migration{Indirection: internal.Indirection{}, Version: mustParseVersion(t, "2345")},
					internal.Migration{Indirection: internal.Indirection{}, Version: mustParseVersion(t, "3456")},
				)
				return applied, nil
			},
		}

		var buf bytes.Buffer
		err := godfish.Info(t.Context(), driver, okFS, true, "3456", &buf, "json", "")
		if err != nil {
			t.Fatalf("unexpected error, %v", err)
		}

		expLabels := []string{"alpha", "bravo", "charlie"}
		lines := bytes.TrimSpace(buf.Bytes())
		for i, line := range bytes.Split(lines, []byte{'\n'}) {
			var item struct{ Label string }
			if err = json.Unmarshal(line, &item); err != nil {
				t.Fatal(err)
			}
			expLabel := expLabels[i]
			if got := item.Label; got != expLabel {
				t.Errorf("item %d, got %q, expected %q", i, got, expLabel)
			}
		}
	})
}

func TestInit(t *testing.T) {
	var err error
	testOutputDir := t.TempDir()

	pathToFile := filepath.Clean(filepath.Join(testOutputDir, "config.json"))

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
		err = os.WriteFile(pathToFile, append(data, byte('\n')), os.FileMode(0640))
		if err != nil {
			t.Fatal(err)
		}
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
	var thing any = new(sql.Rows)
	if _, ok := thing.(godfish.AppliedVersions); !ok {
		t.Fatalf("expected %T to implement godfish.AppliedVersions", thing)
	}
}

func TestUpgradeSchemaMigrations(t *testing.T) {
	t.Run("error running UpgradeSchemaMigrations", func(t *testing.T) {
		oof := errors.New("OOF")
		driver := &Driver{
			appliedVersionsFn: func(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error) {
				return nil, godfish.ErrSchemaMigrationsMissingColumns
			},
			upgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				return oof
			},
		}
		err := godfish.UpgradeSchemaMigrations(t.Context(), driver, "test")
		if !errors.Is(err, oof) {
			t.Errorf("expected error (%v) to be %v", err, oof)
		}
	})

	t.Run("ok", func(t *testing.T) {
		var calledUpgradeFn bool
		driver := &Driver{
			appliedVersionsFn: func(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error) {
				return nil, godfish.ErrSchemaMigrationsMissingColumns
			},
			upgradeSchemaMigrationsFn: func(ctx context.Context, migrationsTable string) error {
				calledUpgradeFn = true
				return nil
			},
		}
		err := godfish.UpgradeSchemaMigrations(t.Context(), driver, "test")
		if err != nil {
			t.Fatal(err)
		}
		if !calledUpgradeFn {
			t.Errorf("expected to call the upgrade method")
		}
	})
}

type Driver struct {
	*stub.Driver
	appliedVersionsFn         func(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error)
	upgradeSchemaMigrationsFn func(ctx context.Context, migrationsTable string) error
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error) {
	if d.appliedVersionsFn == nil {
		panic("define appliedVersionsFn")
	}
	return d.appliedVersionsFn(ctx, migrationsTable)
}

func (d *Driver) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	if d.upgradeSchemaMigrationsFn == nil {
		panic("define upgradeSchemaMigrationsFn")
	}
	return d.upgradeSchemaMigrationsFn(ctx, migrationsTable)
}

func TestDriver(t *testing.T) {
	// These tests also run in the stub package. They are duplicated here to
	// make test coverage tool consider the tests in godfish.go as covered.
	t.Setenv(internal.DSNKey, "stub_dsn")
	test.RunDriverTests(t, stub.NewDriver())
}

func mustParseVersion(t *testing.T, v string) internal.Version {
	out, err := internal.ParseVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
