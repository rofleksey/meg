package logging

import (
	"context"
	"log/slog"
	"strings"
)

const redactedValue = "[REDACTED]"

// RedactingHandler replaces attributes whose keys contain common credential names.
// It inspects group members and attributes added through WithAttrs.
type RedactingHandler struct {
	next  slog.Handler
	group string
}

func NewRedactingHandler(next slog.Handler) *RedactingHandler {
	return &RedactingHandler{next: next}
}

func (handler *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.next.Enabled(ctx, level)
}

func (handler *RedactingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		redacted.AddAttrs(redactAttr(handler.group, attr))
		return true
	})

	return handler.next.Handle(ctx, redacted)
}

func (handler *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for index := range attrs {
		redacted[index] = redactAttr(handler.group, attrs[index])
	}

	return &RedactingHandler{next: handler.next.WithAttrs(redacted), group: handler.group}
}

func (handler *RedactingHandler) WithGroup(name string) slog.Handler {
	group := name
	if handler.group != "" {
		group = handler.group + "." + name
	}

	return &RedactingHandler{next: handler.next.WithGroup(name), group: group}
}

func redactAttr(group string, attr slog.Attr) slog.Attr {
	if attr.Equal(slog.Attr{}) {
		return attr
	}

	qualified := strings.ToLower(strings.Trim(strings.Join([]string{group, attr.Key}, "."), "."))
	if sensitiveKey(qualified) {
		return slog.String(attr.Key, redactedValue)
	}
	// Resolve lazy values before inspecting groups so their nested fields cannot
	// bypass redaction when the wrapped handler resolves them later.
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		members := attr.Value.Group()

		values := make([]slog.Attr, len(members))
		for index := range members {
			values[index] = redactAttr(qualified, members[index])
		}

		attr.Value = slog.GroupValue(values...)
	}

	return attr
}

func sensitiveKey(key string) bool {
	for _, fragment := range []string{"authorization", "cookie", "password", "secret", "token", "credential"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}

	return false
}
