package operation

import (
	"context"
	"errors"
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
