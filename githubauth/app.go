// Copyright 2024 The Authors (see AUTHORS file)
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

package githubauth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// defaultBaseURL is the default API base URL.
const defaultBaseURL = "https://api.github.com"

// App is an object that can be used to generate application level JWTs or to
// request an OIDC token on behalf of an installation.
type App struct {
	appID      string
	privateKey *rsa.PrivateKey

	baseURL    string
	httpClient *http.Client
}

// AppID returns the GitHub App's ID.
func (a *App) AppID() string { //nolint:stylecheck // "AppID" is the name GitHub uses
	return a.appID
}

// Option is a function that provides an option to the GitHub App creation.
type Option func(g *App) *App

// WithBaseURL allows overriding of the GitHub API url. This is usually only
// overidden for testing or private GitHub installations.
func WithBaseURL(url string) Option {
	return func(g *App) *App {
		g.baseURL = strings.TrimSuffix(url, "/")
		return g
	}
}

// WithHTTPClient is an option that allows a consumer to provider their own http
// client implementation. This HTTP client will be shared among all
// [AppInstallation].
func WithHTTPClient(client *http.Client) Option {
	return func(g *App) *App {
		g.httpClient = client
		return g
	}
}

// NewApp creates a new GitHub App from the given inputs.
//
// The privateKey can be the [*rsa.PrivateKey], or a PEM-encoded string (or
// []byte) of the private key material.
func NewApp[T *rsa.PrivateKey | string | []byte](appID string, privateKeyT T, opts ...Option) (*App, error) {
	var privateKey *rsa.PrivateKey
	var err error

	switch t := any(privateKeyT).(type) {
	case *rsa.PrivateKey:
		privateKey = t
	case string:
		privateKey, err = parseRSAPrivateKeyPEM([]byte(t))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key as a PEM-encoded string: %w", err)
		}
	case []byte:
		privateKey, err = parseRSAPrivateKeyPEM(t)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key as a PEM-encoded []byte: %w", err)
		}
	default:
		panic("impossible")
	}

	app := &App{
		appID:      appID,
		privateKey: privateKey,

		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	for _, opt := range opts {
		app = opt(app)
	}

	return app, nil
}

// TokenRequest is a struct that contains the list of repositories and the
// requested permissions / scopes that are requested when generating a new
// installation access token.
type TokenRequest struct {
	Repositories []string          `json:"repositories"`
	Permissions  map[string]string `json:"permissions"`
}

// TokenRequestAllRepos is a struct that contains the requested
// permissions/scopes that are requested when generating a new installation
// access token.
//
// This struct intentionally omits the repository properties to generate a token
// for all repositories granted to this GitHub app installation.
type TokenRequestAllRepos struct {
	Permissions map[string]string `json:"permissions"`
}

// AppToken creates a signed JWT to authenticate a GitHub app so that it can
// make API calls to GitHub.
func (g *App) AppToken() (string, error) {
	// Make the current time 30 seconds in the past to combat clock skew issues
	// where the JWT we issue looks like it is coming from the future when it gets
	// to GitHub
	iat := time.Now().UTC().Add(-30 * time.Second)
	exp := iat.Add(5 * time.Minute)

	b64Encode := base64.RawURLEncoding.EncodeToString

	headers := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9" // {"alg":"RS256", "typ":"JWT"}

	token, err := json.Marshal(map[string]any{
		"exp": exp.Unix(),
		"iat": iat.Unix(),
		"iss": g.appID,
	})
	if err != nil {
		return "", fmt.Errorf("error building JWT: %w", err)
	}

	unsigned := headers + "." + b64Encode(token)

	h := sha256.New()
	h.Write([]byte(unsigned))
	digest := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(nil, g.privateKey, crypto.SHA256, digest)
	if err != nil {
		return "", fmt.Errorf("error signing JWT: %w", err)
	}

	return unsigned + "." + b64Encode(signature), nil
}

// OAuthAppTokenSource adheres to the oauth2 TokenSource interface and returns a oauth2 token
// by creating a JWT token.
func (i *App) OAuthAppTokenSource() oauth2.TokenSource {
	return oauth2TokenSource(func() (*oauth2.Token, error) {
		jwt, err := i.AppToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate app token: %w", err)
		}

		return &oauth2.Token{
			AccessToken: jwt,
		}, nil
	})
}

// AppInstallation represents a specific installation of the app (on a repo,
// org, or user).
type AppInstallation struct {
	app            *App
	accessTokenURL string
}

// App returns the underlying app for this installation. This is a pointer back
// to the exact [App] that created the installation, meaning callers cannot
// assume exclusive ownership over the result.
func (i *AppInstallation) App() *App {
	return i.app
}

// InstallationForID returns an AccessTokensURLFunc that gets the access token
// url for the given installation.
func (a *App) InstallationForID(ctx context.Context, installationID string) (*AppInstallation, error) {
	u, err := a.accessTokenURL(ctx, fmt.Sprintf("%s/app/installations/%s", a.baseURL, installationID))
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for installation %s: %w", installationID, err)
	}

	return &AppInstallation{
		app:            a,
		accessTokenURL: u,
	}, nil
}

