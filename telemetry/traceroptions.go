package telemetry

import "context"

type TracerOptions struct {
	StdOutExport bool
}

type TracerOption interface {
	SetTracer(opts *TracerOptions) error
}

type tracerOption func(*TracerOptions) error

func (topt tracerOption) SetTracer(opts *TracerOptions) error {
	return topt(opts)
}

func newTracerOption(fn func(opts *TracerOptions) error) TracerOption {
	return tracerOption(fn)
}

// WithStdoutExportTraces provides an option to export traces to stdout.
func WithStdoutExportTraces() TracerOption {
	return newTracerOption(func(opts *TracerOptions) error {
		opts.StdOutExport = true
		return nil
	})
}

func tracerOptions(_ context.Context, opts []TracerOption) (TracerOptions, error) {
	tracerOpts := TracerOptions{}
	for _, opt := range opts {
		if err := opt.SetTracer(&tracerOpts); err != nil {
			return TracerOptions{}, err
		}
	}
	return tracerOpts, nil
}
