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
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAssetIncludeTag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name string
		tmpl string
		exp  string
	}{
		{
			name: "css_single",
			tmpl: `{{ cssIncludeTag "css/one.css" }}`,
			exp:  `<link rel="stylesheet" href="/css/one.css" integrity="sha512-HO667IerAcsnrUiKG3h8BrdLoCQVIJ7wTVKR0hyiKb0rrJDIbq25rEHg0QvcrQN+Qwc7LRs9+YxypwqlsHnQTw" crossorigin="anonymous" referrerpolicy="no-referrer" />`,
		},
		{
			name: "css_multi",
			tmpl: `{{ cssIncludeTag "css/*.css" }}`,
			exp: `
<link rel="stylesheet" href="/css/one.css" integrity="sha512-HO667IerAcsnrUiKG3h8BrdLoCQVIJ7wTVKR0hyiKb0rrJDIbq25rEHg0QvcrQN+Qwc7LRs9+YxypwqlsHnQTw" crossorigin="anonymous" referrerpolicy="no-referrer" />
<link rel="stylesheet" href="/css/two.css" integrity="sha512-CZ/ju4L53fGRLdwSN7Cl0QyO336hYAIgXyASelQXWHb354JD9u+MBMEgFtCjKuFzQ84MYzBHcOzPEqK/roRvYA" crossorigin="anonymous" referrerpolicy="no-referrer" />
		`,
		},
		{
			name: "js_single",
			tmpl: `{{ jsIncludeTag "js/one.js" }}`,
			exp:  `<script defer src="/js/one.js" integrity="sha512-YPaegQoTqkvUCdGwIwEOxL91PyfbGEQEBYz0m+em4XCjDWc6W55nu8xl9/iVZfJchfNPdlOBsbNKi9oN6n4a7Q" crossorigin="anonymous" referrerpolicy="no-referrer"></script>`,
		},
		{
			name: "js_multi",
			tmpl: `{{ jsIncludeTag "js/*.js" }}`,
			exp: `
<script defer src="/js/one.js" integrity="sha512-YPaegQoTqkvUCdGwIwEOxL91PyfbGEQEBYz0m+em4XCjDWc6W55nu8xl9/iVZfJchfNPdlOBsbNKi9oN6n4a7Q" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
<script defer src="/js/two.js" integrity="sha512-6OtiBKWBCzIySzWxzqVGnB5SzK5N58jRo8UDLw53i3EY5nDRYITiU+VerGa7ZkNdUp5y86mg5/Hmf+lUEQ2Ulg" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
		`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sys := fstest.MapFS{
				"css/one.css": &fstest.MapFile{
					Data: []byte(`
				html {
					background: '#000';
				}
			`),
				},
				"css/two.css": &fstest.MapFile{
					Data: []byte(`
				body {
					background: '#000';
				}
			`),
				},
				"js/one.js": &fstest.MapFile{
					Data: []byte(`
				alert('hello');
			`),
				},
				"js/two.js": &fstest.MapFile{
					Data: []byte(`
				alert('world');
			`),
				},
				"template.html": &fstest.MapFile{
					Data: []byte(fmt.Sprintf(`
						{{ define "template" }}%s{{ end }}
					`, tc.tmpl)),
				},
			}

			r, err := New(ctx, sys, WithDebug(true))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			r.RenderHTMLStatus(w, 200, "template", nil)
			w.Flush()

			if got, want := strings.TrimSpace(w.Body.String()), strings.TrimSpace(tc.exp); !strings.Contains(got, want) {
				t.Errorf("expected\n\n%s\n\nto contain %q", got, want)
			}
		})
	}
}
