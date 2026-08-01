// Alpine linux doesn't have a cassandra client. This command can be used by the
// test environment to check if the server is ready, and when it is, prepare a
// keyspace for the tests.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gocql/gocql"
)

func init() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %s dbhost keyspace\n",
			filepath.Base(os.Args[0]),
		)
	}
}

func main() {
	err := run(context.Background(), os.Args[1:]...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args ...string) error {
	if len(args) < 2 {
		return fmt.Errorf("requires 2 positional args; got %d; %q", len(args), args)
	}

	host, keyspace := args[0], args[1]
	err := setupKeyspace(host, keyspace)
	if err != nil {
		return fmt.Errorf("setting up cassandra keyspace: %w", err)
	}
	slog.Info("ok")
	return nil
}

func setupKeyspace(dbhost, keyspace string) error {
	cluster := gocql.NewCluster(dbhost)
	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	statement := `CREATE KEYSPACE IF NOT EXISTS ` + keyspace + ` WITH replication = {'class':'SimpleStrategy', 'replication_factor': 1}`
	if err = session.Query(statement).Exec(); err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	return nil
}
