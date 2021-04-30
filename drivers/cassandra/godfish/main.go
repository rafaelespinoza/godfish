package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/internal/commands"
)

func main() {
	if err := commands.Run(cassandra.NewDriver()); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
