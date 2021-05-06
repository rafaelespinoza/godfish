package postgres_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	test.RunDriverTests(t, postgres.NewDriver(), test.DefaultQueries)
}
