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

func TestDefault_Warn(t *testing.T) {
	t.Parallel()

	logger := Default()
	if got, want := logger.Level().CapitalString(), "WARN"; got != want {
		t.Errorf("Default log level got=%v, want=%v", got, want)
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

func TestFactory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		level     string
		wantLevel string
	}{
		{
			name:      "default",
			wantLevel: "warn",
		},
		{
			name:      "overwrite_level",
			level:     "debug",
			wantLevel: "debug",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := &Factory{}
			if tc.level != "" {
				f.SetLevel(tc.level)
			}

			logger := f.New()
			if got, want := logger.Level().String(), tc.wantLevel; got != want {
				t.Errorf("logger level got=%v, want=%v", got, want)
			}
		})
	}
}
