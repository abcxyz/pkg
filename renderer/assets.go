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
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	htmltemplate "html/template"
	"io"
	"io/fs"
	"strings"
	"sync"
	texttemplate "text/template"
)

const sriPrefix = "sha512-"

var (
	cssIncludeTmpl = texttemplate.Must(texttemplate.New(`cssIncludeTmpl`).Parse(strings.TrimSpace(`
{{ range . -}}
<link rel="stylesheet" href="/{{.Path}}" integrity="{{.SRI}}" crossorigin="anonymous" referrerpolicy="no-referrer" />
{{ end }}
`)))

	cssIncludeTagCache htmltemplate.HTML
)

var (
	jsIncludeTmpl = texttemplate.Must(texttemplate.New(`jsIncludeTmpl`).Parse(strings.TrimSpace(`
{{ range . -}}
<script defer src="/{{.Path}}" integrity="{{.SRI}}" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
{{ end }}
`)))
	jsIncludeTagCache htmltemplate.HTML
)

// asset represents a javascript or css asset.
type asset struct {
	// Path is the virtual path, relative to the URL root.
	Path string

	// SRI is the sha384 resource integrity.
	SRI string
}

// assetIncludeTag searches the fs for all assets of the given search type and
// renders the template. In non-dev mode, the results are cached on the first
// invocation.
func assetIncludeTag(fsys fs.FS, tmpl *texttemplate.Template, cache *htmltemplate.HTML, devMode bool) func(string) (htmltemplate.HTML, error) {
	var mu sync.Mutex

	return func(search string) (htmltemplate.HTML, error) {
		if !devMode {
			mu.Lock()
			defer mu.Unlock()
			if *cache != "" {
				return *cache, nil
			}
		}
		// Check if this is a single file first.
		entries, err := fs.Glob(fsys, search)
		if err != nil {
			return "", fmt.Errorf("failed to search entries: %w", err)
		}

		list := make([]*asset, 0, len(entries))
		for _, name := range entries {
			f, err := fsys.Open(name)
			if err != nil {
				return "", fmt.Errorf("failed to open %s: %w", name, err)
			}

			integrity, err := generateSRI(f)
			if err != nil {
				return "", fmt.Errorf("failed to generate SRI for %s: %w", name, err)
			}

			list = append(list, &asset{
				Path: name,
				SRI:  integrity,
			})
		}

		var b bytes.Buffer
		if err := tmpl.Execute(&b, list); err != nil {
			return "", fmt.Errorf("failed to render %s asset: %w", search, err)
		}
		result := htmltemplate.HTML(b.String()) //nolint:gosec // No user-supplied input

		if !devMode {
			*cache = result
		}

		return result, nil
	}
}

// generateSRI is a helper that generates an SRI from the given reader. It
// closes the given reader.
func generateSRI(r io.ReadCloser) (string, error) {
	defer r.Close()

	h := sha512.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to generate sri hash: %w", err)
	}
	return sriPrefix + base64.RawStdEncoding.EncodeToString(h.Sum(nil)), nil
}
