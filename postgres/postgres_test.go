package postgres_test

import (
	"testing"

	"bitbucket.org/rafaelespinoza/godfish/internal"
	"bitbucket.org/rafaelespinoza/godfish/postgres"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, &postgres.DSN{})
}
