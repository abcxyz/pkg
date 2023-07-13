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
	"html/template"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWithTemplateFuncs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name string
		fns  []template.FuncMap
		exp  template.FuncMap
	}{
		{
			name: "nil",
			fns:  []template.FuncMap{nil, nil, nil},
			exp:  template.FuncMap{},
		},
		{
			name: "single",
			fns: []template.FuncMap{
				map[string]any{"a": "a"},
			},
			exp: map[string]any{
				"a": "a",
			},
		},
		{
			name: "multiple",
			fns: []template.FuncMap{
				map[string]any{
					"a": "a",
					"b": "b",
				},
				map[string]any{"c": "c"},
			},
			exp: map[string]any{
				"a": "a",
				"b": "b",
				"c": "c",
			},
		},
		{
			name: "overwrites",
			fns: []template.FuncMap{
				map[string]any{
					"a": "a",
					"b": "b",
				},
				map[string]any{"a": "2"},
				map[string]any{"b": "2"},
			},
			exp: map[string]any{
				"a": "2",
				"b": "2",
			},
		},
		{
			name: "deletes_nil",
			fns: []template.FuncMap{
				map[string]any{
					"a": "a",
					"b": "b",
				},
				map[string]any{"a": nil},
			},
			exp: map[string]any{
				"b": "b",
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, err := New(ctx, nil, WithTemplateFuncs(tc.fns...))
			if err != nil {
				t.Fatal(err)
			}

			// Remove any built-in funcs for comparisons.
			for k := range builtinFuncs() {
				delete(h.templateFuncs, k)
			}
			delete(h.templateFuncs, "cssIncludeTag")
			delete(h.templateFuncs, "jsIncludeTag")

			if diff := cmp.Diff(h.templateFuncs, tc.exp); diff != "" {
				t.Errorf("template diff (+want, -got):\n%s", diff)
			}
		})
	}
}
