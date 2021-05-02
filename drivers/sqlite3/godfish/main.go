package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/internal/commands"
)

func main() {
	if err := commands.Run(sqlite3.NewDriver()); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
