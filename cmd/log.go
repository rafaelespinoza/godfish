package cmd

import (
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/romantomjak/devslog"
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
		return slog.DiscardHandler
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
		handler = newDevslogHandler(w, lvl)
	default:
		handler = newDevslogHandler(w, lvl)
		slog.Warn("invalid log format, using default",
			slog.String("input_log_format", logFormat),
			slog.String("default_log_format", defaultLogFmt),
		)
	}

	return handler
}

func newDevslogHandler(w io.Writer, lvl slog.Level) slog.Handler {
	return devslog.NewHandler(w, &slog.HandlerOptions{Level: lvl})
}
