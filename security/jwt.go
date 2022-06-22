// Copyright 2022 Google LLC
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

// Package security provides mechanisms for interacting with JWTs and getting authentication information.
package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	grpcmetadata "google.golang.org/grpc/metadata"
)

const (
	IAMKeyEndpoint = "https://www.googleapis.com/oauth2/v3/certs"
	jwtKey         = "authorization"
	jwtPrefix      = "bearer "
	emailKey       = "email"
)

// JWTVerifier allows for getting JWK keys from the JVS and validating JWTs with
// those keys.
type JWTVerifier struct {
	keys jwk.Set
}

// NewJWTVerifier returns a JWTVerifier with the cache initialized.
func NewJWTVerifier(ctx context.Context, endpoint string) (*JWTVerifier, error) {
	c := jwk.NewCache(ctx)
	if err := c.Register(endpoint); err != nil {
		return nil, fmt.Errorf("failed to register: %w", err)
	}

	// check that cache is correctly set up and certs are available
	if _, err := c.Refresh(ctx, endpoint); err != nil {
		return nil, fmt.Errorf("failed to retrieve JVS public keys: %w", err)
	}

	cached := jwk.NewCachedSet(c, endpoint)

	return &JWTVerifier{
		keys: cached,
	}, nil
}

// ValidateJWT takes a jwt string, converts it to a jwt.Token, and validates the signature.
func (j *JWTVerifier) ValidateJWT(jwtStr string) (*jwt.Token, error) {
	verifiedToken, err := jwt.Parse([]byte(jwtStr), jwt.WithKeySet(j.keys, jws.WithInferAlgorithmFromKey(true)))
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt %s: %w", jwtStr, err)
	}

	return &verifiedToken, nil
}

type GRPCAuthenticationHandler struct {
	*JWTVerifier
}

// NewGRPCAuthenticationHandler returns a GRPCAuthenticationHandler with a verifier initialized.
func NewGRPCAuthenticationHandler(ctx context.Context, endpoint string) (*GRPCAuthenticationHandler, error) {
	verifier, err := NewJWTVerifier(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return &GRPCAuthenticationHandler{verifier}, nil
}

// RequestPrincipalFromGRPC extracts the JWT principal from the grpcmetadata in the context.
func (g *GRPCAuthenticationHandler) RequestPrincipalFromGRPC(ctx context.Context) (string, error) {
	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("gRPC metadata in incoming context is missing")
	}

	vals := md.Get(jwtKey)
	if len(vals) == 0 {
		return "", fmt.Errorf("unable to find matching jwt in grpc metadata")
	}
	jwtRaw := vals[0]
	// We compare prefix case insensitively.
	if !strings.HasPrefix(strings.ToLower(jwtRaw), jwtPrefix) {
		return "", fmt.Errorf("expected prefix %s, but not found in jwt: %s", jwtPrefix, jwtRaw)
	}
	idToken := jwtRaw[len(jwtPrefix):]

	validatedToken, err := g.ValidateJWT(idToken)
	if err != nil {
		return "", fmt.Errorf("unable to validate jwt: %s", err)
	}

	tokenMap, err := (*validatedToken).AsMap(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't convert token to map: %s", err)
	}

	// Retrieve the principal from claims.
	principalRaw, ok := tokenMap[emailKey]
	if !ok {
		return "", fmt.Errorf("jwt claims are missing the email key %q", emailKey)
	}
	principal, ok := principalRaw.(string)
	if !ok {
		return "", fmt.Errorf("expecting string in jwt claims %q, got %T", emailKey, principalRaw)
	}
	if principal == "" {
		return "", fmt.Errorf("nil principal under claims %q", emailKey)
	}

	return principal, nil
}
