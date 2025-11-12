package cmd

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestNewLogHandler(t *testing.T) {
	tests := []struct {
		name       string
		loggingOff bool
		logLevel   string
		logFormat  string
		expOutput  bool
	}{
		{name: "logging off", loggingOff: true, expOutput: false},
		{name: "invalid log level", logLevel: "BAD", logFormat: "JSON", expOutput: true},
		{name: "invalid log format", logLevel: "INFO", logFormat: "BAD", expOutput: true},
		{name: "ok JSON", logLevel: "INFO", logFormat: "JSON", expOutput: true},
		{name: "ok JsOn", logLevel: "INFO", logFormat: "JsOn", expOutput: true},
		{name: "ok TEXT", logLevel: "INFO", logFormat: "TEXT", expOutput: true},
		{name: "ok TeXt", logLevel: "INFO", logFormat: "TeXt", expOutput: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := newLogHandler(&buf, test.loggingOff, test.logLevel, test.logFormat)
			if handler == nil {
				t.Fatal("expected non-empty handler")
			}

			slog.New(handler).Info("test")

			output := buf.String()
			t.Log(output)
			if test.expOutput && len(output) < 1 {
				t.Error("expected output but got none")
			} else if !test.expOutput && len(output) > 0 {
				t.Errorf("unexpected output: %q", output)
			}
		})
	}
}
