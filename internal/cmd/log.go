package cmd

import (
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/lmittmann/tint"
)

const defaultLogFmt, jsonLogFmt, textLogFmt = "COLORS", "JSON", "TEXT"

var (
	validLoggingLevels  = []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	validLoggingFormats = []string{defaultLogFmt, jsonLogFmt, textLogFmt}

	defaultLoggingLevel  = validLoggingLevels[1]
	defaultLoggingFormat = validLoggingFormats[0]
)

func newLogHandler(w io.Writer, loggingOff bool, logLevel, logFormat string) slog.Handler {
	if loggingOff {
		return slog.NewTextHandler(io.Discard, nil)
	}

	var lvl slog.Level
	levels := make([]string, len(validLoggingLevels))
	for i, validLevel := range validLoggingLevels {
		levels[i] = validLevel.String()
	}
	if ind := slices.Index(levels, strings.ToUpper(strings.TrimSpace(logLevel))); ind >= 0 {
		lvl = validLoggingLevels[ind]
	} else {
		lvl = defaultLoggingLevel
		slog.Warn("invalid logging level, using default",
			slog.String("input_log_level", logLevel),
			slog.String("default_log_level", lvl.String()),
		)
	}

	var handler slog.Handler
	opts := slog.HandlerOptions{Level: lvl}

	switch strings.ToUpper(strings.TrimSpace(logFormat)) {
	case jsonLogFmt:
		handler = slog.NewJSONHandler(w, &opts)
	case textLogFmt:
		handler = slog.NewTextHandler(w, &opts)
	case defaultLogFmt:
		handler = newTintHandler(w, lvl)
	default:
		handler = newTintHandler(w, lvl)
		slog.Warn("invalid log format, using default",
			slog.String("input_log_format", logFormat),
			slog.String("default_log_format", defaultLogFmt),
		)
	}

	return handler
}

const brightRedANSI = 9

func newTintHandler(w io.Writer, lvl slog.Level) slog.Handler {
	opts := tint.Options{
		Level:      lvl,
		TimeFormat: "2006-01-02T15:04:05.000",
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.KindAny {
				if _, ok := a.Value.Any().(error); ok {
					return tint.Attr(brightRedANSI, a)
				}
			}
			return a
		},
	}
	return tint.NewHandler(w, &opts)
}
