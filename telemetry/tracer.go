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
	"time"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// StartTracer initializes a tracer to collect and export OTLP traces.
// The tracer is configured with attributes related to GCP deployment,
// environment, etc.
//
// shutdown, err := telemetry.StartTracer(ctx)
// if err != nil { // handle err }
// defer shutdown()
//
// The default export destination is to an OTLP collector agent running
// on localhost:4317.
//
// For testing purposes, override the export destination to stdout.
// telemetry.StartTracer(ctx, telemetry.WithStdoutExportTraces())
func StartTracer(ctx context.Context, options ...TracerOption) (shutdown func(context.Context) error, err error) {
	defaultInitOnce.Do(func() {
		err = setupPropagator(ctx)
		if err != nil {
			return
		}
	})
	shutdown, err = startTracer(ctx, options...)
	return
}

func startTracer(ctx context.Context, options ...TracerOption) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	tracerOpts, err := tracerOptions(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to apply options: %w", err)
	}

	// Set up some automatic attributes regarding deployment environment,
	// process, GCP service, etc. to attach to traces.
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

	// Set up tracer.
	var tracerProvider *trace.TracerProvider
	if tracerOpts.StdOutExport {
		tracerProvider, err = newStdoutTraceProvider(ctx, res)
	} else {
		tracerProvider, err = newOTLPTraceProvider(ctx, res)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to set up tracing provider: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)
	return
}

func newOTLPTraceProvider(ctx context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = defaultOtelAgentAddr
	}
	traceClient := otlptracegrpc.NewClient(
		// Insecure allows skipping for TLS cert which
		// is fine for exporting telemetry within localhost.
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
