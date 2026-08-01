module github.com/rafaelespinoza/godfish/drivers/mysql

go 1.25.0

require (
	github.com/go-sql-driver/mysql v1.10.0
	github.com/rafaelespinoza/godfish v0.0.0-00010101000000-000000000000
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/romantomjak/devslog v1.1.0 // indirect
	github.com/urfave/cli-altsrc/v3 v3.1.0 // indirect
	github.com/urfave/cli/v3 v3.10.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/rafaelespinoza/godfish => ../../
	github.com/rafaelespinoza/godfish/drivers/cassandra => ../cassandra
	github.com/rafaelespinoza/godfish/drivers/postgres => ../postgres
	github.com/rafaelespinoza/godfish/drivers/sqlite3 => ../sqlite3
	github.com/rafaelespinoza/godfish/drivers/sqlserver => ../sqlserver
)
