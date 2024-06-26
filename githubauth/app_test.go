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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/abcxyz/pkg/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaPrivateKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaPrivateKey),
	})
	rsaPrivateKeyString := string(rsaPrivateKeyBytes)

	testClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	cases := []struct {
		name           string
		appID          string
		installationID string

		privateKey       *rsa.PrivateKey
		privateKeyString string
		privateKeyBytes  []byte

		options []Option

		want      *App
		wantError string
	}{
		{
			name:           "private_key_rsa_key",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			want: &App{
				appID:      "test-app-id",
				privateKey: rsaPrivateKey,
				baseURL:    "https://api.github.com",
				httpClient: &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:             "private_key_string",
			appID:            "test-app-id",
			installationID:   "test-install-id",
			privateKeyString: rsaPrivateKeyString,
			want: &App{
				appID:      "test-app-id",
				privateKey: rsaPrivateKey,
				baseURL:    "https://api.github.com",
				httpClient: &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:            "private_key_bytes",
			appID:           "test-app-id",
			installationID:  "test-install-id",
			privateKeyBytes: rsaPrivateKeyBytes,
			want: &App{
				appID:      "test-app-id",
				privateKey: rsaPrivateKey,
				baseURL:    "https://api.github.com",
				httpClient: &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:           "with_base_url",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			options:        []Option{WithBaseURL("https://foo.bar/")},
			want: &App{
				appID:      "test-app-id",
				privateKey: rsaPrivateKey,
				baseURL:    "https://foo.bar",
				httpClient: &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:           "with_http_client",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			options:        []Option{WithHTTPClient(testClient)},
			want: &App{
				appID:      "test-app-id",
				privateKey: rsaPrivateKey,
				baseURL:    "https://api.github.com",
				httpClient: testClient,
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got *App
			var err error
			switch {
			case tc.privateKey != nil:
				got, err = NewApp(tc.appID, tc.privateKey, tc.options...)
			case tc.privateKeyString != "":
				got, err = NewApp(tc.appID, tc.privateKeyString, tc.options...)
			case tc.privateKeyBytes != nil:
				got, err = NewApp(tc.appID, tc.privateKeyBytes, tc.options...)
			default:
				t.Fatal("missing private key")
			}
			if diff := testutil.DiffErrString(err, tc.wantError); diff != "" {
				t.Fatalf("unexpected err: %s", diff)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(App{}),
				cmpopts.IgnoreFields(App{},
					"installationCache",
					"installationCacheLock"),
			}
			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestApp_AppToken(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	app, err := NewApp("my-app-id", privateKey)
	if err != nil {
		t.Fatal(err)
	}

	token, err := app.AppToken()
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(token, ".")
	if exp := 3; len(parts) != exp {
		t.Fatalf("expected %d items, got %q", exp, parts)
	}

	header := testBase64Decode(t, parts[0])
	if got, want := string(header), `{"alg":"RS256","typ":"JWT"}`; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	body := testBase64Decode(t, parts[1])
	if got, want := string(body), `"iss":"my-app-id"`; !strings.Contains(got, want) {
		t.Errorf("expected %q to contain %q", got, want)
	}

	signature := testBase64Decode(t, parts[2])

	h := sha256.New()
	h.Write([]byte(parts[0] + "." + parts[1]))
	digest := h.Sum(nil)

	if err := rsa.VerifyPKCS1v15(&privateKey.PublicKey, crypto.SHA256, digest, signature); err != nil {
		t.Fatal(err)
	}
}

func TestApp_OAuthAppTokenSource(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	app, err := NewApp("my-app-id", privateKey)
	if err != nil {
		t.Fatal(err)
	}

	token, err := app.AppToken()
	if err != nil {
		t.Fatal(err)
	}

	oauthToken, err := app.OAuthAppTokenSource().Token()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := oauthToken.AccessToken, token; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestApp_AccessToken(t *testing.T) {
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
		tc := tc

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

			app, err := NewApp("my-app-id", privateKey, WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}

			installation, err := app.InstallationForID(ctx, "123")
			if err != nil {
				t.Fatal(err)
			}

			token, err := installation.AccessToken(ctx, tc.req)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Errorf(diff)
			}

			if got, want := token, tc.expToken; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestApp_SelectedReposTokenSource(t *testing.T) {
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

	app, err := NewApp("my-app-id", privateKey, WithBaseURL(srv.URL))
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

func TestApp_AccessTokenAllRepos(t *testing.T) {
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
		tc := tc

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

			app, err := NewApp("my-app-id", privateKey, WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}

			installation, err := app.InstallationForID(ctx, "123")
			if err != nil {
				t.Fatal(err)
			}

			token, err := installation.AccessTokenAllRepos(ctx, tc.req)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Errorf(diff)
			}

			if got, want := token, tc.expToken; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestApp_AllReposTokenSource(t *testing.T) {
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

	app, err := NewApp("my-app-id", privateKey, WithBaseURL(srv.URL))
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

func testBase64Decode(tb testing.TB, s string) []byte {
	tb.Helper()

	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}
