// Package stub implements godfish interfaces for testing purposes.
package stub

import (
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

func (d *driver) CreateSchemaMigrationsTable() error {
	if d.appliedVersions == nil {
		d.appliedVersions = NewAppliedVersions()
	}
	return nil
}

func (d *driver) Execute(q string, a ...interface{}) error {
	if strings.Contains(q, "invalid SQL") {
		return fmt.Errorf(q)
	}
	return nil
}

func (d *driver) UpdateSchemaMigrations(forward bool, version, label string) error {
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
		ind := internal.Indirection{Value: internal.DirForward, Label: "forward"}
		filename := internal.MakeFilename(version, ind, label)
		mig, err := internal.ParseMigration(filename)
		if err != nil {
			return fmt.Errorf("failed to parse migration from filename %q: %w", filename, err)
		}
		stubbedAV.versions = append(stubbedAV.versions, mig)
	} else {
		for i, v := range stubbedAV.versions {
			if version == v.Version().String() {
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
