package sqlserver_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	test.RunDriverTests(t, sqlserver.NewDriver())
}
