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

// Package render exposes high-performance HTML and JSON rendering
// functionality. Most use cases can use the [Renderer] without modification.
// More advanced use cases can customize template functions and error handling.
//
// The renderer accepts a filesystem ([fs.FS]). In most cases, this will be a
// filesystem on disk. However, it accepts the FS interface for testing and
// [embed] purposes. Because embed does not perform hot reloading, you may want
// to use a different fs for development versus production:
//
//	//go:embed assets assets/**/*
//	var _assetsFS embed.FS
//
//	func AssetsFS() fs.FS {
//	  // In dev, just read directly from disk
//	  if v, _ := strconv.ParseBool(os.Getenv("DEV_MODE")); v {
//	    return os.DirFS("./assets")
//	  }
//
//	  // Otherwise use the embedded fs
//	  return _assetsFS
//	}
//
// The the renderer includes some prebuilt functions, including static asset
// parsing for CSS and Javascript files. The renderer assumes these files exist
// in a `static/css` and `static/js` directory at the root of the provided
// filesystem.
//
//	assets/
//	  \_ static/
//	    \_ css/
//	    \_ js/
//
// To render the include tags in a template:
//
//	{{ define "home" }}
//	  {{ cssIncludeTag }}
//	  {{ jsIncludeTag }}
//	{{ end }}
package renderer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"
	"sync"
)

// Renderer is responsible for rendering various content and templates like HTML
// and JSON responses. This implementation caches templates and uses a pool of
// buffers.
type Renderer struct {
	// rendererPool is a pool of *bytes.Buffer, used as a rendering buffer to
	// prevent partial responses being sent to clients.
	rendererPool *sync.Pool

	// templates is the actual collection of templates. templatesLock is a mutex
	// to prevent concurrent modification of the templates field.
	templates     *template.Template
	templatesLock sync.RWMutex

	// fs is the underlying filesystem to read.
	fs fs.FS

	// debug indicates templates should be reloaded on each invocation and real
	// error responses should be rendered. Do not enable in production.
	debug bool

	// onError is a function that is called when irrecoverable errors are
	// encountered. This is guaranteed to be non-nil when calling [New].
	onError func(err error)

	// templateFuncs is the compiled list of template functions.
	templateFuncs template.FuncMap
}

// Option is an interface for options to creating a renderer.
type Option func(*Renderer) *Renderer

// WithDebug configures debugging on the renderer.
func WithDebug(v bool) Option {
	return func(r *Renderer) *Renderer {
		r.debug = v
		return r
	}
}

// WithOnError overwrites the onError handler with the given function. This
// handler is invoked when an irrecoverable error occurs while rendering, but
// information cannot be sent back to the client. For example, if HTTP rendering
// fails after a partial response has been sent.
func WithOnError(fn func(err error)) Option {
	return func(r *Renderer) *Renderer {
		r.onError = fn
		return r
	}
}

// WithTemplateFuncs registers additional template functions. The renderer
// includes many helpful functions, but some applications may wish to
// inject/define their own template helpers. Functions in this map take
// precedence over the built-in list.
func WithTemplateFuncs(fns template.FuncMap) Option {
	return func(r *Renderer) *Renderer {
		r.templateFuncs = fns
		return r
	}
}

// New creates a new renderer with the given details.
func New(ctx context.Context, fsys fs.FS, opts ...Option) (*Renderer, error) {
	r := &Renderer{
		rendererPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
		fs: fsys,
	}

	for _, opt := range opts {
		if opt != nil {
			r = opt(r)
		}
	}

	// Ensure there's an error handler so we don't have to nil-check each time.
	if r.onError == nil {
		r.onError = func(err error) {}
	}

	// Wrap the error function to recover from panics.
	origOnError := r.onError
	r.onError = func(err error) {
		defer func() {
			if r := recover(); r != nil {
				// do nothing
				_ = r
			}
		}()
		origOnError(err)
	}

	// Compile template functions.
	fns := builtinFuncs()
	fns["cssIncludeTag"] = assetIncludeTag(r.fs, "static/css", cssIncludeTmpl, &cssIncludeTagCache, r.debug)
	fns["jsIncludeTag"] = assetIncludeTag(r.fs, "static/js", jsIncludeTmpl, &jsIncludeTagCache, r.debug)
	for k, v := range r.templateFuncs {
		fns[k] = v
	}
	r.templateFuncs = fns

	// Load initial templates
	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// executeTemplate executes a single HTML template with the provided data.
func (r *Renderer) executeTemplate(w io.Writer, name string, data interface{}) error {
	r.templatesLock.RLock()
	defer r.templatesLock.RUnlock()

	if r.templates == nil {
		return fmt.Errorf("no html templates are defined")
	}

	return r.templates.ExecuteTemplate(w, name, data) //nolint:wrapcheck // There's no additional context we can add
}

// loadTemplates loads or reloads all templates.
func (r *Renderer) loadTemplates() error {
	r.templatesLock.Lock()
	defer r.templatesLock.Unlock()

	if r.fs == nil {
		return nil
	}

	htmltmpl := template.New("").
		Option("missingkey=zero").
		Funcs(r.templateFuncs)

	if err := loadTemplates(r.fs, htmltmpl); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	r.templates = htmltmpl
	return nil
}

func loadTemplates(fsys fs.FS, tmpl *template.Template) error {
	// You might be thinking to yourself, wait, why don't you just use
	// template.ParseFS(fsys, "**/*.html"). Well, still as of Go 1.16, glob
	// doesn't support shopt globbing, so you still have to walk the entire
	// filepath.
	if err := fs.WalkDir(fsys, ".", func(pth string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".html") {
			if _, err := tmpl.ParseFS(fsys, pth); err != nil {
				return fmt.Errorf("failed to parse %s: %w", pth, err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk filesystem for templates: %w", err)
	}

	return nil
}
