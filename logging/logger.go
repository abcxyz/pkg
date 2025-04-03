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

// Package logging is an opinionated structured logging library based on
// [log/slog].
//
// This package also aliases most top-level functions in [log/slog] to reduce
// the need to manage the additional import.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/abcxyz/pkg/timeutil"
)

// contextKey is a private string type to prevent collisions in the context map.
type contextKey string

// loggerKey points to the value in the context where the logger is stored.
const loggerKey = contextKey("logger")

// defaultLogger returns a function that returns the default logger. which
// writes JSON output to stdout at the "Info" level.
//
// It is initialized once when called the first time.
var defaultLoggerOnce = sync.OnceValue[*slog.Logger](func() *slog.Logger {
	return New(os.Stdout, LevelInfo, FormatJSON, false)
})

// New creates a new logger in the specified format and writes to the provided
// writer at the provided level. Use the returned leveler to dynamically change
// the level to a different value after creation.
//
// If debug is true, the logging level is set to the lowest possible value
// (meaning all messages will be printed), and the output will include source
// information. This is very expensive, and you should not enable it unless
// actively debugging.
//
// It returns the configured logger and a leveler which can be used to change
// the logger's level dynamically. The leveler does not require locking to
// change the level.
func New(w io.Writer, level slog.Level, format Format, debug bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		ReplaceAttr: cloudLoggingAttrsEncoder(),
	}

	// Enable the most detailed log level and add source information in debug
	// mode.
	if debug {
		opts.AddSource = true
		level = math.MinInt
	}

	switch format {
	case FormatJSON:
		return slog.New(NewLevelHandler(level, slog.NewJSONHandler(w, opts)))
	case FormatText:
		return slog.New(NewLevelHandler(level, slog.NewTextHandler(w, opts)))
	default:
		panic(fmt.Sprintf("unknown log format %q", format))
	}
}

// NewFromEnv is a convenience function for creating a logger that is configured
// from the environment. It sources the following environment variables, first
// checking any with the prefix, then falling back to the global unprefixed
// value:
//
//   - LOG_LEVEL: string representation of the log level. It panics if no such log level exists.
//   - LOG_FORMAT: format in which to output logs (e.g. json, text). It panics if no such format exists.
//   - LOG_DEBUG: enable the most detailed debug logging. It panics iff the given value is not a valid boolean.
func NewFromEnv(envPrefix string) *slog.Logger {
	return newFromEnv(envPrefix, os.Getenv)
}

// newFromEnv is a helper that makes it easier to test [NewFromEnv].
func newFromEnv(envPrefix string, getenvFunc func(string) string) *slog.Logger {
	levelEnvVarKey, levelEnvVarValue := multiGetenv(getenvFunc, envPrefix+"LOG_LEVEL", "LOG_LEVEL")
	level, err := LookupLevel(levelEnvVarValue)
	if err != nil {
		panic(fmt.Sprintf("log level: invalid value for %s: %s", levelEnvVarKey, err))
	}

	formatEnvVarKey, formatEnvVarValue := multiGetenv(getenvFunc, envPrefix+"LOG_FORMAT", "LOG_FORMAT")
	format, err := LookupFormat(formatEnvVarValue)
	if err != nil {
		panic(fmt.Sprintf("log format: invalid value for %s: %s", formatEnvVarKey, err))
	}

	debugEnvVarKey, debugEnvVarValue := multiGetenv(getenvFunc, envPrefix+"LOG_DEBUG", "LOG_DEBUG")
	debug, err := strconv.ParseBool(debugEnvVarValue)
	if err != nil {
		if debugEnvVarValue != "" {
			panic(fmt.Sprintf("log debug: invalid value for %s: %s", debugEnvVarKey, err))
		}
	}

	targetEnvVarKey, targetEnvVarValue := multiGetenv(getenvFunc, envPrefix+"LOG_TARGET", "LOG_TARGET")
	target, err := LookupTarget(targetEnvVarValue)
	if err != nil {
		panic(fmt.Sprintf("log target: invalid value for %s: %s", targetEnvVarKey, err))
	}

	return New(target, level, format, debug)
}

// multiGetenv is a helper function for looking up a collection of environment
// variables.
func multiGetenv(f func(string) string, ss ...string) (string, string) {
	if len(ss) == 0 {
		return "", ""
	}

	for _, s := range ss {
		if v := strings.TrimSpace(f(s)); v != "" {
			return s, v
		}
	}
	return ss[0], ""
}

// SetLevel adjusts the level on the provided logger. The handler on the given
// logger must be a [LevelableHandler] or else this function panics. If you
// created a logger through this package, it will automatically satisfy that
// interface.
//
// This function is safe for concurrent use.
//
// It returns the provided logger for convenience and easier chaining.
func SetLevel(logger *slog.Logger, level slog.Level) *slog.Logger {
	if typ, ok := logger.Handler().(LevelableHandler); ok {
		typ.SetLevel(level)
		return logger
	}

	panic("handler is not capable of setting levels")
}

// DefaultLogger creates a default logger.
func DefaultLogger() *slog.Logger {
	return defaultLoggerOnce()
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
	return DefaultLogger()
}

// cloudLoggingAttrsEncoder updates the [slog.Record] attributes to match the
// key names and [format for Google Cloud Logging].
//
// [format for Google Cloud Logging]: https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
func cloudLoggingAttrsEncoder() func([]string, slog.Attr) slog.Attr {
	const (
		keySeverity = "severity"
		keyError    = "error"
		keyMessage  = "message"
		keySource   = "logging.googleapis.com/sourceLocation"
	)

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

		// Google Cloud Logging uses "logging.google..." instead of "source":
		// https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
		if a.Key == slog.SourceKey {
			a.Key = keySource
		}

		// Re-format durations to be their string format.
		if a.Value.Kind() == slog.KindDuration {
			val := a.Value.Duration()
			a.Value = slog.StringValue(timeutil.HumanDuration(val))
		}

		return a
	}
}
