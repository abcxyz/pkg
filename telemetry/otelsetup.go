package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var defaultInitOnce sync.Once

const (
	defaultOtelAgentAddr = "0.0.0.0:4317"
)

func setupPropagator(_ context.Context) error {
	// Set up propagator to W3C trace context, a common specification for
	// distributed tracing. See https://opentelemetry.io/docs/concepts/context-propagation.
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
	return nil
}
