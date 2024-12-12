package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

const (
	defaultMetricExportInterval = 1 * time.Minute
)

// StartMeter initializes a meter to collect and export OTLP metrics.
// The meter is configured with attributes related to GCP deployment,
// environment, etc.
//
// shutdown, err := telemetry.StartMeter(ctx, telemetry.WithExportInterval(10*time.Second))
// if err != nil { // handle err }
// defer shutdown()
//
// The default export destination is to an OTLP collector agent running
// on localhost:4317.
//
// For testing purposes, override the export destination to stdout.
// telemetry.StartMeter(ctx, telemetry.WithStdoutExportMeter())
func StartMeter(ctx context.Context, options ...MeterOption) (shutdown func(context.Context) error, err error) {
	defaultInitOnce.Do(func() {
		err = setupPropagator(ctx)
		if err != nil {
			return
		}
	})
	shutdown, err = startMeter(ctx, options...)
	return
}

func startMeter(ctx context.Context, options ...MeterOption) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	meterOpts, err := meterOptions(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to apply options: %w", err)
	}

	// Set up some automatic attributes regarding deployment environment, service,
	// GCP, etc to attach to traces.
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create opentelemetry resource: %w", err)
	}

	// Set up meter.
	meterProvider, err := newMeterProvider(ctx, res, meterOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to set up tracing provider: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)
	return
}

func newMeterProvider(ctx context.Context, res *resource.Resource, opts MeterOptions) (*metric.MeterProvider, error) {
	var (
		metricExporter metric.Exporter
		err            error
	)

	if opts.StdOutExport {
		metricExporter, err = stdoutmetric.New(
			stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithTemporalitySelector(deltaSelector))
		if err != nil {
			return nil, err //nolint:wrapcheck // Want passthrough
		}
	} else {
		otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if !ok {
			otelAgentAddr = defaultOtelAgentAddr
		}
		metricExporter, err = otlpmetricgrpc.New(
			ctx,
			// Insecure allows skipping for TLS cert which
			// is fine for exporting telemetry within localhost.
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithEndpoint(otelAgentAddr),
			otlpmetricgrpc.WithTemporalitySelector(deltaSelector))
		if err != nil {
			return nil, err //nolint:wrapcheck // Want passthrough
		}
	}

	exportInterval := defaultMetricExportInterval
	if opts.ExportInterval != 0 {
		exportInterval = opts.ExportInterval
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(exportInterval))),
	)
	return meterProvider, nil
}

// Borrowed from https://docs.datadoghq.com/opentelemetry/guide/otlp_delta_temporality/#configuring-your-opentelemetry-sdk
func deltaSelector(kind metric.InstrumentKind) metricdata.Temporality {
	switch kind {
	case metric.InstrumentKindCounter, metric.InstrumentKindGauge, metric.InstrumentKindHistogram, metric.InstrumentKindObservableGauge, metric.InstrumentKindObservableCounter:
		return metricdata.DeltaTemporality
	case metric.InstrumentKindUpDownCounter, metric.InstrumentKindObservableUpDownCounter:
	}
	return metricdata.CumulativeTemporality
}
