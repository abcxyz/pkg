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

// Package security describes the authentication technology that the
// middleware investigates to autofill the principal in a log request.
package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
	grpcmetadata "google.golang.org/grpc/metadata"
)

// Key in a JWT's `claims` where we expect the principal.
const emailKey = "email"

// GRPCContext is an interface that retrieves the principal
// from a gRPC security context. A gRPC security context describes
// the technology used to authenticate a principal (e.g. JWT).
type GRPCContext interface {
	RequestPrincipal(context.Context) (string, error)
}

// JWKs provides JWKs to validate a JWT.
type JWKs struct {
	// Endpoint is the endpoint to retrieve the JWKs to validate JWT.
	Endpoint string `yaml:"endpoint,omitempty"`
}

// JWTRule provides info for how to retrieve security context from
// a raw JWT.
type JWTRule struct {
	// Key is the metadata key whose value is a JWT.
	Key string `yaml:"key,omitempty"`
	// Prefix is the prefix to truncate the metadata value
	// to retrieve the JWT.
	Prefix string `yaml:"prefix,omitempty"`
	// JWKs specifies the JWKs to validate the JWT.
	// If JWTs is nil, the JWT won't be validated.
	JWKs *JWKs `yaml:"jwks,omitempty"`
}

// Validate validates the FromRawJWT.
func (j *JWTRule) Validate() error {
	if j.Key == "" {
		return fmt.Errorf("key must be specified")
	}
	return nil
}

// JWTRules is a list of JWTRules used to retrieve security context from raw JWTs.
type JWTRules struct {
	Rules []*JWTRule `yaml:"jwt_rules,omitempty"`
}

// RequestPrincipal extracts the JWT principal from the grpcmetadata in the context.
// TODO: This method does not verify the JWT #20.
func (j *JWTRules) RequestPrincipal(ctx context.Context) (string, error) {
	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("gRPC metadata in incoming context is missing")
	}

	idToken, err := j.findJWT(md)
	if err != nil {
		return "", err
	}

	// Parse the JWT into claims.
	p := &jwt.Parser{}
	claims := jwt.MapClaims{}
	if _, _, err := p.ParseUnverified(idToken, claims); err != nil {
		return "", fmt.Errorf("unable to parse JWT: %w", err)
	}

	// Retrieve the principal from claims.
	principalRaw, ok := claims[emailKey]
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

// findJWT looks for a JWT from the gRPC metadata that matches the rules.
func (j *JWTRules) findJWT(md grpcmetadata.MD) (string, error) {
	for _, fj := range j.Rules {
		// Keys in grpc metadata are all lowercases.
		vals := md.Get(fj.Key)
		if len(vals) == 0 {
			continue
		}
		jwtRaw := vals[0]
		// We compare prefix case insensitively.
		if !strings.HasPrefix(strings.ToLower(jwtRaw), strings.ToLower(fj.Prefix)) {
			continue
		}
		idToken := jwtRaw[len(fj.Prefix):]
		return idToken, nil
	}

	return "", fmt.Errorf("no JWT found matching rules: %#v", j.Rules)
}
