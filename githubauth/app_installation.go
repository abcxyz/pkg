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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

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

// SelectedReposOAuth2TokenSource creates an [oauth2.TokenSource] which can be
// used in combination with [oauth2.NewClient] to create an authenticated HTTP
// client capable of being passed to the go-github library.
func (i *AppInstallation) SelectedReposOAuth2TokenSource(ctx context.Context, permissions map[string]string, repos ...string) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, oauth2TokenSource(func() (*oauth2.Token, error) {
		token, err := i.AccessToken(ctx, &TokenRequest{
			Permissions:  permissions,
			Repositories: repos,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get github access token for repos %q: %w", repos, err)
		}

		return &oauth2.Token{
			AccessToken: token,
			Expiry:      time.Now().Add(55 * time.Minute), // GitHub's expiration is 1 hour
		}, nil
	}))
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

// AllReposOAuth2TokenSource creates an [oauth2.TokenSource] which can be used
// in combination with [oauth2.NewClient] to create an authenticated HTTP client
// capable of being passed to the go-github library.
func (i *AppInstallation) AllReposOAuth2TokenSource(ctx context.Context, permissions map[string]string) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, oauth2TokenSource(func() (*oauth2.Token, error) {
		token, err := i.AccessTokenAllRepos(ctx, &TokenRequestAllRepos{
			Permissions: permissions,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get github access token for all repos: %w", err)
		}

		return &oauth2.Token{
			AccessToken: token,
			Expiry:      time.Now().Add(55 * time.Minute), // GitHub's expiration is 1 hour
		}, nil
	}))
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
