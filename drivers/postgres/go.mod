module github.com/rafaelespinoza/godfish/drivers/postgres

go 1.25.0

require (
	github.com/lib/pq v1.12.3
	github.com/rafaelespinoza/godfish v0.0.0-00010101000000-000000000000
)

require (
	github.com/romantomjak/devslog v1.1.0 // indirect
	github.com/urfave/cli-altsrc/v3 v3.1.0 // indirect
	github.com/urfave/cli/v3 v3.10.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/rafaelespinoza/godfish => ../../
	github.com/rafaelespinoza/godfish/drivers/cassandra => ../cassandra
	github.com/rafaelespinoza/godfish/drivers/mysql => ../mysql
	github.com/rafaelespinoza/godfish/drivers/sqlite3 => ../sqlite3
	github.com/rafaelespinoza/godfish/drivers/sqlserver => ../sqlserver
)
