// Package logging provides a structured logger with central field redaction.
package logging

import (
	"io"
	"log/slog"
)

// New creates a JSON logger by default, or a text logger for local use.
func New(format string, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(output, options)
	} else {
		handler = slog.NewJSONHandler(output, options)
	}

	return slog.New(NewRedactingHandler(handler))
}
