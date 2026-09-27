package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestLoggerRedactsNestedAndWithFields(t *testing.T) {
	var output bytes.Buffer
	logger := New("json", &output).With("credential_id", "private", "safe", "ok")
	logger.InfoContext(context.Background(), "event",
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
