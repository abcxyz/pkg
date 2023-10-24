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

// Package githubapp contains a class with methods for any service that needs
// to interact with GitHub as an app.
package githubapp

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/abcxyz/pkg/cache"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
)

// URL used to retrieve access tokens. The pattern must contain a single '%s' which represents where in the url
// to insert the installation id.
const defaultGitHubAccessTokenURLPattern = "https://api.github.com/app/installations/%s/access_tokens" //nolint

const cacheKey = "github-app-jwt"

// Config contains all of the required configuration for operating as a GitHub App.
// This includes the three major components, the App ID, the Install ID and the Private Key.
type Config struct {
	AppID                 string
	InstallationID        string
	PrivateKey            *rsa.PrivateKey
	accessTokenURLPattern string
	jwtTokenExpiration    time.Duration
	jwtCacheDuration      time.Duration
	client                *http.Client
}

// ConfigOption is a function type that applies a mechanism to set optional
// configuration values.
type ConfigOption func(f *Config)

// WithAccessTokenURLPattern allows overriding of the GitHub api url that is
// used when generating installation access tokens. The default is the primary
// GitHub api url which should only be overridden for private GitHub installations.
//
// The `pattern` parameter expects a single `%s` that represents the installation
// id that is provided with the rest of the configuration.
func WithAccessTokenURLPattern(pattern string) ConfigOption {
	return func(f *Config) {
		f.accessTokenURLPattern = pattern
	}
}

// WithJWTTokenExpiration is an option that allows overriding the default expiration
// date of the application JWTs.
func WithJWTTokenExpiration(exp time.Duration) ConfigOption {
	return func(f *Config) {
		f.jwtTokenExpiration = exp
	}
}

// WithJWTTokenCaching is an option that tells the GitHub app to cache its
// JWT App tokens. The amount of time that the tokens are cached is based
// on the provided `beforeExp` parameter + the configured token expiration.
// This results in a cache expiration of <token expiration> - <beforeExp>.
func WithJWTTokenCaching(beforeExp time.Duration) ConfigOption {
	return func(f *Config) {
		exp := f.jwtTokenExpiration
		f.jwtCacheDuration = exp - beforeExp
	}
}

// WithHTTPClient is an option that allows a consumer to provider their own
// http client implementation.
func WithHTTPClient(client *http.Client) ConfigOption {
	return func(f *Config) {
		f.client = client
	}
}

