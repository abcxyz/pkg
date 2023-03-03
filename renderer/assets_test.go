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
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAssetIncludeTag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys := fstest.MapFS{
		"static/css/index.css": &fstest.MapFile{
			Data: []byte(`
				body {
					background: '#000';
				}
			`),
		},
		"static/js/index.js": &fstest.MapFile{
			Data: []byte(`
				alert('hi');
			`),
		},
		"template.html": &fstest.MapFile{
			Data: []byte(`
				{{ define "template" }}
					{{ cssIncludeTag }}
					{{ jsIncludeTag }}
				{{ end }}
			`),
		},
	}

	r, err := New(ctx, sys, WithDebug(true))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		tmpl string
		exp  string
	}{
		{
			name: "css",
			tmpl: "template",
			exp:  `<link rel="stylesheet" href="/static/css/index.css" integrity="sha512-CZ/ju4L53fGRLdwSN7Cl0QyO336hYAIgXyASelQXWHb354JD9u+MBMEgFtCjKuFzQ84MYzBHcOzPEqK/roRvYA" crossorigin="anonymous" referrerpolicy="no-referrer" />`,
		},
		{
			name: "js",
			tmpl: "template",
			exp:  `<script defer src="/static/js/index.js" integrity="sha512-DFr7R1UkHXJDzmXpnOEq+oYqK4oj/4aCo4zeWnWXo5DkAV2TNwNlrGM4J8nEYwTCFcXwwshSgaSqN8UFv6wZ6w" crossorigin="anonymous" referrerpolicy="no-referrer"></script>`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()

			r.RenderHTMLStatus(w, 200, tc.tmpl, nil)
			w.Flush()

			if got, want := w.Body.String(), tc.exp; !strings.Contains(got, want) {
				t.Errorf("expected\n\n%s\n\nto contain %q", got, want)
			}
		})
	}
}
