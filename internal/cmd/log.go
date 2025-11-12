package cmd

import (
	"io"
	"log/slog"
	"slices"
	"strings"
)

var (
	validLoggingLevels  = []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	validLoggingFormats = []string{"JSON", "TEXT"}

	defaultLoggingLevel  = validLoggingLevels[1]
	defaultLoggingFormat = validLoggingFormats[len(validLoggingFormats)-1]
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
	case "JSON":
		handler = slog.NewJSONHandler(w, &opts)
	case "TEXT":
		handler = slog.NewTextHandler(w, &opts)
	default:
		handler = slog.NewTextHandler(w, &opts)
		slog.Warn("invalid log format, using default",
			slog.String("input_log_format", logFormat),
			slog.String("default_log_format", "TEXT"),
		)
	}

	return handler
}
