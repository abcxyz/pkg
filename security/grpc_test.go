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
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"google.golang.org/grpc/metadata"
)

func TestFromRawJWT_RequestPrincipal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ctx           context.Context //nolint:containedctx // Only for testing
		jwtRule       []*JWTRule
		want          string
		wantErrSubstr string
	}{
		{
			name: "valid_jwt",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + testutil.JWTFromClaims(t, map[string]interface{}{
					"email": "user@example.com",
				}),
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "Bearer ",
			}},
			want: "user@example.com",
		},
		{
			name: "valid_jwt_with_capitalized_config",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "bearer " + testutil.JWTFromClaims(t, map[string]interface{}{
					"email": "user@example.com",
				}),
			})),
			jwtRule: []*JWTRule{{
				Key:    "Authorization",
				Prefix: "Bearer ",
			}},
			want: "user@example.com",
		},
		{
			name: "multi_jwts",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"x-jwt-assertion": testutil.JWTFromClaims(t, map[string]interface{}{
					"email": "user@example.com",
				}),
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "Bearer ",
			}, {
				Key: "x-jwt-assertion",
			}},
			want: "user@example.com",
		},
		{
			name: "error_from_missing_jwt_email_claim",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + testutil.JWTFromClaims(t, map[string]interface{}{}),
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "Bearer ",
			}},
			wantErrSubstr: `jwt claims are missing the email key "email"`,
		},
		{
			name: "error_from_slice_as_jwt_email_claim",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + testutil.JWTFromClaims(t, map[string]interface{}{
					"email": []string{"foo", "bar"},
				}),
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "Bearer ",
			}},
			wantErrSubstr: `expecting string in jwt claims "email", got []interface {}`,
		},
		{
			name:          "error_from_missing_grpc_metadata",
			ctx:           context.Background(),
			wantErrSubstr: "gRPC metadata in incoming context is missing",
		},
		{
			name: "error_from_inexistent_jwt_key",
			ctx:  metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "Bearer ",
			}},
			wantErrSubstr: `no JWT found matching rules`,
		},
		{
			name: "error_from_prefix_longer_than_jwt",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "short",
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "loooooong",
			}},
			wantErrSubstr: `no JWT found matching rules`,
		},
		{
			name: "error_from_empty_string_as_jwt",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "",
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "",
			}},
			wantErrSubstr: `unable to parse JWT: token contains an invalid number of segments`,
		},
		{
			name: "error_from_unparsable_jwt",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "bananas",
			})),
			jwtRule: []*JWTRule{{
				Key:    "authorization",
				Prefix: "",
			}},
			wantErrSubstr: "unable to parse JWT",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			j := &JWTRules{Rules: tc.jwtRule}
			got, err := j.RequestPrincipal(tc.ctx)
			if diff := testutil.DiffErrString(err, tc.wantErrSubstr); diff != "" {
				t.Errorf("j.RequestPrincipal()) got unexpected error substring: %v", diff)
			}

			if got != tc.want {
				t.Errorf("j.RequestPrincipal() = %v, want %v", got, tc.want)
			}
		})
	}
}
