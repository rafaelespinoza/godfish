package cassandra_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/drivers/internal/drivertest"
)

func Test(t *testing.T) {
	drivertest.RunDriverTests(t, cassandra.NewDriver())
}
