package mysql_test

import (
	"os"
	"testing"

	"bitbucket.org/rafaelespinoza/godfish/internal"
	"bitbucket.org/rafaelespinoza/godfish/mysql"
)

func Test(t *testing.T) {
	internal.RunDriverTests(t, mysql.Params{
		Encoding: "UTF8",
		Host:     "localhost",
		Name:     "godfish_test",
		Pass:     os.Getenv("DB_PASSWORD"),
		Port:     os.Getenv("DB_PORT"),
		User:     "godfish",
	})
}
