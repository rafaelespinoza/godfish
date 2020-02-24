package main

import (
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/internal/commands"
	"github.com/rafaelespinoza/godfish/mysql"
)

func main() {
	var dsn mysql.DSN
	err := commands.Run(&dsn)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
