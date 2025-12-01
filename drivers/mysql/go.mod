module github.com/rafaelespinoza/godfish/drivers/mysql

go 1.24.0

require (
	github.com/go-sql-driver/mysql v1.9.3
	github.com/rafaelespinoza/godfish v0.12.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/lmittmann/tint v1.1.2 // indirect
	github.com/rafaelespinoza/alf v0.2.0 // indirect
	github.com/rafaelespinoza/logg v0.1.1 // indirect
)

replace github.com/rafaelespinoza/godfish => ../../
