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

package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	t.Run("default", func(t *testing.T) {
		t.Parallel()

		var b bytes.Buffer
		logger := New(&b, LevelInfo, FormatText, false)

		logger.Log(ctx, -19022, "very low level")
		logger.Log(ctx, LevelWarning, "engine failure")

		if got, want := b.String(), "very low level"; strings.Contains(got, want) {
			t.Errorf("expected %q to not contain %q", got, want)
		}
		if got, want := b.String(), "logging.googleapis.com/sourceLocation="; strings.Contains(got, want) {
			t.Errorf("expected %q to not contain %q", got, want)
		}
		if got, want := b.String(), "engine failure"; !strings.Contains(got, want) {
			t.Errorf("expected %q to contain %q", got, want)
		}
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		var b bytes.Buffer
		logger := New(&b, LevelInfo, FormatJSON, false)
		logger.Log(ctx, LevelWarning, "engine failure")

		if got, want := b.String(), `"message":"engine failure"`; !strings.Contains(got, want) {
			t.Errorf("expected %q to contain %q", got, want)
		}
	})

	t.Run("debug", func(t *testing.T) {
		t.Parallel()

		var b bytes.Buffer
		logger := New(&b, 0, FormatText, true)

		logger.Log(ctx, -19022, "very low level")

		if got, want := b.String(), "very low level"; !strings.Contains(got, want) {
			t.Errorf("expected %q to contain %q", got, want)
		}
		if got, want := b.String(), "logging.googleapis.com/sourceLocation="; !strings.Contains(got, want) {
			t.Errorf("expected %q to contain %q", got, want)
		}
	})
}

func TestNewFromEnv(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cases := []struct {
		name      string
		envPrefix string
		env       map[string]string

		wantLevel slog.Level
		wantPanic string
	}{
		{
			name:      "empty",
			wantLevel: LevelInfo,
		},

		// levels
		{
			name: "custom_level",
			env: map[string]string{
				"LOG_LEVEL": "debug",
			},
			wantLevel: LevelDebug,
		},
		{
			name: "invalid_level",
			env: map[string]string{
				"LOG_LEVEL": "pants",
			},
			wantPanic: "invalid value for LOG_LEVEL: no such level",
		},

		// formats
		{
			name: "custom_format",
			env: map[string]string{
				"LOG_FORMAT": "json",
			},
		},
		{
			name: "invalid_format",
			env: map[string]string{
				"LOG_FORMAT": "pants",
			},
			wantPanic: "invalid value for LOG_FORMAT: no such format",
		},

		// debug
		{
			name: "custom_debug",
			env: map[string]string{
				"LOG_DEBUG": "1",
			},
		},
		{
			name: "invalid_format",
			env: map[string]string{
				"LOG_DEBUG": "pants",
			},
			wantPanic: "invalid value for LOG_DEBUG: strconv.ParseBool",
		},

		// target
		{
			name: "empty_target",
			env: map[string]string{
				"LOG_TARGET": "",
			},
		},
		{
			name: "custom_target",
			env: map[string]string{
				"LOG_TARGET": "STDERR",
			},
		},
		{
			name: "invalid_target",
			env: map[string]string{
				"LOG_TARGET": "ME",
			},
			wantPanic: "invalid value for LOG_TARGET: no such target \"ME\"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if tc.wantPanic != "" {
					if r := recover(); r != nil {
						if got, want := fmt.Sprintf("%v", r), tc.wantPanic; !strings.Contains(got, want) {
							t.Errorf("expected %q to contain %q", got, want)
						}
					}
				}
			}()

			logger := newFromEnv(tc.envPrefix, func(k string) string {
				return tc.env[k]
			})

			if !logger.Handler().Enabled(ctx, tc.wantLevel) {
				t.Errorf("expected handler to be at least %s", tc.wantLevel)
			}
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	t.Parallel()

	logger1 := DefaultLogger()
	logger2 := DefaultLogger()

	if logger1 != logger2 {
		t.Errorf("expected default logger to be a singleton (got %v and %v)", logger1, logger2)
	}
}

func TestContext(t *testing.T) {
	t.Parallel()

	logger1 := slog.New(slog.NewTextHandler(io.Discard, nil))
	logger2 := slog.New(slog.NewTextHandler(io.Discard, nil))

	checkFromContext(t.Context(), t, DefaultLogger())

	ctx := WithLogger(t.Context(), logger1)
	checkFromContext(ctx, t, logger1)

	ctx = WithLogger(ctx, logger2)
	checkFromContext(ctx, t, logger2)
}

func checkFromContext(ctx context.Context, tb testing.TB, want *slog.Logger) {
	tb.Helper()

	if got := FromContext(ctx); want != got {
		tb.Errorf("unexpected logger in context. got: %v, want: %v", got, want)
	}
}
