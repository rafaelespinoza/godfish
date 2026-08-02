package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/ql"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	root := cmd.New(ql.NewDriver(), ql.SampleDSN)
	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
