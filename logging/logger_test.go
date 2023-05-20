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

package logging

import (
	"context"
	"testing"

	"github.com/abcxyz/pkg/cli"
	"go.uber.org/zap"
)

func TestContext(t *testing.T) {
	t.Parallel()

	logger1 := zap.NewNop().Sugar()
	logger2 := zap.NewExample().Sugar()

	checkFromContext(context.Background(), t, Default())

	ctx := WithLogger(context.Background(), logger1)
	checkFromContext(ctx, t, logger1)

	ctx = WithLogger(ctx, logger2)
	checkFromContext(ctx, t, logger2)
}

func checkFromContext(ctx context.Context, tb testing.TB, want *zap.SugaredLogger) {
	tb.Helper()

	if got := FromContext(ctx); want != got {
		tb.Errorf("unexpected logger in context. got: %v, want: %v", got, want)
	}
}

func TestNewFromEnv(t *testing.T) { //nolint: paralleltest // Need to use t.Setenv
	t.Setenv("TEST_NEWFROMENV_LOG_LEVEL", "debug")
	logger := NewFromEnv("TEST_NEWFROMENV_")
	if logger == nil {
		t.Errorf("NewFromEnv got unexpected nil logger")
	}
	if got, want := zap.DebugLevel, logger.Level(); got != want {
		t.Errorf("NewFromEnv logger level got=%v, want=%v", got, want)
	}
}

func TestNewFromFlags(t *testing.T) {
	t.Parallel()

	fs := cli.NewFlagSet()
	RegisterFlags(fs)
	if err := fs.Parse([]string{"--log-level=debug"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	logger := NewFromFlags()
	if logger == nil {
		t.Errorf("NewFromEnv got unexpected nil logger")
	}
	if got, want := zap.DebugLevel, logger.Level(); got != want {
		t.Errorf("NewFromEnv logger level got=%v, want=%v", got, want)
	}
}
