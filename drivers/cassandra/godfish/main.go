package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	const dsnSample = `cassandra://server_host:9042/keyspace_name?timeout_ms=2000&connect_timeout_ms=2000`
	root := cmd.New(cassandra.NewDriver(), dsnSample)
	if err := root.Run(context.Background(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
