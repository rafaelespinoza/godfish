package main

import (
	"database/sql"
	"log"
	"os"

	// Imported for the side effect of registering the driver.
	_ "github.com/microsoft/go-mssqldb"
)

var dsn string

func init() {
	log.SetOutput(os.Stderr)

	const key = "DB_DSN"
	if dsn = os.Getenv(key); dsn == "" {
		log.Fatalf("missing required env var %q", key)
	}
}

func main() {
	if err := ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("ok")
}

func ping() error {
	conn, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping()
}
