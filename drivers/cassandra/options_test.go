package cassandra

import (
	"os"
	"testing"

	"github.com/gocql/gocql"
	"github.com/rafaelespinoza/godfish/internal"
)

func TestDriverWithDB(t *testing.T) {
	dsn := os.Getenv(internal.DSNKey)
	if dsn == "" {
		t.Fatalf("define env var %q for these tests", internal.DSNKey)
	}

	t.Run("connection managed outside of library", func(t *testing.T) {
		parsedDSN, err := parseDSN(dsn)
		if err != nil {
			t.Fatal(err)
		}
		cluster := gocql.NewCluster(parsedDSN.hosts...)
		cluster.Keyspace = parsedDSN.keyspace
		db, err := cluster.CreateSession()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(db.Close)
		driver := NewDriver()
		if err = driver.WithDriverOptions(WithDB(db)); err != nil {
			t.Fatal(err)
		}
		if driver.connOwned {
			t.Error("driver not expected to own connection")
		}

		if err = driver.Close(); err != nil {
			t.Errorf("unexpected error closing DB: %s", err.Error())
		}
		if driver.connection == nil {
			t.Fatal("driver connection unexpectedly nil")
		}
		// Despite calling the driver.Close method, the underlying
		// connection remains open.
		query := driver.connection.
			Query("SELECT key FROM system.local LIMIT 1").
			WithContext(t.Context())
		if err = query.Exec(); err != nil {
			t.Errorf("unexpected Ping error: %s", err.Error())
		}
	})

	t.Run("connection managed within library", func(t *testing.T) {
		driver := NewDriver()
		err := driver.Connect(dsn)
		if err != nil {
			t.Fatal(err)
		}
		if !driver.connOwned {
			t.Error("driver expected to own connection")
		}

		t.Cleanup(func() {
			cerr := driver.Close()
			if cerr != nil {
				t.Logf("closing db in cleanup: %s", cerr.Error())
			}
		})

		if err = driver.Close(); err != nil {
			t.Errorf("unexpected error closing DB: %s", err.Error())
		}
		if driver.connection != nil {
			t.Fatal("driver connection unexpectedly non-nil")
		}
	})
}
