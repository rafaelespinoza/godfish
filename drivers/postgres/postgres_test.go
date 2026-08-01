package postgres_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/internal/drivertest"
	"github.com/rafaelespinoza/godfish/drivers/postgres"
)

func Test(t *testing.T) {
	drivertest.RunDriverTests(t, postgres.NewDriver())
}
