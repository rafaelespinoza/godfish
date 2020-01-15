package postgres_test

import (
	"os"
	"testing"

	"bitbucket.org/rafaelespinoza/godfish/internal"
	"bitbucket.org/rafaelespinoza/godfish/postgres"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, postgres.Params{
		Encoding: "UTF8",
		Host:     "localhost",
		Name:     "godfish_test",
		Pass:     os.Getenv("DB_PASSWORD"),
		Port:     os.Getenv("DB_PORT"),
		User:     "godfish",
	})
}
