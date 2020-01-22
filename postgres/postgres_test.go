package postgres_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/postgres"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &postgres.DSN{})
}
