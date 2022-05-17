// Copyright 2022 abcxyz authors (see AUTHORS file)
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
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
)

// contextKey is a private string type to prevent collisions in the context map.
type contextKey string

const (
	// loggerKey points to the value in the context where the logger is stored.
	loggerKey = contextKey("logger")

	// googleCloudTraceHeader is the header with trace data.
	googleCloudTraceHeader = "X-Cloud-Trace-Context"

	// googleCloudTraceKey is the key in the structured log where trace information
	// is expected to be present.
	googleCloudTraceKey = "logging.googleapis.com/trace"
)

var (
	// defaultLogger is the default logger. It is initialized once per package
	// include upon calling DefaultLogger.
	defaultLogger     *zap.SugaredLogger
	defaultLoggerOnce sync.Once
)

// NewFromEnv creates a new logger from env vars.
// Set envPrefix+"LOG_LEVEL" to overwrite log level. Default log level is warning.
// Set envPrefix+"LOG_MODE" to overwrite log mode. Default log mode is production.
func NewFromEnv(envPrefix string) *zap.SugaredLogger {
	level := os.Getenv(envPrefix + "LOG_LEVEL")
	devMode := strings.ToLower(strings.TrimSpace(os.Getenv(envPrefix+"LOG_MODE"))) == "development"
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

// Default creates a default logger.
// To overwrite log level and mode, set AUDIT_CLIENT_LOG_LEVEL and AUDIT_CLIENT_LOG_MODE.
// This is mostly used for audit client. The caller doesn't have to explicitly set up a logger.
func Default() *zap.SugaredLogger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewFromEnv("AUDIT_CLIENT_")
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

// GRPCInterceptor returns a gRPC interceptor that populates a logger in the context.
func GRPCInterceptor(logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var th string
		md, ok := grpcmetadata.FromIncomingContext(ctx)
		if ok {
			if len(md.Get(googleCloudTraceHeader)) > 0 {
				th = md.Get(googleCloudTraceHeader)[0]
			}
		}

		// Best effort to get the current project ID.
		projectID, _ := metadata.ProjectID()

		// This allows us to associate logs to traces.
		// See "logging.googleapis.com/trace" at:
		// https://cloud.google.com/logging/docs/agent/logging/configuration#special-fields
		if th != "" && projectID != "" {
			// The trace header is in the format:
			// "X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=TRACE_TRUE"
			// See: https://cloud.google.com/trace/docs/setup#force-trace
			parts := strings.Split(th, "/")
			if len(parts) > 0 && len(parts[0]) > 0 {
				val := fmt.Sprintf("projects/%s/traces/%s", projectID, parts[0])
				logger = logger.With(googleCloudTraceKey, val)
			}
		}

		ctx = WithLogger(ctx, logger)
		return handler(ctx, req)
	}
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

// timeEncoder encodes the time as RFC3339 nano
func timeEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(time.RFC3339Nano))
	}
}
