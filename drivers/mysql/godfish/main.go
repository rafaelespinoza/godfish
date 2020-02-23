package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/internal/commands"
)

func main() {
	var dsn mysql.DSN
	err := commands.Run(&dsn)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
