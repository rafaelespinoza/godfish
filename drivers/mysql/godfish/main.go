package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	const dsnSample = `username:password@tcp(server_host)/db_name?param1=value&paramN=valueN`
	root := cmd.New(mysql.NewDriver(), dsnSample)
	if err := root.Run(context.TODO(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
