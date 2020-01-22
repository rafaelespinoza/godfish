package mysql_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/mysql"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &mysql.DSN{})
}
