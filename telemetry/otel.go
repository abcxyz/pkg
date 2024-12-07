// Copyright 2024 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package telemetry initializes the OpenTelemetry SDK for library users. It
// primarily serves as a convenience package for setting up OpenTelemetry SDK
// for GCP applications with some useful settings.
//
// Once initialized, users get automatic instrumentation of metrics and traces
// and can start writing custom metrics and configuring custom traces.
//
// To initialize ... somewhere in your service setup code ...
//
//	shutdown, err := telemetry.Start(ctx)
//	defer shutdown()
//
// Metrics and traces will then be exported to an OpenTelemetry collector endpoint
// with default address set to localhost:4317.
//
// For testing before setting up and exporting to a collector, use
// telemetry.StartWithStdoutExport instead.
//
// Much of this code is inspired from https://opentelemetry.io/docs/languages/go/getting-started/#initialize-the-opentelemetry-sdk
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

var defaultStartOnce sync.Once

const (
	defaultOtelAgentAddr = "0.0.0.0:4317"
)

type (
	meterProviderFunc  func(context.Context, *resource.Resource) (*metric.MeterProvider, error)
	tracerProviderFunc func(context.Context, *resource.Resource) (*trace.TracerProvider, error)
)

// Start initializes automatic instrumentation for exporting OTLP metrics and traces
// to an OpenTelemetry collector agent.
func Start(ctx context.Context) (shutdown func(context.Context) error, err error) {
	defaultStartOnce.Do(func() {
		shutdown, err = start(ctx, newOTLPTraceProvider, newOTLPMeterProvider)
	})
	return
}

// StartWithStdoutExport initializes automatic instrumentation for exporting OTLP
// metrics and traces to stdout (e.g. Cloud Logging if using Cloud Run).
//
// This is intended for testing and debugging metrics and traces before exporting
// them to an OpenTelemetry collector agent.
func StartWithStdoutExport(ctx context.Context) (shutdown func(context.Context) error, err error) {
	defaultStartOnce.Do(func() {
		shutdown, err = start(ctx, newStdoutTraceProvider, newStdoutMeterProvider)
	})
	return
}

func start(ctx context.Context, newTracer tracerProviderFunc, newMeter meterProviderFunc) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	if err := runtime.Start(); err != nil {
		return nil, fmt.Errorf("failed to start runtime instrumentation: %w", err)
	}
	// Set up some automatic attributes regarding deployment environment, service,
	// GCP, etc to attach to telemetry.
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

	// Set up propagator to W3C trace context, a common specification for
	// distributed tracing. See https://opentelemetry.io/docs/concepts/context-propagation.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up tracing.
	tracerProvider, err := newTracer(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("failed to set up tracing provider: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up monitoring.
	meterProvider, err := newMeter(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("failed to set up meter provider: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newOTLPTraceProvider(ctx context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = defaultOtelAgentAddr
	}
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr))
	traceExporter, err := otlptrace.New(ctx, traceClient)
	if err != nil {
		return nil, err //nolint:wrapcheck // Want passthrough
	}

	bsp := trace.NewBatchSpanProcessor(traceExporter)
	traceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
	return traceProvider, nil
}

func newStdoutTraceProvider(_ context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err //nolint:wrapcheck // Want passthrough
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(traceExporter,
			trace.WithBatchTimeout(5*time.Second)),
	)
	return traceProvider, nil
}

func newOTLPMeterProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, error) {
	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = defaultOtelAgentAddr
	}
	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(otelAgentAddr),
		otlpmetricgrpc.WithTemporalitySelector(deltaSelector))
	if err != nil {
		return nil, err //nolint:wrapcheck // Want passthrough
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(1*time.Minute))),
	)
	return meterProvider, nil
}

func newStdoutMeterProvider(_ context.Context, res *resource.Resource) (*metric.MeterProvider, error) {
	metricExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithTemporalitySelector(deltaSelector))
	if err != nil {
		return nil, err //nolint:wrapcheck // Want passthrough
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(1*time.Minute))),
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
