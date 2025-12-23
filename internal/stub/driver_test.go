package stub_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	t.Setenv(internal.DSNKey, "stub_dsn")
	test.RunDriverTests(t, stub.NewDriver())
}
