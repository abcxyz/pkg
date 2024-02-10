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
	"fmt"
	"strings"
)

var _ TokenSource = (*StaticTokenSource)(nil)

// StaticTokenSource is a GitHubToken provider that returns the provided token.
type StaticTokenSource struct {
	token string
}

// NewStaticTokenSource returns a [StaticTokenSource] which returns the token
// string as given.
func NewStaticTokenSource(token string) (*StaticTokenSource, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("provided token is empty")
	}

	return &StaticTokenSource{
		token: token,
	}, nil
}

// GitHubToken implements [TokenSource].
func (s *StaticTokenSource) GitHubToken(ctx context.Context) (string, error) {
	return s.token, nil
}
