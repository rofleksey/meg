// Package otelmetrics configures optional OTLP/HTTP metrics export.
package otelmetrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Provider struct {
	meterProvider metric.MeterProvider
	sdk           *sdkmetric.MeterProvider
}

func (provider *Provider) MeterProvider() metric.MeterProvider {
	if provider == nil {
		return noop.NewMeterProvider()
	}
	return provider.meterProvider
}

// Open reads the standard OTEL export settings. A missing endpoint disables
// export unless OTEL_METRICS_EXPORTER explicitly requests OTLP.
func Open(ctx context.Context, serviceName string) (*Provider, error) {
	enabled, err := metricsRequested()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return Disabled(), nil
	}
	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP metrics exporter: %w", err)
	}
	service, err := serviceResource(serviceName)
	if err != nil {
		return nil, fmt.Errorf("create metrics resource: %w", err)
	}
	sdk := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(exportDuration("OTEL_METRIC_EXPORT_INTERVAL", 30*time.Second)),
			sdkmetric.WithTimeout(exportDuration("OTEL_METRIC_EXPORT_TIMEOUT", 5*time.Second)),
		)),
		sdkmetric.WithResource(service),
	)
	if err := otelruntime.Start(otelruntime.WithMeterProvider(sdk), otelruntime.WithMinimumReadMemStatsInterval(15*time.Second)); err != nil {
		_ = sdk.Shutdown(context.Background())
		return nil, fmt.Errorf("start Go runtime metrics: %w", err)
	}
	return &Provider{meterProvider: sdk, sdk: sdk}, nil
}

func Disabled() *Provider {
	return &Provider{meterProvider: noop.NewMeterProvider()}
}

func (provider *Provider) Shutdown(ctx context.Context) error {
	if provider == nil || provider.sdk == nil {
		return nil
	}
	return provider.sdk.Shutdown(ctx)
}

func metricsRequested() (bool, error) {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_METRICS_EXPORTER")))
	switch value {
	case "", "none":
		return value == "" && (strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")) != "" ||
			strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) != ""), nil
	case "otlp":
		return true, nil
	default:
		return false, errors.New("OTEL_METRICS_EXPORTER must be otlp or none")
	}
}

func exportDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	milliseconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || milliseconds <= 0 {
		return fallback
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func serviceResource(serviceName string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName)))
}
