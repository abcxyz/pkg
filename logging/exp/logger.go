// Copyright 2023 The Authors (see AUTHORS file)
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

// Package exp is an experimental version of logging using slog.
//
// This package also aliases most top-level functions in [golang.org/x/exp/slog]
// to reduce the need to manage the additional import.
package exp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/abcxyz/pkg/timeutil"
	"golang.org/x/exp/slog"
)

// contextKey is a private string type to prevent collisions in the context map.
type contextKey string

// loggerKey points to the value in the context where the logger is stored.
const loggerKey = contextKey("logger")

var (
	// defaultLogger is the default logger. It is initialized once per package
	// include upon calling DefaultLogger.
	defaultLogger     *slog.Logger
	defaultLoggerOnce sync.Once
)

// NewFromEnv creates a new logger from env vars.
// Set envPrefix+"LOG_LEVEL" to overwrite log level. Default log level is warning.
// Set envPrefix+"LOG_MODE" to overwrite log mode. Default log mode is production.
func NewFromEnv(envPrefix string) *slog.Logger {
	logMode := strings.ToLower(strings.TrimSpace(os.Getenv(envPrefix + "LOG_MODE")))
	devMode := strings.HasPrefix(logMode, "dev")

	level, err := LookupLevel(os.Getenv(envPrefix + "LOG_LEVEL"))
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).
			Warn("invalid log level, defaulting to info", "error", err)
		level = LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource:   devMode,
		Level:       level,
		ReplaceAttr: cloudLoggingAttrsEncoder(),
	}

	var handler slog.Handler
	if devMode {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// Default creates a default logger.
func Default() *slog.Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewFromEnv("")
	})
	return defaultLogger
}

// WithLogger creates a new context with the provided logger attached.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger stored in the context. If no such logger
// exists, a default logger is returned.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return Default()
}

const (
	keySeverity = "severity"
	keyError    = "error"
	keyMessage  = "message"
)

// cloudLoggingAttrsEncoder updates the [slog.Record] attributes to match the
// key names and [format for Google Cloud Logging].
//
// [format for Google Cloud Logging]: https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
func cloudLoggingAttrsEncoder() func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		// Google Cloud Logging uses "severity" instead of "level":
		// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
		if a.Key == slog.LevelKey {
			a.Key = keySeverity

			// Use the custom level names to match Google Cloud logging.
			val := a.Value.Any()
			typ, ok := val.(slog.Level)
			if !ok {
				panic(fmt.Sprintf("level is not slog.Level (got %T)", val))
			}
			a.Value = LevelSlogValue(typ)
		}

		// Google Cloud Logging uses "message" instead of "msg":
		// https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
		if a.Key == slog.MessageKey {
			a.Key = keyMessage
		}

		// Re-format durations to be their string format.
		if a.Value.Kind() == slog.KindDuration {
			val := a.Value.Duration()
			a.Value = slog.StringValue(timeutil.HumanDuration(val))
		}

		return a
	}
}
