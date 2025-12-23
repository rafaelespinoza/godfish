package test

import (
	"bytes"
	"io/fs"
	"os"
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
		path := setup(t, driver, stubs, "34560102030405")
		defer teardown(t, driver, path, "foos", "bars")

		t.Run("forward", func(t *testing.T) {
			dirFS := os.DirFS(path)
			err := godfish.Info(driver, dirFS, true, "", os.Stderr, "tsv")
			if err != nil {
				t.Errorf(
					"could not output info in %s Direction; %v",
					internal.DirForward, err,
				)
			}
		})

		t.Run("reverse", func(t *testing.T) {
			dirFS := os.DirFS(path)
			err := godfish.Info(driver, dirFS, false, "", os.Stderr, "json")
			if err != nil {
				t.Errorf(
					"could not output info in %s Direction; %v",
					internal.DirReverse, err,
				)
			}
		})
	})

	t.Run("embedded", func(t *testing.T) {
		subdir := getTestdataSubdir(driver)
		dirFS, err := fs.Sub(testdata.Migrations, subdir)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		if err = godfish.Info(driver, dirFS, true, "", &buf, "json"); err != nil {
			t.Fatal(err)
		}
		t.Log(buf.String())

		if err = godfish.Info(driver, dirFS, false, "", &buf, "json"); err != nil {
			t.Fatal(err)
		}
		t.Log(buf.String())
	})
}
