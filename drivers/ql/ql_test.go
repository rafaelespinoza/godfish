package ql_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/internal/drivertest"
	"github.com/rafaelespinoza/godfish/drivers/ql"
)

func Test(t *testing.T) {
	drivertest.RunDriverTests(t, ql.NewDriver())
}
