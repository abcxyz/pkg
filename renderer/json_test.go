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

package renderer

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// cycled exists to force a JSON cycle to test error handling.
	type cycled struct {
		Next *cycled `json:"next,omitempty"`
	}

	cycle := &cycled{}
	cycle.Next = cycle

	cases := []struct {
		name  string
		code  int
		in    any
		debug bool
		exp   string
	}{
		{
			name: "nil_200",
			code: 200,
			in:   nil,
			exp:  `{"ok":true}`,
		},
		{
			name: "nil_403",
			code: 403,
			in:   nil,
			exp:  `{"errors":["Forbidden"]}`,
		},
		{
			name: "joined_error",
			code: 400,
			in:   errors.Join(fmt.Errorf("one"), fmt.Errorf("two")),
			exp:  `{"errors":["one","two"]}`,
		},
		{
			name: "error_slice",
			code: 400,
			in:   []error{fmt.Errorf("one"), fmt.Errorf("two")},
			exp:  `{"errors":["one","two"]}`,
		},
		{
			name: "error",
			code: 400,
			in:   []error{fmt.Errorf("one")},
			exp:  `{"errors":["one"]}`,
		},
		{
			name: "complex_structure",
			code: 200,
			in: map[string]any{
				"foo": []string{
					"one",
					"two",
				},
			},
			exp: `{"foo":["one","two"]}`,
		},
		{
			name: "json_cycle",
			code: 500,
			in:   cycle,
			exp:  `{"errors":["An internal error occurred."]}`,
		},
		{
			name:  "json_cycle_debug",
			code:  500,
			in:    cycle,
			debug: true,
			exp:   `{"errors":["json: unsupported value: encountered a cycle via *renderer.cycled"]}`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()

			r, err := New(ctx, nil, WithDebug(tc.debug))
			if err != nil {
				t.Fatal(err)
			}

			r.RenderJSON(w, tc.code, tc.in)
			w.Flush()

			if got, want := w.Header().Get("Content-Type"), "application/json"; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
			if got, want := w.Code, tc.code; got != want {
				t.Errorf("expected %d to be %d", got, want)
			}

			if got, want := strings.TrimSpace(w.Body.String()), tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}
