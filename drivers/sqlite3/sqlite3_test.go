package sqlite3_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/internal/drivertest"
	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
)

func Test(t *testing.T) {
	drivertest.RunDriverTests(t, sqlite3.NewDriver())
}
