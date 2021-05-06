package cassandra_test

import (
	"testing"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/internal/test"
)

func Test(t *testing.T) {
	queries := test.Queries{
		CreateFoos: test.MigrationContent{
			Forward: "CREATE TABLE foos (id int PRIMARY KEY);",
			Reverse: "DROP TABLE foos;",
		},
		CreateBars: test.MigrationContent{
			Forward: "CREATE TABLE bars (id int PRIMARY KEY);",
			Reverse: "DROP TABLE bars;",
		},
		AlterFoos: test.MigrationContent{
			Forward: "ALTER TABLE foos ADD a varchar;",
			Reverse: "ALTER TABLE foos DROP a;",
		},
	}

	test.RunDriverTests(t, cassandra.NewDriver(), queries)
}
