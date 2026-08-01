package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/cmd"
	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
)

func main() {
	root := cmd.New(sqlserver.NewDriver(), sqlserver.SampleDSN)
	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
