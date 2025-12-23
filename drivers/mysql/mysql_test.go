package mysql_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	test.RunDriverTests(t, mysql.NewDriver())
}
