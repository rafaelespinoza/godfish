package log

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Run("nil output writer", func(t *testing.T) {
		expectedLevel := slog.LevelError
		logger := newLogger(nil, "", "")

		if logger.Enabled(context.Background(), expectedLevel) {
			t.Errorf("did not expect output logger to be enabled at level %q", expectedLevel)
		}
	})

	t.Run("output writer is non-empty", func(t *testing.T) {
		var buf bytes.Buffer
		level := slog.LevelInfo
		logger := newLogger(&buf, level.String(), "JSON")

		if !logger.Enabled(context.Background(), level) {
			t.Errorf("expected output logger to be enabled at level %q", level)
		}
		logger.LogAttrs(context.Background(), level, "test")

		if buf.Len() < 1 {
			t.Errorf("expected some data written to buffer but got none")
		}
	})

	t.Run("invalid log level will result in a default value", func(t *testing.T) {
		var buf bytes.Buffer
		expLevel := slog.LevelInfo
		logger := newLogger(&buf, "invalid", "JSON")

		if !logger.Enabled(context.Background(), expLevel) {
			t.Errorf("expected output logger to be enabled at level %q", expLevel)
		}

		logger.LogAttrs(context.Background(), expLevel, "test")

		if buf.Len() < 1 {
			t.Errorf("expected some data written to buffer but got none")
		}
	})

	t.Run("invalid log format will result in a default value", func(t *testing.T) {
		var buf bytes.Buffer
		level := slog.LevelInfo
		logger := newLogger(&buf, "INFO", "invalid")

		if !logger.Enabled(context.Background(), level) {
			t.Errorf("expected output logger to be enabled at level %q", level)
		}

		logger.LogAttrs(context.Background(), level, "test")

		if buf.Len() < 1 {
			t.Errorf("expected some data written to buffer but got none")
		}
	})
}
