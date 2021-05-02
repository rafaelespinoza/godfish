package sqlite3_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/internal"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, sqlite3.NewDriver())
}
