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

package testutil

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func CreateJWT(tb testing.TB, id string, email string) jwt.Token {
	tb.Helper()

	tok, err := jwt.NewBuilder().
		Audience([]string{"test_aud"}).
		Expiration(time.Now().UTC().Add(5 * time.Minute)).
		JwtID(id).
		IssuedAt(time.Now().UTC()).
		Issuer(`test_iss`).
		NotBefore(time.Now().UTC()).
		Subject("test_sub").
		Build()
	if err != nil {
		tb.Fatalf("failed to build token: %s\n", err)
	}
	if err := tok.Set("email", email); err != nil {
		tb.Fatal(err)
	}
	return tok
}

func SignToken(tb testing.TB, tok jwt.Token, privateKey *ecdsa.PrivateKey, keyID string) string {
	tb.Helper()

	hdrs := jws.NewHeaders()
	if err := hdrs.Set(jws.KeyIDKey, keyID); err != nil {
		tb.Fatal(err)
	}

	valid, err := jwt.Sign(tok, jwt.WithKey(jwa.ES256, privateKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		tb.Fatalf("failed to sign token: %s\n", err)
	}
	return string(valid)
}
