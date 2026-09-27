package otelmetrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestDisabledAndInvalidExporter(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	provider, err := Open(context.Background(), "sample")
	if err != nil || provider.MeterProvider() == nil || provider.Shutdown(context.Background()) != nil {
		t.Fatalf("disabled provider = %v, error = %v", provider, err)
	}
	t.Setenv("OTEL_METRICS_EXPORTER", "invalid")
	if _, err := Open(context.Background(), "sample"); err == nil {
		t.Fatal("invalid exporter was accepted")
	}
}

func TestServiceResource(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	service, err := serviceResource("sample")
	if err != nil {
		t.Fatal(err)
	}
	value, ok := service.Set().Value(attribute.Key("service.name"))
	if !ok || value.AsString() != "sample" {
		t.Fatalf("service.name = %q, present=%v", value.AsString(), ok)
	}
}
