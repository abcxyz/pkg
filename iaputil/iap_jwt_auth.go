// Copyright 2023 The Authors (see AUTHORS file)
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

package iaputil

import (
	"context"
	"fmt"

	"github.com/abcxyz/pkg/jwtutil"
)

// IAPJWTAuthenticationHandler allows for retrieving principal information from JWT tokens provided by
// Identity Aware Proxy (IAP).
type IAPJWTAuthenticationHandler struct {
	*jwtutil.Verifier
	// PrincipalClaimKey is the key in the JWTs claims which corresponds to the user's email.
	PrincipalClaimKey string
	// Endpoint is the endpoint where JWKs public keys can be found to do JWT validation.
	Endpoint string
}

// NewIAPJWTAuthenticationHandler returns a IAPJWTAuthenticationHandler with a verifier initialized. Uses defaults
// for JWT related fields that will retrieve a user email when using IAP on GCP.
func NewIAPJWTAuthenticationHandler(ctx context.Context) (*IAPJWTAuthenticationHandler, error) {
	j := &IAPJWTAuthenticationHandler{
		PrincipalClaimKey: "email",
		// For context: https://cloud.google.com/iap/docs/signed-headers-howto#verifying_the_jwt_header
		Endpoint: "https://www.gstatic.com/iap/verify/public_key-jwk",
	}

	verifier, err := jwtutil.NewVerifier(ctx, j.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create verifier: %w", err)
	}
	j.Verifier = verifier
	return j, nil
}

// option 1 standard approach

// RequestPrincipalFromIAP validates and extracts the JWT principal from the IAP JWT.
func (g *IAPJWTAuthenticationHandler) RequestPrincipal(ctx context.Context, iapJWT string) (string, error) {
	token, err := g.ValidateJWT(iapJWT)
	if err != nil {
		return "", fmt.Errorf("unable to validate jwt: %w", err)
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
