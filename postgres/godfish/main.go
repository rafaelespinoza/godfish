package main

import (
	"log"
	"os"

	"bitbucket.org/rafaelespinoza/godfish/internal/commands"
	"bitbucket.org/rafaelespinoza/godfish/postgres"
)

func main() {
	var dsn postgres.DSN
	err := commands.Run(&dsn)
	if err != nil {
		log.Printf("%#v\n", err)
		os.Exit(1)
	}
}
