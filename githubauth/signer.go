// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package githubauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"fmt"
	"io"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
)

// NewPrivateKeySigner utilizes a private key that is provided
// directly to sign requests. The private key can be an actual rsa.Private
// or a string / byte[] representation of the key.
func NewPrivateKeySigner[T *rsa.PrivateKey | string | []byte](privateKeyT T) (crypto.Signer, error) {
	var privateKey *rsa.PrivateKey
	var err error

	switch t := any(privateKeyT).(type) {
	case *rsa.PrivateKey:
		privateKey = t
	case string:
		privateKey, err = parseRSAPrivateKeyPEM([]byte(t))
	case []byte:
		privateKey, err = parseRSAPrivateKeyPEM(t)
	default:
		panic("impossible")
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}
	return &privateKeySigner{
		privateKey: privateKey,
	}, nil
}

// privateKeySigner is a Signer implementation that uses a provided
// private key to sign the request.
type privateKeySigner struct {
	privateKey *rsa.PrivateKey
}

func (s *privateKeySigner) Public() crypto.PublicKey {
	return s.privateKey.Public()
}

// Sign creates a signature for the provided digest using the private key.
func (s *privateKeySigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	signature, err := rsa.SignPKCS1v15(nil, s.privateKey, opts.HashFunc(), digest)
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}
	return signature, nil
}

// NewGoogleCloudKMSSigner creates a new instance of a crypto.Signer
// using the provided KMS key ID. Signature creation is done via Google Cloud
// KMS using the provided key to sign the request. The keyID is in the
// format "projects/*/locations/*/keyRings/*/cryptoKeys/*.".
func NewGoogleCloudKMSSigner(ctx context.Context, keyID string) (crypto.Signer, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup client: %w", err)
	}
	signer, err := gcpkms.NewSigner(ctx, client, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}
	return signer, nil
}
