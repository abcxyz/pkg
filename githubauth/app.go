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
	"time"

	"golang.org/x/oauth2"

	"github.com/abcxyz/pkg/cache"
)

// URL used to retrieve access tokens. The pattern must contain a single '%s'
// which represents where in the url to insert the installation id.
const defaultGitHubAccessTokenURLPattern = "https://api.github.com/app/installations/%s/access_tokens" //nolint

const cacheKey = "github-app-jwt"

// App is an object that can be used to generate application level JWTs or to
// request an OIDC token on behalf of an installation.
type App struct {
	AppID          string
	InstallationID string
	PrivateKey     *rsa.PrivateKey

	accessTokenURLPattern string

	jwtTokenExpiration time.Duration
	jwtCacheDuration   time.Duration
	jwtCache           *cache.Cache[[]byte]

	httpClient *http.Client
}

// Option is a function that provides an option to the GitHub App creation.
type Option func(g *App) *App

// WithAccessTokenURLPattern allows overriding of the GitHub api url that is
// used when generating installation access tokens. The default is the primary
// GitHub api url which should only be overridden for private GitHub
// installations.
//
// The `pattern` parameter expects a single `%s` that represents the
// installation id that is provided with the rest of the configuration.
func WithAccessTokenURLPattern(pattern string) Option {
	return func(g *App) *App {
		g.accessTokenURLPattern = pattern
		return g
	}
}

// WithJWTTokenExpiration is an option that allows overriding the default
// expiration date of the application JWTs.
func WithJWTTokenExpiration(exp time.Duration) Option {
	return func(g *App) *App {
		g.jwtTokenExpiration = exp
		return g
	}
}

// WithJWTTokenCaching is an option that tells the GitHub app to cache its JWT
// App tokens. The amount of time that the tokens are cached is based on the
// provided `beforeExp` parameter + the configured token expiration. This
// results in a cache expiration of <token expiration> - <beforeExp>.
func WithJWTTokenCaching(beforeExp time.Duration) Option {
	return func(g *App) *App {
		exp := g.jwtTokenExpiration
		g.jwtCacheDuration = exp - beforeExp
		return g
	}
}

// WithHTTPClient is an option that allows a consumer to provider their own http
// client implementation.
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
func NewApp[T *rsa.PrivateKey | string | []byte](appID, installationID string, privateKeyT T, opts ...Option) (*App, error) {
	var privateKey *rsa.PrivateKey
	var err error

	switch t := any(privateKeyT).(type) {
	case nil:
		return nil, fmt.Errorf("missing private key")
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
		AppID:          appID,
		InstallationID: installationID,
		PrivateKey:     privateKey,

		accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,

		jwtTokenExpiration: 9 * time.Minute,
		jwtCacheDuration:   0 * time.Nanosecond,

		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	for _, opt := range opts {
		app = opt(app)
	}

	// Do this last, since it depends on the cache duration.
	if app.jwtCacheDuration != 0 {
		app.jwtCache = cache.New[[]byte](app.jwtCacheDuration)
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
func (g *App) AppToken() ([]byte, error) {
	// If token caching is enabled, look first in the cache
	if g.jwtCache != nil {
		token, ok := g.jwtCache.Lookup(cacheKey)
		if ok {
			return token, nil
		}
	}

	token, err := generateAppJWT(g.PrivateKey, g.AppID, g.jwtTokenExpiration)
	if err != nil {
		return nil, fmt.Errorf("error generating the JWT for GitHub app access: %w", err)
	}

	if g.jwtCache != nil {
		g.jwtCache.Set(cacheKey, token)
	}

	return token, nil
}

// OAuthAppTokenSource adheres to the oauth2 TokenSource interface and returns a oauth2 token
// by creating a JWT token.
func (g *App) OAuthAppTokenSource() oauth2.TokenSource {
	return oauth2TokenSource(func() (*oauth2.Token, error) {
		jwt, err := g.AppToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate app token: %w", err)
		}

		return &oauth2.Token{
			AccessToken: string(jwt),
		}, nil
	})
}

// AccessToken calls the GitHub API to generate a new access token for this
// application installation with the requested permissions and repositories.
func (g *App) AccessToken(ctx context.Context, request *TokenRequest) (string, error) {
	if request == nil || request.Repositories == nil {
		return "", fmt.Errorf("requested repositories cannot be nil, did you mean to use AccessTokenAllRepos to request all repos?")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling request data: %w", err)
	}

	return g.githubAccessToken(ctx, requestJSON)
}

// SelectedReposTokenSource returns a [TokenSource] that mints a GitHub token
// with permissions on the selected repos.
func (g *App) SelectedReposTokenSource(permissions map[string]string, repos ...string) TokenSource {
	return TokenSourceFunc(func(ctx context.Context) (string, error) {
		token, err := g.AccessToken(ctx, &TokenRequest{
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
func (g *App) AccessTokenAllRepos(ctx context.Context, request *TokenRequestAllRepos) (string, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling request data: %w", err)
	}

	return g.githubAccessToken(ctx, requestJSON)
}

// AllReposTokenSource returns a [TokenSource] that mints a GitHub token with
// permissions on all repos.
func (g *App) AllReposTokenSource(permissions map[string]string) TokenSource {
	return TokenSourceFunc(func(ctx context.Context) (string, error) {
		token, err := g.AccessTokenAllRepos(ctx, &TokenRequestAllRepos{
			Permissions: permissions,
		})
		if err != nil {
			return "", fmt.Errorf("failed to get github access token for all repos: %w", err)
		}
		return token, nil
	})
}

// githubAccessToken calls the GitHub API to generate a new access token with
// provided JSON payload bytes.
func (g *App) githubAccessToken(ctx context.Context, requestJSON []byte) (string, error) {
	appJWT, err := g.AppToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate github app jws: %w", err)
	}
	requestURL := fmt.Sprintf(g.accessTokenURLPattern, g.InstallationID)

	requestReader := bytes.NewReader(requestJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, requestReader)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", appJWT))

	res, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make http request: %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(io.LimitReader(res.Body, 64_000))
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

// generateAppJWT builds a signed JWT that can be used to communicate with
// GitHub as an application.
func generateAppJWT(privateKey *rsa.PrivateKey, iss string, ttl time.Duration) ([]byte, error) {
	// Make the current time 30 seconds in the past to combat clock skew issues
	// where the JWT we issue looks like it is coming from the future when it gets
	// to GitHub
	iat := time.Now().Add(-30 * time.Second)
	exp := iat.Add(ttl)

	b64Encode := base64.RawURLEncoding.EncodeToString

	headers := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9" // {"alg":"RS256", "typ":"JWT"}

	token, err := json.Marshal(map[string]any{
		"exp": exp.Unix(),
		"iat": iat.Unix(),
		"iss": iss,
	})
	if err != nil {
		return nil, fmt.Errorf("error building JWT: %w", err)
	}

	unsigned := headers + "." + b64Encode(token)

	h := sha256.New()
	h.Write([]byte(unsigned))
	digest := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, digest)
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}

	return []byte(unsigned + "." + b64Encode(signature)), nil
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
