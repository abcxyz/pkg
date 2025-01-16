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
	"log"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

// Signer is an interface that exposes a single method that is used
// to sign a digest using a private key.
type Signer interface {
	Sign(ctx context.Context, digest []byte) ([]byte, error)
}

// PrivateKeySigner is a Signer implementation that uses a provided
// private key to sign the request.
type PrivateKeySigner struct {
	PrivateKey *rsa.PrivateKey
}

// NewPrivateKeySigner creates a new instance of the PrivateKeySigner
// with the provided private key.
func NewPrivateKeySigner(privateKey *rsa.PrivateKey) Signer {
	return &PrivateKeySigner{
		PrivateKey: privateKey,
	}
}

// Sign creates a signature for the provided digest using the private key.
func (s *PrivateKeySigner) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	signature, err := rsa.SignPKCS1v15(nil, s.PrivateKey, crypto.SHA256, digest)
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}
	return signature, nil
}

// KMSSigner is a Signer implementation that uses Google Cloud
// KMS to sign a request.
type KMSSigner struct {
	client *kms.KeyManagementClient
	keyID  string
}

// Sign implements Signer using Google Cloud KMS to produce a
// signature for the provided digest.
func (s *KMSSigner) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	req := &kmspb.AsymmetricSignRequest{
		Name: s.keyID,
		Data: digest,
	}
	resp, err := s.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error signing via kms key: %w", err)
	}
	return resp.GetSignature(), nil
}

// NewKMSSigner creates a new instance of a KMSSigner
// using the provided KMS key ID.
func NewKMSSigner(ctx context.Context, keyID string) Signer {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		log.Fatalf("failed to setup client: %v", err)
	}
	return &KMSSigner{
		client: client,
	}
}
