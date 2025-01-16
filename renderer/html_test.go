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

func TestRenderHTMLStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys := fstest.MapFS{
		"template.html": &fstest.MapFile{
			Data: []byte(`
				{{ define "template" }}
					Hello World!
				{{ end }}
			`),
			Mode: 0o600,
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
			name: "no_template",
			tmpl: "non_existenxt",
			exp:  "&#34;non_existenxt&#34; is undefined",
		},
		{
			name: "success",
			tmpl: "template",
			exp:  "Hello World!",
		},
	}

	for _, tc := range cases {
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
