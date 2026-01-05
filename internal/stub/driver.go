// Package stub implements godfish interfaces for testing purposes.
package stub

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

type driver struct {
	appliedVersions godfish.AppliedVersions
}

func NewDriver() godfish.Driver { return &driver{} }

func (d *driver) Name() string             { return "stub" }
func (d *driver) Connect(dsn string) error { return nil }
func (d *driver) Close() error             { return nil }

func (d *driver) CreateSchemaMigrationsTable(migrationsTable string) error {
	if _, err := cleanIdentifier(migrationsTable); err != nil {
		return err
	}

	if d.appliedVersions == nil {
		d.appliedVersions = NewAppliedVersions()
	}
	return nil
}

func (d *driver) Execute(q string, a ...any) error {
	if strings.Contains(q, "invalid SQL") {
		return errors.New(q)
	}
	return nil
}

func (d *driver) UpdateSchemaMigrations(migrationsTable string, forward bool, version string) error {
	if _, err := cleanIdentifier(migrationsTable); err != nil {
		return err
	}

	var stubbedAV *appliedVersions
	av, err := d.AppliedVersions(migrationsTable)
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

func (d *driver) AppliedVersions(migrationsTable string) (godfish.AppliedVersions, error) {
	if _, err := cleanIdentifier(migrationsTable); err != nil {
		return nil, err
	}

	if d.appliedVersions == nil {
		return nil, godfish.ErrSchemaMigrationsDoesNotExist
	}
	return d.appliedVersions, nil
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

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, func(s string) string { return s })
}
