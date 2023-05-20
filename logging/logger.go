// Copyright 2022 The Authors (see AUTHORS file)
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

// Package logging sets up and configures standard logging.
package logging

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/abcxyz/pkg/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// contextKey is a private string type to prevent collisions in the context map.
type contextKey string

const (
	// loggerKey points to the value in the context where the logger is stored.
	loggerKey = contextKey("logger")
)

var (
	// defaultLogger is the default logger. It is initialized once per package
	// include upon calling DefaultLogger.
	defaultLogger     *zap.SugaredLogger
	defaultLoggerOnce sync.Once

	flagLogLevel string
	flagLogMode  string
)

// RegisterFlags register common logging flags to the given
// [github.com/abcxyz/pkg/cli.FlagSet].
func RegisterFlags(fs *cli.FlagSet) {
	f := fs.NewSection("LOG OPTIONS")
	f.StringVar(&cli.StringVar{
		Name:    "log-level",
		Example: "info",
		Default: "warning",
		EnvVar:  "LOG_LEVEL",
		Target:  &flagLogLevel,
		Usage:   "Log verbosity, one of debug|info|warning|error.",
	})
	f.StringVar(&cli.StringVar{
		Name:    "log-mode",
		Example: "production",
		Default: "production",
		EnvVar:  "LOG_MODE",
		Target:  &flagLogMode,
		Usage:   "Log mode, one of dev|production.",
	})
}

// NewFromEnv creates a new logger from env vars.
// Set envPrefix+"LOG_LEVEL" to overwrite log level. Default log level is warning.
// Set envPrefix+"LOG_MODE" to overwrite log mode. Default log mode is production.
func NewFromEnv(envPrefix string) *zap.SugaredLogger {
	level := os.Getenv(envPrefix + "LOG_LEVEL")
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(envPrefix + "LOG_MODE")))
	return newLogger(level, mode)
}

// NewFromFlags creates a new logger from flags registered via [RegisterFlags].
// If the flags were not registered, the log level will default to warning and
// the log mode will default to production.
func NewFromFlags() *zap.SugaredLogger {
	return newLogger(flagLogLevel, flagLogMode)
}

func newLogger(level, mode string) *zap.SugaredLogger {
	devMode := strings.HasPrefix(mode, "dev")
	var cfg zap.Config
	if devMode {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig = developmentEncoderConfig
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig = productionEncoderConfig
	}

	var l zapcore.Level
	if err := l.Set(level); err != nil {
		// Invalid level? Default to warn.
		l = zapcore.WarnLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(l)

	logger, err := cfg.Build()
	if err != nil {
		logger = zap.NewNop()
	}

	return logger.Sugar()
}

// Default creates a default logger. To overwrite log level and mode, set
// LOG_LEVEL and LOG_MODE.
func Default() *zap.SugaredLogger {
	defaultLoggerOnce.Do(func() {
		if flagLogLevel == "" && flagLogMode == "" {
			// If the flags were not initialized, fall back env vars to create the
			// logger.
			defaultLogger = NewFromEnv("")
		} else {
			defaultLogger = NewFromFlags()
		}
	})
	return defaultLogger
}

// WithLogger creates a new context with the provided logger attached.
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger stored in the context. If no such logger
// exists, a default logger is returned.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return Default()
}

// TestLogger returns a logger configured for tests. It will only output log
// information if specific test fails or is run in verbose mode. See [zaptest]
// for more information.
//
//	func TestMyThing(t *testing.T) {
//		logger := logging.TestLogger(t)
//		thing := &MyThing{logger: logger}
//	}
//
// [zaptest]: https://pkg.go.dev/go.uber.org/zap/zaptest
func TestLogger(tb zaptest.TestingT, opts ...zaptest.LoggerOption) *zap.SugaredLogger {
	warnLevelOpt := zaptest.Level(zap.WarnLevel)
	opts = append([]zaptest.LoggerOption{warnLevelOpt}, opts...)
	return zaptest.NewLogger(tb, opts...).Sugar()
}

const (
	timestamp  = "timestamp"
	severity   = "severity"
	logger     = "logger"
	caller     = "caller"
	message    = "message"
	stacktrace = "stacktrace"

	levelDebug     = "DEBUG"
	levelInfo      = "INFO"
	levelWarning   = "WARNING"
	levelError     = "ERROR"
	levelCritical  = "CRITICAL"
	levelAlert     = "ALERT"
	levelEmergency = "EMERGENCY"
)

var productionEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        timestamp,
	LevelKey:       severity,
	NameKey:        logger,
	CallerKey:      caller,
	MessageKey:     message,
	StacktraceKey:  stacktrace,
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    levelEncoder(),
	EncodeTime:     timeEncoder(),
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

var developmentEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "",
	LevelKey:       "L",
	NameKey:        "N",
	CallerKey:      "C",
	FunctionKey:    zapcore.OmitKey,
	MessageKey:     "M",
	StacktraceKey:  "S",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.CapitalLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

// levelEncoder transforms a zap level to the associated stackdriver level.
func levelEncoder() zapcore.LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch l {
		case zapcore.InvalidLevel:
			enc.AppendString(levelAlert)
		case zapcore.DebugLevel:
			enc.AppendString(levelDebug)
		case zapcore.InfoLevel:
			enc.AppendString(levelInfo)
		case zapcore.WarnLevel:
			enc.AppendString(levelWarning)
		case zapcore.ErrorLevel:
			enc.AppendString(levelError)
		case zapcore.DPanicLevel:
			enc.AppendString(levelCritical)
		case zapcore.PanicLevel:
			enc.AppendString(levelAlert)
		case zapcore.FatalLevel:
			enc.AppendString(levelEmergency)
		}
	}
}

// timeEncoder encodes the time as RFC3339 nano.
func timeEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(time.RFC3339Nano))
	}
}
