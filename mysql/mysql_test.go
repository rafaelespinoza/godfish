package mysql_test

import (
	"testing"

	"bitbucket.org/rafaelespinoza/godfish/internal"
	"bitbucket.org/rafaelespinoza/godfish/mysql"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &mysql.DSN{})
}
