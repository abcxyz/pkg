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

package grpcutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc/metadata"
)

func TestRequestPrincipalFromGRPC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// create another key, to show the correct key is retrieved from cache and used for validation.
	privateKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
	keyID := key + "/cryptoKeyVersions/[VERSION]-0"
	keyID2 := key + "/cryptoKeyVersions/[VERSION]-1"

	ecdsaKey, err := jwk.FromRaw(privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ecdsaKey.Set(jwk.KeyIDKey, keyID); err != nil {
		t.Fatal(err)
	}
	ecdsaKey2, err := jwk.FromRaw(privateKey2.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ecdsaKey2.Set(jwk.KeyIDKey, keyID2); err != nil {
		t.Fatal(err)
	}
	jwks := make(map[string][]jwk.Key)
	jwks["keys"] = []jwk.Key{ecdsaKey, ecdsaKey2}

	j, err := json.MarshalIndent(jwks, "", " ")
	if err != nil {
		t.Fatal("couldn't create jwks json")
	}

	path := "/.well-known/jwks"
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", j)
	})

	svr := httptest.NewServer(mux)

	t.Cleanup(func() {
		svr.Close()
	})

	tok := testutil.CreateJWT(t, "test_id", "user@example.com")
	validJWT := testutil.SignToken(t, tok, privateKey, keyID)

	tok2 := testutil.CreateJWT(t, "test_id_2", "me@example.com")
	validJWT2 := testutil.SignToken(t, tok2, privateKey2, keyID2)

	unsig, err := jwt.NewSerializer().Serialize(tok)
	if err != nil {
		t.Fatal("Couldn't get signing string.")
	}
	unsignedJWT := string(unsig)

	split := strings.Split(validJWT2, ".")
	sig := split[len(split)-1]

	invalidSignatureJWT := unsignedJWT + sig // signature from a different JWT

	g, err := NewJWTAuthenticationHandler(ctx, svr.URL+path)
	if err != nil {
		t.Fatal(fmt.Printf("couldn't create grpc authentication handler: %s", err))
	}

	tests := []struct {
		name          string
		ctx           context.Context //nolint:containedctx // Only for testing
		want          string
		wantErrSubstr string
	}{
		{
			name: "valid_jwt",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "bearer " + validJWT,
			})),
			want: "user@example.com",
		},
		{
			name: "different_case_jwt",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + validJWT,
			})),
			want: "user@example.com",
		},
		{
			name: "different_key",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + validJWT2,
			})),
			want: "me@example.com",
		},
		{
			name: "invalid",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + invalidSignatureJWT,
			})),
			wantErrSubstr: "failed to verify jwt",
		},
		{
			name: "unsigned",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": "Bearer " + unsignedJWT,
			})),
			wantErrSubstr: "failed to verify jwt",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := g.RequestPrincipalFromGRPC(tc.ctx)
			if diff := testutil.DiffErrString(err, tc.wantErrSubstr); diff != "" {
				t.Errorf("j.RequestPrincipal()) got unexpected error substring: %v", diff)
			}

			if got != tc.want {
				t.Errorf("j.RequestPrincipal() = %v, want %v", got, tc.want)
			}
		})
	}
}
