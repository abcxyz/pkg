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
	"testing"

	"github.com/golang-jwt/jwt"
)

// JWTFromClaims is a testing helper that builds a JWT from the
// given claims.
func JWTFromClaims(tb testing.TB, claims map[string]interface{}) string {
	tb.Helper()

	var jwtMapClaims jwt.MapClaims = claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtMapClaims)
	signedToken, err := token.SignedString([]byte("secureSecretText"))
	if err != nil {
		tb.Fatal(err)
	}
	return signedToken
}
