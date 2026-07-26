// Package stub implements godfish interfaces for testing purposes.
package stub

import (
	"context"

	"github.com/rafaelespinoza/godfish"
)

// Double is a test double for [Driver] that invokes the function
// field corresponding to the interface method. For example, the interface
// method, Execute, corresponds to the field ExecuteFn.
//
// If any of its interface methods are invoked and the corresponding
// function field is unset, then that method panics with a message to
// remind you to define the field.
type Double struct {
	NameFn                    func() string
	AppliedVersionsFn         func(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error)
	CreateSchemaMigrationsFn  func(ctx context.Context, migrationsTable string) error
	ExecuteFn                 func(ctx context.Context, q string, a ...any) error
	UpdateSchemaMigrationsFn  func(ctx context.Context, migrationsTable string, forward bool, version, label string) error
	UpgradeSchemaMigrationsFn func(ctx context.Context, migrationsTable string) error
}

func (d *Double) Name() string {
	if d.NameFn == nil {
		panic("define NameFn")
	}
	return d.NameFn()
}

func (d *Double) AppliedVersions(ctx context.Context, migrationsTable string) (godfish.AppliedVersions, error) {
	if d.AppliedVersionsFn == nil {
		panic("define AppliedVersionsFn")
	}
	return d.AppliedVersionsFn(ctx, migrationsTable)
}

func (d *Double) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) error {
	if d.CreateSchemaMigrationsFn == nil {
		panic("define CreateSchemaMigrationsTableFn")
	}
	return d.CreateSchemaMigrationsFn(ctx, migrationsTable)
}

func (d *Double) Execute(ctx context.Context, q string, a ...any) error {
	if d.ExecuteFn == nil {
		panic("define ExecuteFn")
	}
	return d.ExecuteFn(ctx, q, a...)
}

func (d *Double) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
	if d.UpdateSchemaMigrationsFn == nil {
		panic("define UpdateSchemaMigrationsFn")
	}
	return d.UpdateSchemaMigrationsFn(ctx, migrationsTable, forward, version, label)
}

func (d *Double) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	if d.UpgradeSchemaMigrationsFn == nil {
		panic("define UpgradeSchemaMigrationsFn")
	}
	return d.UpgradeSchemaMigrationsFn(ctx, migrationsTable)
}
