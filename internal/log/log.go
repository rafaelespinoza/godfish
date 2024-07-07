// Package log encapsulates structured logging using a singleton logger
// instance. By default, it will write to stderr in a text format.
package log

import (
	"context"
	"log/slog"
	"strings"
)

func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, theLogger, slog.LevelInfo, msg, attrs...)
}

func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, theLogger, slog.LevelWarn, msg, attrs...)
}

func Error(ctx context.Context, err error, msg string, attrs ...slog.Attr) {
	if err != nil {
		attrs = append([]slog.Attr{slog.String("error", err.Error())}, attrs...)
	}

	log(ctx, theLogger, slog.LevelError, msg, attrs...)
}

const prefix = "godfish:"

type slogger interface {
	Enabled(ctx context.Context, lvl slog.Level) bool
	LogAttrs(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr)
}

func log(ctx context.Context, s slogger, lvl slog.Level, msg string, attrs ...slog.Attr) {
	if !s.Enabled(ctx, lvl) {
		return
	}

	if !strings.HasPrefix(msg, prefix) {
		msg = prefix + " " + msg
	}

	s.LogAttrs(ctx, lvl, msg, slog.Attr{Key: "data", Value: slog.GroupValue(attrs...)})
}
