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

// Package githubauth provides interfaces and implementations for authenticating
// to GitHub.
package githubauth

import (
	"context"

	"golang.org/x/oauth2"
)

// TokenSource is an interface which returns a GitHub token.
type TokenSource interface {
	// GitHubToken returns a GitHub token, or any error that occurs.
	GitHubToken(ctx context.Context) (string, error)
}

// TokenSourceFunc is a function that implements [TokenSource].
type TokenSourceFunc func(ctx context.Context) (string, error)

func (f TokenSourceFunc) GitHubToken(ctx context.Context) (string, error) {
	return f(ctx)
}

// oauth2TokenSource is a wrapper function for making an oauth2 token source.
type oauth2TokenSource func() (*oauth2.Token, error)

func (f oauth2TokenSource) Token() (*oauth2.Token, error) {
	return f()
}
