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

package renderer_test

import (
	"net/http"
	"sort"
	"testing/fstest"
	"time"

	"github.com/abcxyz/pkg/renderer"
)

type Server struct {
	db map[string]any
	h  *renderer.Renderer
}

func NewServer() *Server {
	// Normally this would come from the filesystem, but to make the test fit in a
	// single file...
	fsys := fstest.MapFS{
		"users/index.html": &fstest.MapFile{
			Data: []byte(`
				{{ define "users/index" }}
					<ul>
						{{ range .Users }}
							<li>{{ . | toUpper }}</li>
						{{ end }}
					</ul>
				{{ end }}
			`),
			Mode: 0o600,
		},
	}

	h, err := renderer.New(fsys, renderer.WithDebug(true))
	if err != nil {
		panic(err)
	}

	return &Server{
		db: make(map[string]any),
		h:  h,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/users.html", s.HandleUsersIndex())
	mux.Handle("/users.json", s.HandleUsersAPI())
	return mux
}

func (s *Server) HandleUsersIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.h.RenderHTML(w, "users/index", s.users())
	})
}

func (s *Server) HandleUsersAPI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.h.RenderJSON(w, 200, s.users())
	})
}

func (s *Server) users() []string {
	users := make([]string, 0, len(s.db))
	for k := range s.db {
		users = append(users, k)
	}
	sort.Strings(users)
	return users
}

func Example() {
	s := NewServer()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: s.Routes(),

		ReadHeaderTimeout: 2 * time.Second,
	}
	_ = srv.ListenAndServe()
}
