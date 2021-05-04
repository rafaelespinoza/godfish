package test

import (
	"testing"

	"github.com/rafaelespinoza/godfish"
)

func testInfo(t *testing.T, driver godfish.Driver) {
	path, err := setup(driver, t.Name(), _DefaultTestDriverStubs, "34560102030405")
	if err != nil {
		t.Errorf("could not setup test; %v", err)
		return
	}
	defer teardown(driver, path, "foos", "bars")

	t.Run("forward", func(t *testing.T) {
		err := godfish.Info(driver, path, godfish.DirForward, "")
		if err != nil {
			t.Errorf(
				"could not output info in %s Direction; %v",
				godfish.DirForward, err,
			)
		}
	})

	t.Run("reverse", func(t *testing.T) {
		err := godfish.Info(driver, path, godfish.DirReverse, "")
		if err != nil {
			t.Errorf(
				"could not output info in %s Direction; %v",
				godfish.DirReverse, err,
			)
		}
	})
}
