package main

import (
	"context"
	"log"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/internal/cmd"
)

func main() {
	root := cmd.New(mysql.NewDriver())
	if err := root.Run(context.TODO(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
