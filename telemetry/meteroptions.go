package telemetry

import (
	"context"
	"time"
)

type MeterOptions struct {
	StdOutExport   bool
	ExportInterval time.Duration
}

type MeterOption interface {
	SetMeter(opts *MeterOptions) error
}

type meterOption func(*MeterOptions) error

func (mopt meterOption) SetMeter(opts *MeterOptions) error {
	return mopt(opts)
}

func newMeterOption(fn func(opts *MeterOptions) error) MeterOption {
	return meterOption(fn)
}

// WithStdoutExportTraces provides an option to export traces to stdout.
func WithStdoutExportMetrics() MeterOption {
	return newMeterOption(func(opts *MeterOptions) error {
		opts.StdOutExport = true
		return nil
	})
}

// WithExportInterval provides an option to define the export interval
// duration. The default is 1 minute.
//
// Setting the interval too low may result in throttling if exporting to a
// collector. It's a good idea to check how much throughput it is configured
// to handle before doing so. See https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/memorylimiterprocessor/README.md
func WithExportInterval(duration time.Duration) MeterOption {
	return newMeterOption(func(opts *MeterOptions) error {
		opts.ExportInterval = duration
		return nil
	})
}

func meterOptions(_ context.Context, opts []MeterOption) (MeterOptions, error) {
	meterOpts := MeterOptions{}
	for _, opt := range opts {
		if err := opt.SetMeter(&meterOpts); err != nil {
			return MeterOptions{}, err
		}
	}
	return meterOpts, nil
}
