package otelmetrics

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func TestDisabledAndInvalidExporter(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "none")

	provider, err := Open(t.Context(), "sample")
	if err != nil || provider.MeterProvider() == nil || provider.Shutdown(t.Context()) != nil {
		t.Fatalf("disabled provider = %v, error = %v", provider, err)
	}

	t.Setenv("OTEL_METRICS_EXPORTER", "invalid")

	if _, err := Open(t.Context(), "sample"); err == nil {
		t.Fatal("invalid exporter was accepted")
	}
}

func TestServiceResource(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("OTEL_SERVICE_NAME", "")

	service, err := serviceResource(t.Context(), "sample")
	if err != nil {
		t.Fatal(err)
	}

	value, ok := service.Set().Value(attribute.Key("service.name"))
	if !ok || value.AsString() != "sample" {
		t.Fatalf("service.name = %q, present=%v", value.AsString(), ok)
	}
}

func TestMetricsRequested(t *testing.T) {
	tests := []struct {
		name            string
		exporter        string
		endpoint        string
		metricsEndpoint string
		requested       bool
		invalid         bool
	}{
		{name: "unset is disabled"},
		{name: "blank endpoints are disabled", endpoint: " ", metricsEndpoint: "\t"},
		{name: "generic endpoint enables", endpoint: "http://localhost:4318", requested: true},
		{name: "metrics endpoint enables", metricsEndpoint: "http://localhost:4318/v1/metrics", requested: true},
		{name: "none overrides endpoints", exporter: "none", endpoint: "http://localhost:4318", metricsEndpoint: "http://localhost:4318/v1/metrics"},
		{name: "explicit otlp uses default endpoint", exporter: "otlp", requested: true},
		{name: "normalized exporter", exporter: " OTLP ", requested: true},
		{name: "unknown exporter", exporter: "prometheus", invalid: true},
		{name: "multiple exporters unsupported", exporter: "otlp,none", invalid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXPORTER", test.exporter)
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", test.endpoint)
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", test.metricsEndpoint)

			requested, err := metricsRequested()
			if requested != test.requested || (err != nil) != test.invalid {
				t.Fatalf("metricsRequested() = %t, %v; want %t, invalid=%t", requested, err, test.requested, test.invalid)
			}
		})
	}
}

func TestExportDuration(t *testing.T) {
	const (
		fallback            = 5 * time.Second
		maximumMilliseconds = math.MaxInt64 / int64(time.Millisecond)
	)

	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "empty", want: fallback},
		{name: "milliseconds", raw: "1500", want: 1500 * time.Millisecond},
		{name: "whitespace", raw: " 12 ", want: 12 * time.Millisecond},
		{name: "zero", raw: "0", want: fallback},
		{name: "negative", raw: "-1", want: fallback},
		{name: "units rejected", raw: "10s", want: fallback},
		{name: "fraction rejected", raw: "1.5", want: fallback},
		{name: "maximum", raw: strconv.FormatInt(maximumMilliseconds, 10), want: time.Duration(maximumMilliseconds) * time.Millisecond},
		{name: "duration overflow", raw: strconv.FormatInt(maximumMilliseconds+1, 10), want: fallback},
		{name: "integer overflow", raw: "9223372036854775808", want: fallback},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", test.raw)

			if got := exportDuration("OTEL_METRIC_EXPORT_INTERVAL", fallback); got != test.want {
				t.Fatalf("exportDuration(%q) = %v, want %v", test.raw, got, test.want)
			}
		})
	}
}

func TestServiceResourceReadsEnvironmentPerProvider(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "environment-service")

	for _, instance := range []string{"first", "second"} {
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=attribute-service,service.instance.id="+instance)

		service, err := serviceResource(t.Context(), "explicit-service")
		if err != nil {
			t.Fatal(err)
		}

		for key, want := range map[string]string{
			"service.name":        "explicit-service",
			"service.instance.id": instance,
			"telemetry.sdk.name":  "opentelemetry",
		} {
			value, ok := service.Set().Value(attribute.Key(key))
			if !ok || value.AsString() != want {
				t.Errorf("%s = %q, present=%t; want %q", key, value.AsString(), ok, want)
			}
		}
	}
}

func TestOpenRejectsMalformedResource(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "invalid-pair")
	t.Setenv("OTEL_SERVICE_NAME", "")

	if provider, err := Open(t.Context(), "sample"); err == nil || provider != nil {
		t.Fatalf("Open() = %v, %v; want initialization error", provider, err)
	}
}

func TestInitializationCleanupIsBoundedAndIndependentOfCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	wantErr := errors.New("cleanup failure")

	err := cleanupInitialization(ctx, func(cleanupCtx context.Context) error {
		if err := cleanupCtx.Err(); err != nil {
			t.Fatalf("cleanup context inherited startup cancellation: %v", err)
		}

		deadline, ok := cleanupCtx.Deadline()
		if remaining := time.Until(deadline); !ok || remaining <= 0 || remaining > initializationCleanupTimeout {
			t.Fatalf("cleanup deadline = %v, present=%t", deadline, ok)
		}

		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("cleanup error = %v, want %v", err, wantErr)
	}
}
