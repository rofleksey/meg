package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerRedactsNestedAndWithFields(t *testing.T) {
	var output bytes.Buffer

	logger := New("json", &output).With("credential_id", "private", "safe", "ok")
	logger.InfoContext(t.Context(), "event",
		"nested", slog.GroupValue(slog.String("password", "secret"), slog.String("result", "ok")),
	)

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}

	nested, ok := record["nested"].(map[string]any)
	if !ok || record["credential_id"] != redactedValue || record["safe"] != "ok" ||
		nested["password"] != redactedValue || nested["result"] != "ok" {
		t.Fatalf("record = %#v", record)
	}
}

type lazyGroup []slog.Attr

func (value lazyGroup) LogValue() slog.Value {
	return slog.GroupValue(value...)
}

func TestLoggerRedactsLazyGroups(t *testing.T) {
	tests := []struct {
		name     string
		logger   func(*slog.Logger, slog.Attr) *slog.Logger
		group    string
		attached bool
	}{
		{
			name: "record attribute",
			logger: func(logger *slog.Logger, _ slog.Attr) *slog.Logger {
				return logger
			},
		},
		{
			name: "WithAttrs",
			logger: func(logger *slog.Logger, attr slog.Attr) *slog.Logger {
				return slog.New(logger.Handler().WithAttrs([]slog.Attr{attr}))
			},
			attached: true,
		},
		{
			name: "WithGroup",
			logger: func(logger *slog.Logger, _ slog.Attr) *slog.Logger {
				return logger.WithGroup("request")
			},
			group: "request",
		},
		{
			name: "WithGroup and WithAttrs",
			logger: func(logger *slog.Logger, attr slog.Attr) *slog.Logger {
				return logger.WithGroup("request").With(attr)
			},
			group:    "request",
			attached: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer

			attr := slog.Any("nested", lazyGroup{
				slog.String("password", "private-password"),
				slog.String("result", "ok"),
				slog.Any("inner", lazyGroup{
					slog.String("access_token", "private-token"),
					slog.Int("count", 2),
				}),
			})

			logger := test.logger(New("json", &output), attr)
			if test.attached {
				logger.InfoContext(t.Context(), "event")
			} else {
				logger.LogAttrs(t.Context(), slog.LevelInfo, "event", attr)
			}

			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatal(err)
			}

			if test.group != "" {
				group, ok := record[test.group].(map[string]any)
				if !ok {
					t.Fatalf("group missing from record: %#v", record)
				}

				record = group
			}

			nested, ok := record["nested"].(map[string]any)
			if !ok || nested["password"] != redactedValue || nested["result"] != "ok" {
				t.Fatalf("nested fields = %#v", record)
			}

			inner, ok := nested["inner"].(map[string]any)
			if !ok || inner["access_token"] != redactedValue || inner["count"] != float64(2) {
				t.Fatalf("inner fields = %#v", nested)
			}

			if strings.Contains(output.String(), "private-") {
				t.Fatalf("private value leaked: %s", output.String())
			}
		})
	}
}
