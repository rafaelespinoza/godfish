// Package internal defines common functionality available within the library.
package internal

import (
	"errors"
	"log/slog"
)

// Config is for various runtime settings.
type Config struct {
	PathToFiles  string `json:"path_to_files"`
	ForwardLabel string `json:"forward_label"`
	ReverseLabel string `json:"reverse_label"`
}

// LogValue lets this type implement the [slog.LogValuer] interface.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("path_to_files", c.PathToFiles),
		slog.String("forward_label", c.ForwardLabel),
		slog.String("reverse_label", c.ReverseLabel),
	)
}

// General error values to help shape behavior.
var (
	ErrNotFound    = errors.New("not found")
	ErrDataInvalid = errors.New("data invalid")
)

// DSNKey is the name of the environment variable for connecting to the DB.
const DSNKey = "DB_DSN"
