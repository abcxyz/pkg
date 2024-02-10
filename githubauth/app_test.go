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

func TestConfig_New(t *testing.T) {
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
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				httpClient:            &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:             "private_key_string",
			appID:            "test-app-id",
			installationID:   "test-install-id",
			privateKeyString: rsaPrivateKeyString,
			want: &App{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				httpClient:            &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:            "private_key_bytes",
			appID:           "test-app-id",
			installationID:  "test-install-id",
			privateKeyBytes: rsaPrivateKeyBytes,
			want: &App{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				httpClient:            &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:           "with_token_url_pattern",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			options:        []Option{WithAccessTokenURLPattern("test/%s")},
			want: &App{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: "test/%s",
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				httpClient:            &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:           "with_token_expiration",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			options:        []Option{WithJWTTokenExpiration(3 * time.Minute)},
			want: &App{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    3 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				httpClient:            &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:           "with_token_caching",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			options:        []Option{WithJWTTokenCaching(1 * time.Minute)},
			want: &App{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      8 * time.Minute,
				httpClient:            &http.Client{Timeout: 10 * time.Second},
			},
		},
		{
			name:           "with_http_client",
			appID:          "test-app-id",
			installationID: "test-install-id",
			privateKey:     rsaPrivateKey,
			options:        []Option{WithHTTPClient(testClient)},
			want: &App{
				AppID:                 "test-app-id",
				InstallationID:        "test-install-id",
				PrivateKey:            rsaPrivateKey,
				accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
				jwtTokenExpiration:    9 * time.Minute,
				jwtCacheDuration:      0 * time.Nanosecond,
				httpClient:            testClient,
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
				got, err = NewApp(tc.appID, tc.installationID, tc.privateKey, tc.options...)
			case tc.privateKeyString != "":
				got, err = NewApp(tc.appID, tc.installationID, tc.privateKeyString, tc.options...)
			case tc.privateKeyBytes != nil:
				got, err = NewApp(tc.appID, tc.installationID, tc.privateKeyBytes, tc.options...)
			default:
				t.Fatal("missing private key")
			}
			if diff := testutil.DiffErrString(err, tc.wantError); diff != "" {
				t.Fatalf("unexpected err: %s", diff)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(App{}),
				cmpopts.IgnoreFields(App{}, "jwtCache"),
			}
			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
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
		request     *TokenRequest
		want        string
		expErr      string
		handlerFunc http.HandlerFunc
	}{
		{
			name:      "basic_request",
			appID:     "test-app-id",
			installID: "test-install-id",
			request: &TokenRequest{
				Repositories: []string{"test"},
				Permissions:  map[string]string{"test": "test"},
			},
			want:        `{"token":"this-is-the-token-from-github"}`,
			expErr:      "",
			handlerFunc: nil,
		},
		{
			name:      "non_201_response",
			appID:     "test-app-id",
			installID: "test-install-id",
			request: &TokenRequest{
				Repositories: []string{"test"},
				Permissions:  map[string]string{"test": "test"},
			},
			expErr: "failed to retrieve token from GitHub - Status: 500 Internal Server Error - Body: ",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			},
		},
		{
			name:      "201_not_json",
			appID:     "test-app-id",
			installID: "test-install-id",
			request: &TokenRequest{
				Repositories: []string{"test"},
				Permissions:  map[string]string{"test": "test"},
			},
			expErr: "invalid access token from GitHub - Body: not json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				fmt.Fprintf(w, "not json")
			},
		},
		{
			name:      "201_no_body",
			appID:     "test-app-id",
			installID: "test-install-id",
			request: &TokenRequest{
				Repositories: []string{"test"},
				Permissions:  map[string]string{"test": "test"},
			},
			expErr: "invalid access token from GitHub - Body:",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
		},
		{
			name:      "allow_empty_repositories",
			appID:     "test-app-id",
			installID: "test-install-id",
			request: &TokenRequest{
				Repositories: []string{},
				Permissions:  map[string]string{"test": "test"},
			},
			want:        `{"token":"this-is-the-token-from-github"}`,
			expErr:      "",
			handlerFunc: nil,
		},
		{
			name:      "missing_repositories",
			appID:     "test-app-id",
			installID: "test-install-id",
			request: &TokenRequest{
				Permissions: map[string]string{"test": "test"},
			},
			expErr: "requested repositories cannot be nil, did you mean to use AccessTokenAllRepos to request all repos?",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

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

			app, err := NewApp(tc.appID, tc.installID, rsaPrivateKey, WithAccessTokenURLPattern(fakeGitHub.URL+"/%s/access_tokens"))
			if err != nil {
				t.Fatal(err)
			}

			got, err := app.AccessToken(ctx, tc.request)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Errorf(diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateAppJWT(t *testing.T) {
	t.Parallel()

	base64Decode := func(tb testing.TB, i string) []byte {
		tb.Helper()

		b, err := base64.RawURLEncoding.DecodeString(i)
		if err != nil {
			tb.Fatal(err)
		}
		return b
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	token, err := generateAppJWT(key, "my-iss", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(string(token), ".")
	if exp := 3; len(parts) != exp {
		t.Fatalf("expected %d items, got %q", exp, parts)
	}

	header := base64Decode(t, parts[0])
	if got, want := string(header), `{"alg":"RS256","typ":"JWT"}`; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	body := base64Decode(t, parts[1])
	if got, want := string(body), `"iss":"my-iss"`; !strings.Contains(got, want) {
		t.Errorf("expected %q to contain %q", got, want)
	}

	signature := base64Decode(t, parts[2])

	h := sha256.New()
	h.Write([]byte(parts[0] + "." + parts[1]))
	digest := h.Sum(nil)

	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest, signature); err != nil {
		t.Fatal(err)
	}
}
