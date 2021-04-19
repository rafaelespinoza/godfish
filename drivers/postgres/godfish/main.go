package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/internal/commands"
)

func main() {
	if err := commands.Run(postgres.NewDriver()); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
