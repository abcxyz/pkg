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

package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestConfig_NewConfig(t *testing.T) {
	t.Parallel()

	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	testClient := &http.Client{Timeout: 10 * time.Second}

	cases := []struct {
		name      string
		appID     string
		installID string
		options   []ConfigOption
		want      *Config
	}{
		{
			name:      "basic_config",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{},
			want: &Config{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
			},
		},
		{
			name:      "with_token_url_pattern",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{WithAccessTokenURLPattern("test/%s")},
			want: &Config{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: "test/%s",
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
			},
		},
		{
			name:      "with_token_expiration",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{WithJWTTokenExpiration(3 * time.Minute)},
			want: &Config{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    3 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
			},
		},
		{
			name:      "with_token_caching",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{WithJWTTokenCaching(1 * time.Minute)},
			want: &Config{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      8 * time.Minute,
			},
		},
		{
			name:      "with_http_client",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{WithHTTPClient(testClient)},
			want: &Config{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				client:                testClient,
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := NewConfig(tc.appID, tc.installID, rsaPrivateKey, tc.options...)
			if diff := cmp.Diff(tc.want, got,
				cmp.AllowUnexported(Config{})); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubApp_AccessToken(t *testing.T) {
	t.Parallel()

	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name        string
		appID       string
		installID   string
		options     []ConfigOption
		request     *TokenRequest
		want        string
		expErr      string
		handlerFunc http.HandlerFunc
	}{
		{
			name:        "basic_request",
			appID:       "test-app-id",
			installID:   "test-install-id",
			options:     []ConfigOption{},
			request:     &TokenRequest{Repositories: []string{"test"}, Permissions: map[string]string{"test": "test"}},
			want:        `{"token":"this-is-the-token-from-github"}`,
			expErr:      "",
			handlerFunc: nil,
		},
		{
			name:      "non_201_response",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{},
			request:   &TokenRequest{Repositories: []string{"test"}, Permissions: map[string]string{"test": "test"}},
			expErr:    "failed to retrieve token from GitHub - Status: 500 Internal Server Error - Body: ",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			},
		},
		{
			name:      "201_not_json",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{},
			request:   &TokenRequest{Repositories: []string{"test"}, Permissions: map[string]string{"test": "test"}},
			expErr:    "invalid access token from GitHub - Body: not json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprintf(w, "not json")
			},
		},
		{
			name:      "201_no_body",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{},
			request:   &TokenRequest{Repositories: []string{"test"}, Permissions: map[string]string{"test": "test"}},
			expErr:    "invalid access token from GitHub - Body:",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
		},
		{
			name:        "allow_empty_repositories",
			appID:       "test-app-id",
			installID:   "test-install-id",
			options:     []ConfigOption{},
			request:     &TokenRequest{Repositories: []string{}, Permissions: map[string]string{"test": "test"}},
			want:        `{"token":"this-is-the-token-from-github"}`,
			expErr:      "",
			handlerFunc: nil,
		},
		{
			name:      "missing_repositories",
			appID:     "test-app-id",
			installID: "test-install-id",
			options:   []ConfigOption{},
			request:   &TokenRequest{Permissions: map[string]string{"test": "test"}},
			expErr:    "requested repositories cannot be nil, did you mean to use AccessTokenAllRepos to request all repos?",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fakeGitHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.handlerFunc != nil {
					tc.handlerFunc(w, r)
					return
				}

				if r.Header.Get("Accept") != "application/vnd.github+json" {
					w.WriteHeader(500)
					fmt.Fprintf(w, "missing accept header")
					return
				}
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					w.WriteHeader(500)
					fmt.Fprintf(w, "missing or malformed authorization header")
					return
				}
				w.WriteHeader(201)
				fmt.Fprintf(w, `{"token":"this-is-the-token-from-github"}`)
			}))
			tc.options = append(tc.options, WithAccessTokenURLPattern(fakeGitHub.URL+"/%s/access_tokens"))

			app := New(NewConfig(tc.appID, tc.installID, rsaPrivateKey, tc.options...))
			got, err := app.AccessToken(context.Background(), tc.request)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Errorf(diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
