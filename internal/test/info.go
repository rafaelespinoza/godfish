package test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/testdata"
)

func testInfo(t *testing.T, driver godfish.Driver, queries testdataQueries) {
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
					err := godfish.Info(t.Context(), driver, dirFS, true, "", os.Stderr, "tsv", test.migrationsTable)
					if err != nil {
						t.Errorf(
							"could not output info in %s Direction; %v",
							internal.DirForward, err,
						)
					}
				})

				t.Run("reverse", func(t *testing.T) {
					dirFS := os.DirFS(path)
					err := godfish.Info(t.Context(), driver, dirFS, false, "", os.Stderr, "json", test.migrationsTable)
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
				var buf bytes.Buffer
				if err = godfish.Info(t.Context(), driver, dirFS, true, "", &buf, "json", test.migrationsTable); err != nil {
					t.Fatal(err)
				}
				t.Log(buf.String())

				if err = godfish.Info(t.Context(), driver, dirFS, false, "", &buf, "json", test.migrationsTable); err != nil {
					t.Fatal(err)
				}
				t.Log(buf.String())
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
				err = godfish.Info(t.Context(), driver, dirFS, true, "", nil, "json", test.migrationsTable)
				if !errors.Is(err, internal.ErrDataInvalid) {
					t.Fatalf("expected error (%v) to match %v", err, internal.ErrDataInvalid)
				}
				if msg := err.Error(); !strings.Contains(msg, "identifier") {
					t.Errorf("expected for error message (%q) to mention %q", msg, "identifier")
				}
			})
		}
	})
}
