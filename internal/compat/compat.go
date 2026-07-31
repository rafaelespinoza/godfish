// Package compat is some bridge code to help ease transition from
// deprecated APIs to replacement APIs.
package compat

import (
	"context"
	"io"
	"io/fs"
	"log/slog"

	"github.com/rafaelespinoza/godfish"
)

type MigrationOptParams struct {
	Format          string
	MigrationsTable string
	TargetVersion   string
	Writer          io.Writer

	// for creating migration files

	ForwardLabel string
	ReverseLabel string
	FilenameExt  string
}

// LogValue lets this type implement the [slog.LogValuer] interface.
func (m MigrationOptParams) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("format", m.Format),
		slog.String("migrations_table", m.MigrationsTable),
		slog.String("target_version", m.TargetVersion),
		slog.Bool("writer_nil?", m.Writer == nil),
		slog.String("forward_label", m.ForwardLabel),
		slog.String("reverse_label", m.ReverseLabel),
		slog.String("filename_ext", m.FilenameExt),
	)
}

// MakeMigrationOpts is syntantic sugar to produce a valid list of migration
// options, each of which would return an error if the wrapped value is zero
// value.
func MakeMigrationOpts(m MigrationOptParams) []godfish.Opter {
	out := []godfish.Opter{}
	if m.Format != "" {
		out = append(out, godfish.WithFormat(m.Format))
	}
	if m.MigrationsTable != "" {
		out = append(out, godfish.WithMigrationsTable(m.MigrationsTable))
	}
	if m.TargetVersion != "" {
		out = append(out, godfish.WithTargetVersion(m.TargetVersion))
	}
	if m.Writer != nil {
		out = append(out, godfish.WithWriter(m.Writer))
	}

	// these are only relevant for creating migration files.
	if m.ForwardLabel != "" {
		out = append(out, godfish.WithForwardLabel(m.ForwardLabel))
	}
	if m.ReverseLabel != "" {
		out = append(out, godfish.WithReverseLabel(m.ReverseLabel))
	}
	if m.FilenameExt != "" {
		out = append(out, godfish.WithFilenameExtension(m.FilenameExt))
	}

	return out
}

// generalized function types to help test that deprecated APIs work the same as
// their replacement APIs.
//
// These types can be removed, and tests adjusted to use the replacement API
// after removing the deprecated funcs.
type (
	CreateMigrationFunc func(
		migrationName string,
		reversible bool,
		dirpath string,
		fwdlabel string,
		revlabel string,
		extension string,
	) error

	MigrateFunc func(
		ctx context.Context,
		driver godfish.Driver,
		dirFS fs.FS,
		version string,
		table string,
	) error

	RollbackFunc func(
		ctx context.Context,
		driver godfish.Driver,
		dirFS fs.FS,
		version string,
		table string,
	) error

	InfoFunc func(
		ctx context.Context,
		driver godfish.Driver,
		dirFS fs.FS,
		fwd bool,
		version string,
		w io.Writer,
		format string,
		table string,
	) error

	UpgradeSchemaFunc func(
		ctx context.Context,
		driver godfish.Driver,
		table string,
	) error
)
