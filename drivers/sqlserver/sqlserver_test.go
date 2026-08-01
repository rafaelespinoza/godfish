package sqlserver_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/internal/drivertest"
	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
)

func Test(t *testing.T) {
	drivertest.RunDriverTests(t, sqlserver.NewDriver())
}
