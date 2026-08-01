package mysql_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/internal/drivertest"
	"github.com/rafaelespinoza/godfish/drivers/mysql"
)

func Test(t *testing.T) {
	drivertest.RunDriverTests(t, mysql.NewDriver())
}
