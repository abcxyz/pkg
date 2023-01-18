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

package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"

	"github.com/abcxyz/pkg/logging"
)

var logger *slog.Logger

var myLogger = logging.DefaultLogger()

func ExampleNewFromEnv() {
	logger = logging.NewFromEnv("MY_APP_")
}

func ExampleNewFromEnv_setLevel() {
	logger = logging.SetLevel(logging.NewFromEnv("MY_APP_"), slog.LevelWarn)
}

func ExampleNew() {
	// Write to a buffer instead of stdout
	var b bytes.Buffer
	logger = logging.New(&b, slog.LevelInfo, logging.FormatJSON, false)
}

func ExampleSetLevel() {
	logging.SetLevel(myLogger, slog.LevelDebug) // level is now debug
}

func ExampleSetLevel_safe() {
	// This example demonstrates the totally safe way to set a level, assuming you
	// don't know if the logger is capable of changing levels dynamically.
	typ, ok := myLogger.Handler().(logging.LevelableHandler)
	if !ok {
		// not capable of setting levels
	}
	typ.SetLevel(slog.LevelDebug) // level is now debug
}

func ExampleDefaultLogger() {
	logger = logging.DefaultLogger()
}

func ExampleWithLogger() {
	ctx := context.Background()

	logger = logging.New(os.Stdout, slog.LevelDebug, logging.FormatText, true)
	ctx = logging.WithLogger(ctx, logger)

	logger = logging.FromContext(ctx) // same logger
}

func ExampleFromContext() {
	ctx := context.Background()

	logger = logging.New(os.Stdout, slog.LevelDebug, logging.FormatText, true)
	ctx = logging.WithLogger(ctx, logger)

	logger = logging.FromContext(ctx) // same logger
}
