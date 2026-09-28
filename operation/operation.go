// Package operation bridges wide-event lifecycles into log/slog.
package operation

import (
	"context"
	"errors"
	"log/slog"
	"reflect"

	"github.com/happytoolin/unolog"
	unologslog "github.com/happytoolin/unolog/adapter/slog"
)

const (
	FieldEvent     = "event.name"
	FieldOutcome   = "outcome"
	FieldErrorKind = "error.kind"
)

type safeEventError struct{ kind string }

func (err safeEventError) Error() string { return err.kind }

// NewRuntime sends complete operations to logger. Callers supply a logger
// with any required redaction policy.
func NewRuntime(logger *slog.Logger) *unolog.Runtime {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return unolog.MustCompile(unolog.Config{
		Sink:         unologslog.New(logger),
		SamplingRate: 1,
	})
}

func Start(ctx context.Context, runtime *unolog.Runtime, domain unolog.Domain, name, id string, fields ...any) *unolog.Operation {
	op := unolog.Start(ctx, runtime, unolog.OperationStart{Domain: domain, Name: name, ID: id})
	Enrich(op.Context(), FieldEvent, name)
	Enrich(op.Context(), fields...)

	return op
}

// Enrich adds key/value pairs to the operation in ctx. Incomplete or invalid
// pairs are skipped, matching unolog's context-bound no-op behavior.
func Enrich(ctx context.Context, fields ...any) {
	for index := 0; index+1 < len(fields); index += 2 {
		if key, ok := fields[index].(string); ok && key != "" {
			unolog.Add(ctx, key, fields[index+1])
		}
	}
}

func RecordError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	kind := ErrorKind(err)
	Enrich(ctx, FieldErrorKind, kind)
	unolog.Error(ctx, safeEventError{kind: kind})
}

// ErrorKind returns a bounded description without disclosing an error message.
func ErrorKind(err error) string {
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}

	if typ := reflect.TypeOf(err); typ != nil {
		return typ.String()
	}

	return "unknown"
}

// Complete emits exactly one final event. success and canceled are mapped to
// unolog's canonical outcomes; other outcomes are failures.
func Complete(op *unolog.Operation, outcome string, err error) {
	if op == nil {
		return
	}

	ctx := op.Context()
	Enrich(ctx, FieldOutcome, outcome)

	canonical := unolog.OutcomeFailure

	switch outcome {
	case "succeeded":
		canonical = unolog.OutcomeSuccess
	case "canceled":
		canonical = unolog.OutcomeCanceled
	}

	unolog.Add(ctx, unolog.KeyOpOutcome, string(canonical))
	RecordError(ctx, err)

	_ = op.End(nil)
}