// NewConfig creates a new configuration object containing the three primary required
// configuration values. Options allow for the customization of rarely used configuration
// values. Options are evaluated in order from first to last.
func NewConfig(appID, installationID string, privateKey *rsa.PrivateKey, opts ...ConfigOption) *Config {
	config := Config{
		AppID:                 appID,
		InstallationID:        installationID,
		PrivateKey:            privateKey,
		accessTokenURLPattern: defaultGitHubAccessTokenURLPattern,
		jwtTokenExpiration:    9 * time.Minute,
		jwtCacheDuration:      0 * time.Nanosecond,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return &config
}

// TokenRequest is a struct that contains the list of repositories and the
// requested permissions / scopes that are requested when generating a
// new installation access token.
type TokenRequest struct {
	Repositories []string          `json:"repositories"`
	Permissions  map[string]string `json:"permissions"`
}

// TokenRequestAllRepos is a struct that contains the requested permissions/scopes
// that are requested when generating a new installation access token.
// This struct intentionally omits the repository properties to generate a token
// for all repositories granted to this GitHub app installation.
type TokenRequestAllRepos struct {
	Permissions map[string]string `json:"permissions"`
}

// GitHubApp is an object that can be used to generate application level JWTs
// or to request an OIDC token on behalf of an installation.
type GitHubApp struct {
	config   *Config
	jwtCache *cache.Cache[[]byte]
	client   *http.Client
}

// New creates a GitHubApp instance based on the provided
// GitHubAppConfig object.
func New(config *Config) *GitHubApp {
	app := GitHubApp{
		config: config,
	}
	if config.client != nil {
		app.client = config.client
	}
	if app.client == nil {
		app.client = &http.Client{Timeout: 10 * time.Second}
	}
	if config.jwtCacheDuration != 0 {
		app.jwtCache = cache.New[[]byte](config.jwtCacheDuration)
	}
	return &app
}

// AppToken creates a signed JWT to authenticate a GitHub app
// so that it can make API calls to GitHub.
func (g *GitHubApp) AppToken() ([]byte, error) {
	var token []byte
	// If token caching is enabled, look first in the cache
	if g.jwtCache != nil {
		// Check for a valid JWT in the cache
		signedJwt, ok := g.jwtCache.Lookup(cacheKey)
		if !ok {
			// Create a JWT for reading instance information from GitHub
			signedJwt, err := g.generateAppJWT()
			if err != nil {
				return nil, fmt.Errorf("error generating the JWT for GitHub app access: %w", err)
			}
			g.jwtCache.Set(cacheKey, signedJwt)
		}
		token = signedJwt
	}
	if token == nil {
		// Create a JWT for reading instance information from GitHub
		signedJwt, err := g.generateAppJWT()
		if err != nil {
			return nil, fmt.Errorf("error generating the JWT for GitHub app access: %w", err)
		}
		token = signedJwt
	}
	return token, nil
}

// generateAppJWT builds a signed JWT that can be used to
// communicate with GitHub as an application.
func (g *GitHubApp) generateAppJWT() ([]byte, error) {
	// Make the current time 30 seconds in the past to combat clock
	// skew issues where the JWT we issue looks like it is coming
	// from the future when it gets to GitHub
	iat := time.Now().Add(-30 * time.Second)
	exp := iat.Add(g.config.jwtTokenExpiration)
	iss := g.config.AppID

	token, err := jwt.NewBuilder().
		Expiration(exp).
		IssuedAt(iat).
		Issuer(iss).
		Build()
	if err != nil {
		return nil, fmt.Errorf("error building JWT: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, g.config.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}
	return signed, nil
}

// AccessToken calls the GitHub API to generate a new
// access token for this application installation with the requested
// permissions and repositories.
func (g *GitHubApp) AccessToken(ctx context.Context, request *TokenRequest) (string, error) {
	if request.Repositories == nil {
		return "", fmt.Errorf("requested repositories cannot be nil, did you mean to use AccessTokenAllRepos to request all repos?")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling request data: %w", err)
	}

	return g.githubAccessToken(ctx, requestJSON)
}

// AccessTokenAllRepos calls the GitHub API to generate a new
// access token for this application installation with the requested
// permissions and all granted repositories.
func (g *GitHubApp) AccessTokenAllRepos(ctx context.Context, request *TokenRequestAllRepos) (string, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling request data: %w", err)
	}

	return g.githubAccessToken(ctx, requestJSON)
}

// githubAccessToken calls the GitHub API to generate a new
// access token with provided JSON payload bytes.
func (g *GitHubApp) githubAccessToken(ctx context.Context, requestJSON []byte) (string, error) {
	appJWT, err := g.AppToken()
	if err != nil {
		return "", fmt.Errorf("error generating app jwt: %w", err)
	}
	requestURL := fmt.Sprintf(g.config.accessTokenURLPattern, g.config.InstallationID)

	requestReader := bytes.NewReader(requestJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, requestReader)
	if err != nil {
		return "", fmt.Errorf("error creating http request for GitHub installation information: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", appJWT))

	res, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making http request for GitHub installation access token %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(io.LimitReader(res.Body, 64_000))
	if err != nil {
		return "", fmt.Errorf("error reading http response for GitHub installation access token %w", err)
	}

	if res.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to retrieve token from GitHub - Status: %s - Body: %s", res.Status, string(b))
	}

	// GitHub will respond with a 201 when you send a request for an invalid combination,
	// e.g. 'issues':'write' for an empty repository list. This 201 comes with a response that is not actually JSON.
	// Attempt to parse the JSON to see if this is a valid token, if it is not then respond with an error containing the
	// actual response from GitHub.
	tokenContent := map[string]any{}
	if err := json.Unmarshal(b, &tokenContent); err != nil {
		return "", fmt.Errorf("invalid access token from GitHub - Body: %s", string(b))
	}
	return string(b), nil
}

// Token adheres to the oauth2 TokenSource interface and returns a oauth2 token by
// creating a JWT token.
func (g *GitHubApp) Token() (*oauth2.Token, error) {
	jwt, err := g.AppToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate app token: %w", err)
	}
	return &oauth2.Token{AccessToken: string(jwt)}, nil
}
