package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	root := cmd.New(sqlserver.NewDriver())
	if err := root.Run(context.TODO(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
