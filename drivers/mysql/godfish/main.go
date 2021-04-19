package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/internal/commands"
)

func main() {
	if err := commands.Run(mysql.NewDriver()); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
