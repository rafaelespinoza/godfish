package cassandra_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	test.RunDriverTests(t, cassandra.NewDriver())
}
