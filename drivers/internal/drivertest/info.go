package drivertest

import (
	"context"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/compat"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testInfo(t *testing.T, d driver.Driver, queries testdataQueries) {
	tests := []struct {
		name string
		info compat.InfoFunc
	}{
		{
			name: "Deprecated APIs",
			info: func(
				ctx context.Context,
				driver driver.Driver,
				dirFS fs.FS,
				fwd bool,
				version string,
				w io.Writer,
				format string,
				table string,
			) error {
				return godfish.Info(ctx, driver, dirFS, fwd, version, w, format, table)
			},
		},
		{
			name: "Replacement APIs",
			info: func(
				ctx context.Context,
				driver driver.Driver,
				dirFS fs.FS,
				fwd bool,
				version string,
				w io.Writer,
				format string,
				table string,
			) error {
				opts := compat.MakeMigrationOpts(compat.MigrationOptParams{
					Format:          format,
					MigrationsTable: table,
					Writer:          w,
				})
				return godfish.InfoWith(ctx, driver, dirFS, opts...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { runInfoTests(t, d, queries, test.info) })
	}
}

func runInfoTests(t *testing.T, driver driver.Driver, queries testdataQueries, info compat.InfoFunc) {
	t.Run("migrations on filesystem", func(t *testing.T) {
		stubs := []testDriverStub{
			{
				content: queries.CreateFoos,
				version: formattedTime("12340102030405"),
			},
			{
				content: queries.AlterFoos,
				version: formattedTime("23450102030405"),
			},
			{
				content: queries.CreateBars,
				version: formattedTime("34560102030405"),
			},
		}

		for _, test := range okMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				path := setup(t, driver, stubs, "34560102030405", test.migrationsTable)
				t.Cleanup(func() { teardown(t, driver, path, test.migrationsTable, "foos", "bars") })

				t.Run("forward", func(t *testing.T) {
					dirFS := os.DirFS(path)
					err := info(t.Context(), driver, dirFS, true, "", t.Output(), "tsv", test.migrationsTable)
					if err != nil {
						t.Errorf(
							"could not output info in %s Direction; %v",
							internal.DirForward, err,
						)
					}
				})

				t.Run("reverse", func(t *testing.T) {
					dirFS := os.DirFS(path)
					err := info(t.Context(), driver, dirFS, false, "", t.Output(), "json", test.migrationsTable)
					if err != nil {
						t.Errorf(
							"could not output info in %s Direction; %v",
							internal.DirReverse, err,
						)
					}
				})
			})
		}
	})

	t.Run("embedded", func(t *testing.T) {
		subdir := getTestdataSubdir(driver)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range okMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				if err = info(t.Context(), driver, dirFS, true, "", t.Output(), "json", test.migrationsTable); err != nil {
					t.Fatal(err)
				}

				if err = info(t.Context(), driver, dirFS, false, "", t.Output(), "json", test.migrationsTable); err != nil {
					t.Fatal(err)
				}
			})
		}
	})

	t.Run("invalid migrations table", func(t *testing.T) {
		subdir := getTestdataSubdir(driver)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range invalidMigrationsTableTestCases {
			t.Run(test.name, func(t *testing.T) {
				err = info(t.Context(), driver, dirFS, true, "", nil, "json", test.migrationsTable)
				if !internal.IsInvalidDataError(err) {
					t.Fatalf("expected error (%v) to be an invalid data error", err)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}
			})
		}
	})
}
