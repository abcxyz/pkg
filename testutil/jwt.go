package testutil

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestCreateToken(tb testing.TB, id string, email string) jwt.Token {
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

func TestSignToken(tb testing.TB, tok jwt.Token, privateKey *ecdsa.PrivateKey, keyID string) string {
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
