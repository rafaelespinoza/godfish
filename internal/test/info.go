package test

import (
	"os"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal/info"
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
	path, err := setup(driver, t.Name(), stubs, "34560102030405")
	if err != nil {
		t.Errorf("could not setup test; %v", err)
		return
	}
	defer teardown(driver, path, "foos", "bars")

	t.Run("forward", func(t *testing.T) {
		err := godfish.Info(driver, path, godfish.DirForward, "", info.NewTSV(os.Stderr))
		if err != nil {
			t.Errorf(
				"could not output info in %s Direction; %v",
				godfish.DirForward, err,
			)
		}
	})

	t.Run("reverse", func(t *testing.T) {
		err := godfish.Info(driver, path, godfish.DirReverse, "", info.NewJSON(os.Stderr))
		if err != nil {
			t.Errorf(
				"could not output info in %s Direction; %v",
				godfish.DirReverse, err,
			)
		}
	})
}
