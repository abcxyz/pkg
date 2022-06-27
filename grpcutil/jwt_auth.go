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

// Package grpcutil provides utilities for getting information from the grpc context.
package grpcutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/abcxyz/pkg/jwtutil"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	grpcmetadata "google.golang.org/grpc/metadata"
)

// JWTAuthenticationHandler allows for retrieving principal information from JWT tokens stored in GRPC metadata.
type JWTAuthenticationHandler struct {
	*jwtutil.Verifier
	// JWTPrefix is a prefix that occurs in a string before the signed JWT token.
	JWTPrefix string
	// JWTKey is the key in the GRPC metadata which holds the wanted JWT token.
	JWTKey string
	// PrincipalClaimKey is the key in the JWTs claims which corresponds to the user's email.
	PrincipalClaimKey string
	// In some cases (such as cloud run) the JWT's signature is redacted. In this case, we cannot validate
	// the jwt ourselves. This flag skips the validation, and instead trusts the JWT has been validated
	// upstream.
	ValidationDisabled bool
}

// NewJWTAuthenticationHandler returns a JWTAuthenticationHandler with a verifier initialized. Uses defaults
// for JWT related fields that will retreive a user email when using IAM on GCP.
func NewJWTAuthenticationHandler(ctx context.Context, endpoint string, disableValidation bool) (*JWTAuthenticationHandler, error) {
	verifier, err := jwtutil.NewVerifier(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return &JWTAuthenticationHandler{
		Verifier:           verifier,
		JWTPrefix:          "bearer ",
		JWTKey:             "authorization",
		PrincipalClaimKey:  "email",
		ValidationDisabled: disableValidation,
	}, nil
}

// RequestPrincipal extracts the JWT principal from the grpcmetadata in the context.
func (g *JWTAuthenticationHandler) RequestPrincipal(ctx context.Context) (string, error) {
	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("gRPC metadata in incoming context is missing")
	}

	vals := md.Get(g.JWTKey)
	if len(vals) == 0 {
		return "", fmt.Errorf("unable to find matching jwt in grpc metadata")
	}
	jwtRaw := vals[0]
	// We compare prefix case insensitively.
	if !strings.HasPrefix(strings.ToLower(jwtRaw), g.JWTPrefix) {
		return "", fmt.Errorf("expected prefix %s, but not found in jwt: %s", g.JWTPrefix, jwtRaw)
	}
	idToken := jwtRaw[len(g.JWTPrefix):]

	var token jwt.Token
	if g.ValidationDisabled {
		var err error
		if token, err = jwt.Parse([]byte(idToken), jwt.WithToken(openid.New()), jwt.WithVerify(false), jwt.WithValidate(false)); err != nil {
			return "", fmt.Errorf("failed to parse jwt: %w", err)
		}
	} else {
		var err error
		if token, err = g.ValidateJWT(idToken); err != nil {
			return "", fmt.Errorf("unable to validate jwt: %w", err)
		}
	}

	tokenMap, err := token.AsMap(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't convert token to map: %w", err)
	}

	// Retrieve the principal from claims.
	principalRaw, ok := tokenMap[g.PrincipalClaimKey]
	if !ok {
		return "", fmt.Errorf("jwt claims are missing the email key %q", g.PrincipalClaimKey)
	}
	principal, ok := principalRaw.(string)
	if !ok {
		return "", fmt.Errorf("expecting string in jwt claims %q, got %T", g.PrincipalClaimKey, principalRaw)
	}
	if principal == "" {
		return "", fmt.Errorf("nil principal under claims %q", g.PrincipalClaimKey)
	}

	return principal, nil
}
