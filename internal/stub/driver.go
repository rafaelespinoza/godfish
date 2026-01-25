// Package stub implements godfish interfaces for testing purposes.
package stub

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

type Driver struct {
	appliedVersions godfish.AppliedVersions
}

func NewDriver() *Driver { return &Driver{} }

func (d *Driver) Name() string             { return "stub" }
func (d *Driver) Connect(dsn string) error { return nil }
func (d *Driver) Close() error             { return nil }

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := cleanIdentifier(migrationsTable); err != nil {
		return err
	}

	if d.appliedVersions == nil {
		d.appliedVersions = NewAppliedVersions()
	}
	return nil
}

func (d *Driver) Execute(ctx context.Context, q string, a ...any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.Contains(q, "invalid SQL") {
		return errors.New(q)
	}
	return nil
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := cleanIdentifier(migrationsTable); err != nil {
		return err
	}

	var stubbedAV *appliedVersions
	av, err := d.AppliedVersions(ctx, migrationsTable)
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

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	d, ok := drv.(*Driver)
	if !ok {
		return
	}
	d.appliedVersions = NewAppliedVersions()
}

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, func(s string) string { return s })
}
