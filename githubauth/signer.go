// Copyright 2024 The Authors (see AUTHORS file)
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

package githubauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"hash/crc32"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Signer interface {
	Sign(data string) ([]byte, error)
}

type rsaSigner struct {
	privateKey *rsa.PrivateKey
}

func (s *rsaSigner) Sign(data string) ([]byte, error) {
	h := sha256.New()
	h.Write([]byte(data))
	digest := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(nil, s.privateKey, crypto.SHA256, digest)
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}
	return signature, nil
}

func NewRSASigner(privateKey *rsa.PrivateKey) *rsaSigner {
	return &rsaSigner{privateKey: privateKey}
}

type CloudKmsKey struct {
	Name string
}

type kmsSigner struct {
	key    CloudKmsKey
	client KeyManagementClient
}

// Adapted from the Golang example from https://cloud.google.com/kms/docs/create-validate-signatures
func (s *kmsSigner) Sign(data string) ([]byte, error) {
	ctx := context.Background()

	h := sha256.New()
	h.Write([]byte(data))
	digest := h.Sum(nil)

	crc32c := func(data []byte) uint32 {
		t := crc32.MakeTable(crc32.Castagnoli)
		return crc32.Checksum(data, t)
	}
	digestCRC32C := crc32c(digest)

	req := &kmspb.AsymmetricSignRequest{
		Name: s.key.Name,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest,
			},
		},
		DigestCrc32C: wrapperspb.Int64(int64(digestCRC32C)),
	}

	result, err := s.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}

	// Optional, but recommended: perform integrity verification on result.
	// For more details on ensuring E2E in-transit integrity to and from Cloud KMS visit:
	// https://cloud.google.com/kms/docs/data-integrity-guidelines
	if result.VerifiedDigestCrc32C == false {
		return nil, fmt.Errorf("AsymmetricSign: request corrupted in-transit")
	}
	if result.Name != req.Name {
		return nil, fmt.Errorf("AsymmetricSign: request corrupted in-transit")
	}
	if int64(crc32c(result.Signature)) != result.SignatureCrc32C.Value {
		return nil, fmt.Errorf("AsymmetricSign: response corrupted in-transit")
	}

	return result.Signature, nil
}

func NewCloudKmsSigner(ctx context.Context, key CloudKmsKey) (*kmsSigner, error) {
	client, err := NewCloudKmsClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating cloud kms client: %w", err)
	}
	return &kmsSigner{
		key:    key,
		client: client,
	}, nil
}
