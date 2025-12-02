// Alpine linux doesn't have a cassandra client. This command can be used by the
// test environment to check if the server is ready, and when it is, prepare a
// keyspace for the tests.
package main

import (
	"log"
	"os"

	"github.com/gocql/gocql"
)

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	if len(os.Args) < 3 {
		log.Printf("requires 2 positional args; got %d; %#v\n", len(os.Args), os.Args)
		log.Fatalf("Usage: %s dbhost keyspace", os.Args[0])
	}
	host, keyspace := os.Args[1], os.Args[2]

	err := setupKeyspace(host, keyspace)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("ok")
}

func setupKeyspace(dbhost, keyspace string) error {
	cluster := gocql.NewCluster(dbhost)
	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}
	defer session.Close()

	statement := `CREATE KEYSPACE IF NOT EXISTS ` + keyspace + ` WITH replication = {'class':'SimpleStrategy', 'replication_factor': 1}`
	return session.Query(statement).Exec()
}
