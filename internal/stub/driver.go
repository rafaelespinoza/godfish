// Package stub implements godfish interfaces for testing purposes.
package stub

import (
	"fmt"
	"strings"

	"github.com/rafaelespinoza/godfish"
)

type driver struct {
	appliedVersions godfish.AppliedVersions
	err             error
	errorOnExecute  error
}

func NewDriver() godfish.Driver { return &driver{} }

func (d *driver) Name() string             { return "stub" }
func (d *driver) Connect(dsn string) error { return d.err }
func (d *driver) Close() error             { return d.err }

func (d *driver) CreateSchemaMigrationsTable() error {
	if d.appliedVersions == nil {
		d.appliedVersions = NewAppliedVersions()
	}
	return d.err
}

func (d *driver) Execute(q string, a ...interface{}) error {
	if strings.Contains(q, "invalid SQL") {
		return fmt.Errorf(q)
	}
	return d.errorOnExecute
}

func (d *driver) UpdateSchemaMigrations(forward bool, version string) error {
	var stubbedAV *appliedVersions
	av, err := d.AppliedVersions()
	if err != nil {
		return err
	}
	switch val := av.(type) {
	case *appliedVersions:
		stubbedAV = val
	case nil:
		return godfish.ErrSchemaMigrationsDoesNotExist
	default:
		return fmt.Errorf(
			"if you assign anything to this field, make it a %T", stubbedAV,
		)
	}
	if forward {
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

func (d *driver) AppliedVersions() (godfish.AppliedVersions, error) {
	if d.appliedVersions == nil {
		return nil, godfish.ErrSchemaMigrationsDoesNotExist
	}
	return d.appliedVersions, d.err
}

// Teardown resets the stub driver in tests. All other Driver implementations
// pass through without effect.
func Teardown(drv godfish.Driver) {
	d, ok := drv.(*driver)
	if !ok {
		return
	}
	d.appliedVersions = NewAppliedVersions()
}
