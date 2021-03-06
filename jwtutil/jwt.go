// Copyright 2022 The Authors (see AUTHORS file)
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

// Package jwtutil provides mechanisms for interacting with JWTs.
package jwtutil

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const IAMKeyEndpoint = "https://www.googleapis.com/oauth2/v3/certs"

// Verifier allows for getting public JWK keys from an endpoint and validating JWTs with
// those keys.
type Verifier struct {
	keys jwk.Set
}

// NewVerifier returns a Verifier with the cache initialized. The cache is set up using defaults, and refreshes happen every 15 minutes.
func NewVerifier(ctx context.Context, endpoint string) (*Verifier, error) {
	c := jwk.NewCache(ctx)
	if err := c.Register(endpoint); err != nil {
		return nil, fmt.Errorf("failed to register: %w", err)
	}

	// check that cache is correctly set up and certs are available
	if _, err := c.Refresh(ctx, endpoint); err != nil {
		return nil, fmt.Errorf("failed to retrieve public keys: %w", err)
	}

	cached := jwk.NewCachedSet(c, endpoint)

	return &Verifier{
		keys: cached,
	}, nil
}

// ValidateJWT takes a jwt string, converts it to a jwt.Token, and validates the signature against the public keys in the JWKS endpoint.
func (j *Verifier) ValidateJWT(jwtStr string) (jwt.Token, error) {
	verifiedToken, err := jwt.Parse([]byte(jwtStr), jwt.WithKeySet(j.keys, jws.WithInferAlgorithmFromKey(true)))
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt %s: %w", jwtStr, err)
	}

	return verifiedToken, nil
}
