package main

import (
	"log"
	"os"

	"bitbucket.org/rafaelespinoza/godfish/internal/commands"
	"bitbucket.org/rafaelespinoza/godfish/mysql"
)

func main() {
	var dsn mysql.DSN
	err := commands.Run(&dsn)
	if err != nil {
		log.Printf("%#v\n", err)
		os.Exit(1)
	}
}
