package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	const dsnSample = `file:///path/to/db.sqlite`
	root := cmd.New(sqlite3.NewDriver(), dsnSample)
	if err := root.Run(context.Background(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
