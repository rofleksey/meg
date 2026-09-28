package operation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/happytoolin/unolog"
)

func TestCompleteRecordsSafeOutcomeAndErrorKind(t *testing.T) {
	sink := unolog.NewTestSink()
	runtime := unolog.MustCompile(unolog.Config{Sink: sink, SamplingRate: 1})
	op := Start(context.Background(), runtime, unolog.DomainJob, "job.run", "job-id", "count", 3)
	Complete(op, "failed", errors.New("secret details"))

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}

	event := events[0]
	if name, _ := event.Lookup(FieldEvent); name != "job.run" {
		t.Fatalf("name = %v", name)
	}

	if outcome, _ := event.Lookup(unolog.KeyOpOutcome); outcome != string(unolog.OutcomeFailure) {
		t.Fatalf("outcome = %v", outcome)
	}

	if kind, _ := event.Lookup(FieldErrorKind); kind != "*errors.errorString" {
		t.Fatalf("error kind = %v", kind)
	}
}

func TestCompleteKeepsCancellationSafeAndEmitsOnce(t *testing.T) {
	sink := unolog.NewTestSink()
	runtime := unolog.MustCompile(unolog.Config{Sink: sink, SamplingRate: 1})
	op := Start(t.Context(), runtime, unolog.DomainJob, "job.run", "job-id")
	err := fmt.Errorf("sensitive dependency details: %w", context.Canceled)
	Complete(op, "canceled", err)
	Complete(op, "failed", err)

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("completion events = %d, want 1", len(events))
	}

	event := events[0]
	if outcome, _ := event.Lookup(unolog.KeyOpOutcome); outcome != string(unolog.OutcomeCanceled) {
		t.Fatalf("outcome = %v, want canceled", outcome)
	}

	if kind, _ := event.Lookup(FieldErrorKind); kind != "canceled" {
		t.Fatalf("error kind = %v, want canceled", kind)
	}

	if strings.Contains(fmt.Sprint(event), "sensitive dependency details") {
		t.Fatal("completion event contains the original dependency error message")
	}
}
