package driver_test

import (
	"database/sql"
	"testing"

	"github.com/rafaelespinoza/godfish/driver"
)

func TestAppliedVersions(t *testing.T) {
	// Regression test on the API. It's supposed to wrap this type from the
	// standard library for the most common cases.
	var thing any = new(sql.Rows)
	if _, ok := thing.(driver.AppliedVersions); !ok {
		t.Fatalf("expected %T to implement driver.AppliedVersions", thing)
	}
}
