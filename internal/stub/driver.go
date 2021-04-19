// Package stub implements godfish interfaces for testing purposes.
package stub

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rafaelespinoza/godfish"
)

type Driver struct {
	connection      *sql.DB
	appliedVersions godfish.AppliedVersions
	err             error
	errorOnExecute  error
}

var _ godfish.Driver = (*Driver)(nil)

func (d *Driver) Name() string                        { return "stub" }
func (d *Driver) Connect(dsn string) (*sql.DB, error) { return d.connection, d.err }
func (d *Driver) Close() error                        { return d.err }
func (d *Driver) CreateSchemaMigrationsTable() error {
	if d.appliedVersions == nil {
		d.appliedVersions = MakeAppliedVersions()
	}
	return d.err
}
func (d *Driver) Execute(q string, a ...interface{}) error {
	if strings.Contains(q, "invalid SQL") {
		return fmt.Errorf(q)
	}
	return d.errorOnExecute
}
func (d *Driver) UpdateSchemaMigrations(direction godfish.Direction, version string) error {
	var stubbedAV *AppliedVersions
	av, err := d.AppliedVersions()
	if err != nil {
		return err
	}
	switch val := av.(type) {
	case *AppliedVersions:
		stubbedAV = val
	case nil:
		return godfish.ErrSchemaMigrationsDoesNotExist
	default:
		return fmt.Errorf(
			"if you assign anything to this field, make it a %T", stubbedAV,
		)
	}
	if direction == godfish.DirForward {
		stubbedAV.versions = append(stubbedAV.versions, version)
	} else {
		for i, v := range stubbedAV.versions {
			if v == version {
				stubbedAV.versions = append(
					stubbedAV.versions[:i],
					stubbedAV.versions[i+1:]...,
				)
			}
		}
	}
	d.appliedVersions = stubbedAV
	return nil
}
func (d *Driver) AppliedVersions() (godfish.AppliedVersions, error) {
	if d.appliedVersions == nil {
		return nil, godfish.ErrSchemaMigrationsDoesNotExist
	}
	return d.appliedVersions, d.err
}

func (d *Driver) Teardown() {
	d.appliedVersions = MakeAppliedVersions()
}

type AppliedVersions struct {
	counter  int
	versions []string
}

var _ godfish.AppliedVersions = (*AppliedVersions)(nil)

func MakeAppliedVersions(migrations ...godfish.Migration) godfish.AppliedVersions {
	out := AppliedVersions{
		versions: make([]string, len(migrations)),
	}
	for i, mig := range migrations {
		out.versions[i] = mig.Version().String()
	}
	return &out
}

func (r *AppliedVersions) Close() error {
	r.counter = 0
	return nil
}
func (r *AppliedVersions) Next() bool { return r.counter < len(r.versions) }
func (r *AppliedVersions) Scan(dest ...interface{}) error {
	var out *string
	if s, ok := dest[0].(*string); !ok {
		return fmt.Errorf("pass in *string; got %T", s)
	} else if !r.Next() {
		return fmt.Errorf("no more results")
	} else {
		out = s
	}
	*out = r.versions[r.counter]
	r.counter++
	return nil
}
