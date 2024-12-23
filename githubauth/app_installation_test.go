// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package githubauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abcxyz/pkg/testutil"
)

func TestAppInstallation_AccessToken(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		handler  http.HandlerFunc
		req      *TokenRequest
		expToken string
		expErr   string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprint(w, `{"token": "ghs_expectedtoken"}`)
			},
			req: &TokenRequest{
				Repositories: []string{"my-repo"},
				Permissions:  map[string]string{"issues": "write"},
			},
			expToken: "ghs_expectedtoken",
		},
		{
			name: "success_empty_repositories",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprint(w, `{"token": "ghs_expectedtoken"}`)
			},
			req: &TokenRequest{
				Repositories: []string{},
				Permissions:  map[string]string{"issues": "write"},
			},
			expToken: "ghs_expectedtoken",
		},
		{
			name: "fails_missing_repositories",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
			req: &TokenRequest{
				Repositories: nil,
				Permissions:  map[string]string{"issues": "write"},
			},
			expErr: "requested repositories cannot be nil",
		},
		{
			name: "not_201",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			},
			req: &TokenRequest{
				Repositories: []string{"my-repo"},
				Permissions:  map[string]string{"issues": "write"},
			},
			expErr: "expected 200 to be 201",
		},
		{
			name: "201_not_json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprint(w, `this is not json`)
			},
			req: &TokenRequest{
				Repositories: []string{"my-repo"},
				Permissions:  map[string]string{"issues": "write"},
			},
			expErr: "failed to parse response as json",
		},
		{
			name: "201_empty_body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
			req: &TokenRequest{
				Repositories: []string{"my-repo"},
				Permissions:  map[string]string{"issues": "write"},
			},
			expErr: "failed to parse response as json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			mux := http.NewServeMux()
			mux.Handle("/app/installations/123", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"access_tokens_url": "http://%s/app/installations/123/access_tokens"}`, r.Host)
			}))
			mux.Handle("/app/installations/123/access_tokens", tc.handler)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			app, err := NewApp("my-app-id", WithPrivateKeySigner(privateKey), WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}

			installation, err := app.InstallationForID(ctx, "123")
			if err != nil {
				t.Fatal(err)
			}

			token, err := installation.AccessToken(ctx, tc.req)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Error(diff)
			}

			if got, want := token, tc.expToken; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestAppInstallation_SelectedReposTokenSource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/app/installations/123", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"access_tokens_url": "http://%s/app/installations/123/access_tokens"}`, r.Host)
	}))
	mux.Handle("/app/installations/123/access_tokens", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprint(w, `{"token": "ghs_expectedtoken"}`)
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	app, err := NewApp("my-app-id", WithPrivateKeySigner(privateKey), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	installation, err := app.InstallationForID(ctx, "123")
	if err != nil {
		t.Fatal(err)
	}

	src := installation.SelectedReposTokenSource(map[string]string{"issues": "write"}, "my-repo")
	githubToken, err := src.GitHubToken(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := githubToken, "ghs_expectedtoken"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAppInstallation_AccessTokenAllRepos(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		handler  http.HandlerFunc
		req      *TokenRequestAllRepos
		expToken string
		expErr   string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprint(w, `{"token": "ghs_expectedtoken"}`)
			},
			req: &TokenRequestAllRepos{
				Permissions: map[string]string{"issues": "write"},
			},
			expToken: "ghs_expectedtoken",
		},
		{
			name: "not_201",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			},
			expErr: "expected 200 to be 201",
		},
		{
			name: "201_not_json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprint(w, `this is not json`)
			},
			expErr: "failed to parse response as json",
		},
		{
			name: "201_empty_body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
			expErr: "failed to parse response as json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			mux := http.NewServeMux()
			mux.Handle("/app/installations/123", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"access_tokens_url": "http://%s/app/installations/123/access_tokens"}`, r.Host)
			}))
			mux.Handle("/app/installations/123/access_tokens", tc.handler)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			app, err := NewApp("my-app-id", WithPrivateKeySigner(privateKey), WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}

			installation, err := app.InstallationForID(ctx, "123")
			if err != nil {
				t.Fatal(err)
			}

			token, err := installation.AccessTokenAllRepos(ctx, tc.req)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Error(diff)
			}

			if got, want := token, tc.expToken; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestAppInstallation_AllReposTokenSource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/app/installations/123", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"access_tokens_url": "http://%s/app/installations/123/access_tokens"}`, r.Host)
	}))
	mux.Handle("/app/installations/123/access_tokens", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprint(w, `{"token": "ghs_expectedtoken"}`)
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	app, err := NewApp("my-app-id", WithPrivateKeySigner(privateKey), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	installation, err := app.InstallationForID(ctx, "123")
	if err != nil {
		t.Fatal(err)
	}

	src := installation.AllReposTokenSource(map[string]string{"issues": "write"})
	githubToken, err := src.GitHubToken(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := githubToken, "ghs_expectedtoken"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}
