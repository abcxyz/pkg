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
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// defaultBaseURL is the default API base URL.
const defaultBaseURL = "https://api.github.com"

// App is an object that can be used to generate application level JWTs or to
// request an OIDC token on behalf of an installation.
type App struct {
	appID  string
	signer crypto.Signer

	installationCache     map[string](func() (*AppInstallation, error))
	installationCacheLock sync.Mutex

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
// The signer can be any crypto.Signer. For RSA keys or PEM-encoded strings of
// RSA keys, use [NewPrivateKeySigner]. For Google Cloud KMS, use a signer
// implementation like [github.com/sethvargo/go-gcpkms/pkg/gcpkms.NewSigner]:
//
//	client, err := kms.NewKeyManagementClient(ctx)
//	if err != nil {
//		return nil, fmt.Errorf("failed to setup client: %w", err)
//	}
//	signer, err := gcpkms.NewSigner(ctx, client, keyID)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create signer: %w", err)
//	}
//	return signer, nil
func NewApp(appID string, signer crypto.Signer, opts ...Option) (*App, error) {
	app := &App{
		appID:             appID,
		installationCache: make(map[string](func() (*AppInstallation, error)), 8),
		signer:            signer,

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

	if g.signer != nil {
		signature, err := g.signer.Sign(nil, digest, crypto.SHA256)
		if err != nil {
			return "", fmt.Errorf("error signing JWT: %w", err)
		}
		return unsigned + "." + b64Encode(signature), nil
	}
	return unsigned, nil
}

// OAuthAppTokenSource adheres to the oauth2 TokenSource interface and returns a oauth2 token
// by creating a JWT token.
func (a *App) OAuthAppTokenSource() oauth2.TokenSource {
	return oauth2TokenSource(func() (*oauth2.Token, error) {
		jwt, err := a.AppToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate app token: %w", err)
		}

		return &oauth2.Token{
			AccessToken: jwt,
		}, nil
	})
}

// InstallationForID returns an AccessTokensURLFunc that gets the access token
// url for the given installation.
//
// The initial invocation will make an API call to GitHub to get the access
// token URL for the installation; future calls will return the cached
// installation.
func (a *App) InstallationForID(ctx context.Context, installationID string) (*AppInstallation, error) {
	i, err := a.withInstallationCaching(ctx, "i:"+installationID, "/app/installations/"+installationID)()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for installation %s: %w", installationID, err)
	}
	return i, nil
}

// InstallationForOrg returns an AccessTokensURLFunc that gets the access token url for the
// given org context.
//
// The initial invocation will make an API call to GitHub to get the access
// token URL for the installation; future calls will return the cached
// installation.
func (a *App) InstallationForOrg(ctx context.Context, org string) (*AppInstallation, error) {
	i, err := a.withInstallationCaching(ctx, "org:"+org, "/orgs/"+org+"/installation")()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for org %s: %w", org, err)
	}
	return i, nil
}

// InstallationForRepo returns an AccessTokensURLFunc that gets the access token url for the
// given repo context.
//
// The initial invocation will make an API call to GitHub to get the access
// token URL for the installation; future calls will return the cached
// installation.
func (a *App) InstallationForRepo(ctx context.Context, org, repo string) (*AppInstallation, error) {
	i, err := a.withInstallationCaching(ctx, "repo:"+org+"/"+repo, "/repos/"+org+"/"+repo+"/installation")()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for repo %s/%s: %w", org, repo, err)
	}
	return i, nil
}

// InstallationForUser returns an AccessTokensURLFunc that gets the access token url for the
// given user context.
//
// The initial invocation will make an API call to GitHub to get the access
// token URL for the installation; future calls will return the cached
// installation.
func (a *App) InstallationForUser(ctx context.Context, user string) (*AppInstallation, error) {
	i, err := a.withInstallationCaching(ctx, "user:"+user, "/users/"+user+"/installation")()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token url for user %s: %w", user, err)
	}
	return i, nil
}

// withInstallationCaching returns a closure that caches the app installation by
// key.
func (a *App) withInstallationCaching(ctx context.Context, cacheKey, tokenPath string) func() (*AppInstallation, error) {
	a.installationCacheLock.Lock()
	defer a.installationCacheLock.Unlock()

	entry, ok := a.installationCache[cacheKey]
	if !ok {
		entry = sync.OnceValues(func() (*AppInstallation, error) {
			u, err := a.accessTokenURL(ctx, a.baseURL+tokenPath)
			if err != nil {
				return nil, err
			}

			return &AppInstallation{
				app:            a,
				accessTokenURL: u,
			}, nil
		})
	}
	return entry
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
