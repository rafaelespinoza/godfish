package test

import (
	"os"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func testInfo(t *testing.T, driver godfish.Driver, queries Queries) {
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
		err := godfish.Info(driver, path, true, "", os.Stderr, "tsv")
		if err != nil {
			t.Errorf(
				"could not output info in %s Direction; %v",
				internal.DirForward, err,
			)
		}
	})

	t.Run("reverse", func(t *testing.T) {
		err := godfish.Info(driver, path, false, "", os.Stderr, "json")
		if err != nil {
			t.Errorf(
				"could not output info in %s Direction; %v",
				internal.DirReverse, err,
			)
		}
	})
}
