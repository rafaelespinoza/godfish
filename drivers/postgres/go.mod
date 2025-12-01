module github.com/rafaelespinoza/godfish/drivers/postgres

go 1.24.0

require (
	github.com/lib/pq v1.10.9
	github.com/rafaelespinoza/godfish v0.12.0
)

require (
	github.com/lmittmann/tint v1.1.2 // indirect
	github.com/rafaelespinoza/alf v0.2.0 // indirect
	github.com/rafaelespinoza/logg v0.1.1 // indirect
)

replace github.com/rafaelespinoza/godfish => ../../
