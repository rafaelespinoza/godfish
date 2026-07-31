package compat_test

import (
	"io"
	"testing"

	"github.com/rafaelespinoza/godfish/internal/compat"
)

func TestMakeMigrationOpts(t *testing.T) {
	tests := []struct {
		name      string
		params    compat.MigrationOptParams
		expLength int
	}{
		{
			name:      "all zero values",
			params:    compat.MigrationOptParams{},
			expLength: 0,
		},
		{
			name:      "only Format set",
			params:    compat.MigrationOptParams{Format: "json"},
			expLength: 1,
		},
		{
			name:      "only MigrationsTable set",
			params:    compat.MigrationOptParams{MigrationsTable: "schema_migrations"},
			expLength: 1,
		},
		{
			name:      "only TargetVersion set",
			params:    compat.MigrationOptParams{TargetVersion: "20260101"},
			expLength: 1,
		},
		{
			name:      "only Writer set",
			params:    compat.MigrationOptParams{Writer: io.Discard},
			expLength: 1,
		},
		{
			name:      "only FwdLabel set",
			params:    compat.MigrationOptParams{ForwardLabel: "migrate"},
			expLength: 1,
		},
		{
			name:      "only RevLabel set",
			params:    compat.MigrationOptParams{ReverseLabel: "rollback"},
			expLength: 1,
		},
		{
			name:      "only FilenameExt set",
			params:    compat.MigrationOptParams{FilenameExt: ".abc"},
			expLength: 1,
		},
		{
			name: "partial options set",
			params: compat.MigrationOptParams{
				Format:        "tsv",
				TargetVersion: "20260101",
			},
			expLength: 2,
		},
		{
			name: "all options set",
			params: compat.MigrationOptParams{
				Format:          "json",
				MigrationsTable: "schema_migrations",
				TargetVersion:   "20260101",
				Writer:          io.Discard,
			},
			expLength: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := compat.MakeMigrationOpts(test.params)
			if got := len(opts); got != test.expLength {
				t.Errorf("wrong length; got %d, expected %d", got, test.expLength)
			}
		})
	}
}
