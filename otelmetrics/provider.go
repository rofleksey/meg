// Package otelmetrics configures optional OTLP/HTTP metrics export.
package otelmetrics

import (
	"context"
	"errors"
	"fmt"
	"math"
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

const initializationCleanupTimeout = 5 * time.Second

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
	// Validate the resource before creating an exporter that needs cleanup.
	service, err := serviceResource(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("create metrics resource: %w", err)
	}

	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP metrics exporter: %w", err)
	}

	sdk := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(exportDuration("OTEL_METRIC_EXPORT_INTERVAL", 30*time.Second)),
			sdkmetric.WithTimeout(exportDuration("OTEL_METRIC_EXPORT_TIMEOUT", 5*time.Second)),
		)),
		sdkmetric.WithResource(service),
	)
	if err := otelruntime.Start(otelruntime.WithMeterProvider(sdk), otelruntime.WithMinimumReadMemStatsInterval(15*time.Second)); err != nil {
		cleanupErr := cleanupInitialization(ctx, sdk.Shutdown)
		return nil, errors.Join(fmt.Errorf("start Go runtime metrics: %w", err), cleanupErr)
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
	if err != nil || milliseconds <= 0 || milliseconds > math.MaxInt64/int64(time.Millisecond) {
		return fallback
	}

	return time.Duration(milliseconds) * time.Millisecond
}

func serviceResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	// Build per provider instead of using resource.Default's process-wide cache.
	// The caller's service name continues to take precedence over the environment.
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
}

func cleanupInitialization(ctx context.Context, shutdown func(context.Context) error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), initializationCleanupTimeout)
	defer cancel()

	if err := shutdown(cleanupCtx); err != nil {
		return fmt.Errorf("clean up metrics initialization: %w", err)
	}

	return nil
}
