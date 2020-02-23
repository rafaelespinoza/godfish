package mysql_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/internal"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &mysql.DSN{})
}
