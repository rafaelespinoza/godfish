package postgres_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/internal"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &postgres.DSN{})
}
