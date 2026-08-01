package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/cmd"
	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
)

func main() {
	root := cmd.New(sqlite3.NewDriver(), sqlite3.SampleDSN)
	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
