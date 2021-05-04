package sqlite3_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	test.RunDriverTests(t, sqlite3.NewDriver())
}
