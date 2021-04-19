package stub_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func Test(t *testing.T) {
	var driver stub.Driver
	internal.RunDriverTests(t, &driver)
}
