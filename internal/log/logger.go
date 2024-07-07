package log

import (
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
)

// theLogger is a singleton, logger package-level logger. It can be replaced
// with [SetLogger].
var theLogger = newLogger(os.Stderr, "INFO", "TEXT")

// SetLogger replaces the package-level logger. If w is empty, then all log
// entries are discarded and logging is effectively off. The logLevel and
// logFormat inputs should correspond to a value in Levels or Formats
// respectively.
func SetLogger(w io.Writer, logLevel, logFormat string) {
	theLogger = newLogger(w, logLevel, logFormat)
}

var (
	Levels  = []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	Formats = []string{"JSON", "TEXT"}
)

func newLogger(w io.Writer, logLevel, logFormat string) slogger {
	if w == nil {
		h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1})
		return slog.New(h)
	}

	levels := make([]string, len(Levels))
	for i, validLevel := range Levels {
		levels[i] = validLevel.String()
	}
	var lvl slog.Level
	if ind := slices.Index(levels, strings.ToUpper(logLevel)); ind >= 0 {
		lvl = Levels[ind]
	} else {
		slog.Warn("invalid log level, using default",
			slog.String("input_log_level", logLevel),
			slog.String("default_log_level", lvl.String()),
		)
	}

	var h slog.Handler
	switch strings.ToUpper(logFormat) {
	case "JSON":
		h = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
	case "TEXT":
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl})
	default:
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl})
		slog.Warn("invalid log format, using default",
			slog.String("input_log_format", logFormat),
			slog.String("default_log_format", "text"),
		)
	}

	return slog.New(h)
}