// InstallationForOrg returns an AccessTokensURLFunc that gets the access token url for the
// given org context.
func (a *App) InstallationForOrg(ctx context.Context, org string) (*AppInstallation, error) {
	u, err := a.accessTokenURL(ctx, fmt.Sprintf("%s/orgs/%s/installation", a.baseURL, org))
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for org %s: %w", org, err)
	}

	return &AppInstallation{
		app:            a,
		accessTokenURL: u,
	}, nil
}

// InstallationForRepo returns an AccessTokensURLFunc that gets the access token url for the
// given repo context.
func (a *App) InstallationForRepo(ctx context.Context, org, repo string) (*AppInstallation, error) {
	u, err := a.accessTokenURL(ctx, fmt.Sprintf("%s/repos/%s/%s/installation", a.baseURL, org, repo))
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for repo %s/%s: %w", org, repo, err)
	}

	return &AppInstallation{
		app:            a,
		accessTokenURL: u,
	}, nil
}

// InstallationForUser returns an AccessTokensURLFunc that gets the access token url for the
// given user context.
func (a *App) InstallationForUser(ctx context.Context, user string) (*AppInstallation, error) {
	u, err := a.accessTokenURL(ctx, fmt.Sprintf("%s/users/%s/installation", a.baseURL, user))
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for user %s: %w", user, err)
	}

	return &AppInstallation{
		app:            a,
		accessTokenURL: u,
	}, nil
}

// AccessToken calls the GitHub API to generate a new access token for this
// application installation with the requested permissions and repositories.
func (i *AppInstallation) AccessToken(ctx context.Context, request *TokenRequest) (string, error) {
	if request == nil || request.Repositories == nil {
		return "", fmt.Errorf("requested repositories cannot be nil, did you mean to use AccessTokenAllRepos to request all repos?")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling request data: %w", err)
	}

	return i.githubAccessToken(ctx, requestJSON)
}

// SelectedReposTokenSource returns a [TokenSource] that mints a GitHub token
// with permissions on the selected repos.
func (i *AppInstallation) SelectedReposTokenSource(permissions map[string]string, repos ...string) TokenSource {
	return TokenSourceFunc(func(ctx context.Context) (string, error) {
		token, err := i.AccessToken(ctx, &TokenRequest{
			Permissions:  permissions,
			Repositories: repos,
		})
		if err != nil {
			return "", fmt.Errorf("failed to get github access token for repos %q: %w", repos, err)
		}
		return token, nil
	})
}

// AccessTokenAllRepos calls the GitHub API to generate a new access token for
// this application installation with the requested permissions and all granted
// repositories.
func (i *AppInstallation) AccessTokenAllRepos(ctx context.Context, request *TokenRequestAllRepos) (string, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling request data: %w", err)
	}

	return i.githubAccessToken(ctx, requestJSON)
}

// AllReposTokenSource returns a [TokenSource] that mints a GitHub token with
// permissions on all repos.
func (i *AppInstallation) AllReposTokenSource(permissions map[string]string) TokenSource {
	return TokenSourceFunc(func(ctx context.Context) (string, error) {
		token, err := i.AccessTokenAllRepos(ctx, &TokenRequestAllRepos{
			Permissions: permissions,
		})
		if err != nil {
			return "", fmt.Errorf("failed to get github access token for all repos: %w", err)
		}
		return token, nil
	})
}

// accessTokenURL gets an access token for the given path (which might be an
// org, repo, or user). It uses the app's JWT to authenticate as a Bearer token.
func (a *App) accessTokenURL(ctx context.Context, u string) (string, error) {
	jwt, err := a.AppToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate github app jwt: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)

	res, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make http request: %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(io.LimitReader(res.Body, 4_194_304)) // 4 MiB
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if got, want := res.StatusCode, http.StatusOK; got != want {
		return "", fmt.Errorf("invalid http response status (expected %d to be %d): %s", got, want, string(b))
	}

	var resp struct {
		AccessTokensURL string `json:"access_tokens_url"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response as json: %w: %s", err, string(b))
	}
	return resp.AccessTokensURL, nil
}

// githubAccessToken calls the GitHub API to generate a new access token with
// provided JSON payload bytes.
func (i *AppInstallation) githubAccessToken(ctx context.Context, requestJSON []byte) (string, error) {
	appJWT, err := i.app.AppToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate github app jwt: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.accessTokenURL, bytes.NewReader(requestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", appJWT))

	res, err := i.app.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make http request: %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(io.LimitReader(res.Body, 4_194_304)) // 4 MiB
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if got, want := res.StatusCode, http.StatusCreated; got != want {
		return "", fmt.Errorf("invalid http response status (expected %d to be %d): %s", got, want, string(b))
	}

	// GitHub will respond with a 201 when you send a request for an invalid
	// combination, e.g. 'issues':'write' for an empty repository list. This 201
	// comes with a response that is not actually JSON. Attempt to parse the JSON
	// to see if this is a valid token, if it is not then respond with an error
	// containing the actual response from GitHub.
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response as json: %w: %s", err, string(b))
	}
	return resp.Token, nil
}

// parseRSAPrivateKeyPEM parses the input as a PEM-encoded RSA private key.
func parseRSAPrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse pem: no pem block found")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key pem: %w", err)
	}
	return key, nil
}
