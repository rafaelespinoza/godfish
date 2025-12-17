package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	const dsnSample = `sqlserver://user:pass@server_host/instance?database=test1`
	root := cmd.New(sqlserver.NewDriver(), dsnSample)
	if err := root.Run(context.TODO(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
