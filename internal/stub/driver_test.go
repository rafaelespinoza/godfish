package stub_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal/stub"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	test.RunDriverTests(t, stub.NewDriver(), test.DefaultQueries)
}
