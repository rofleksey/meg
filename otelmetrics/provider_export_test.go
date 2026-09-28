package otelmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenHonorsExporterEnvironmentPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		specific    bool
		wantPath    string
		wantHeaders string
	}{
		{name: "generic endpoint and headers", wantPath: "/v1/metrics", wantHeaders: "generic"},
		{name: "metrics endpoint and headers override generic", specific: true, wantPath: "/custom-metrics", wantHeaders: "metrics"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type exportRequest struct {
				method string
				path   string
				scope  string
			}

			requests := make(chan exportRequest, 4)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests <- exportRequest{method: request.Method, path: request.URL.Path, scope: request.Header.Get("X-Export-Scope")}

				writer.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)
			t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
			t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "X-Export-Scope=generic")
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_HEADERS", "")
			t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
			t.Setenv("OTEL_SERVICE_NAME", "")
			t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "600000")
			t.Setenv("OTEL_METRIC_EXPORT_TIMEOUT", "2000")

			if test.specific {
				t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", server.URL+test.wantPath)
				t.Setenv("OTEL_EXPORTER_OTLP_METRICS_HEADERS", "X-Export-Scope=metrics")
			}

			provider, err := Open(t.Context(), "sample")
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
				defer cancel()

				if err := provider.Shutdown(ctx); err != nil {
					t.Errorf("shutdown: %v", err)
				}
			})

			counter, err := provider.MeterProvider().Meter("test").Int64Counter("test.requests")
			if err != nil {
				t.Fatal(err)
			}

			counter.Add(t.Context(), 1)

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			if err := provider.sdk.ForceFlush(ctx); err != nil {
				t.Fatal(err)
			}

			select {
			case request := <-requests:
				if request.method != http.MethodPost || request.path != test.wantPath || request.scope != test.wantHeaders {
					t.Fatalf("export request = %#v, want POST %s and scope %s", request, test.wantPath, test.wantHeaders)
				}
			default:
				t.Fatal("no metrics export received")
			}
		})
	}
}
