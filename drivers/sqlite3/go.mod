module github.com/rafaelespinoza/godfish/drivers/sqlite3

go 1.25.0

require (
	github.com/rafaelespinoza/godfish v0.0.0-00010101000000-000000000000
	modernc.org/sqlite v1.54.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/romantomjak/devslog v1.1.0 // indirect
	github.com/urfave/cli-altsrc/v3 v3.1.0 // indirect
	github.com/urfave/cli/v3 v3.10.1 // indirect
	golang.org/x/sys v0.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.74.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace (
	github.com/rafaelespinoza/godfish => ../../
	github.com/rafaelespinoza/godfish/drivers/cassandra => ../cassandra
	github.com/rafaelespinoza/godfish/drivers/mysql => ../mysql
	github.com/rafaelespinoza/godfish/drivers/postgres => ../postgres
	github.com/rafaelespinoza/godfish/drivers/sqlserver => ../sqlserver
)
