package stub_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &stub.DSN{})
}
