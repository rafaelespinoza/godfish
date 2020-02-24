package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/internal/commands"
	"github.com/rafaelespinoza/godfish/postgres"
)

func main() {
	var dsn postgres.DSN
	err := commands.Run(&dsn)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
