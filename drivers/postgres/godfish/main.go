package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	// #nosec G101 -- this is an example to jog the user's memory of the format. Not real credentials.
	const dsnSample = `postgresql://username:password@server_host:5432/db_name?param1=value&paramN=valueN`
	root := cmd.New(postgres.NewDriver(), dsnSample)
	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
