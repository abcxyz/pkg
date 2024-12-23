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

type Signer interface {
	Sign(ctx context.Context, digest []byte) ([]byte, error)
}

type PrivateKeySigner struct {
	privateKey *rsa.PrivateKey
}

func NewPrivateKeySigner(privateKey *rsa.PrivateKey) Signer {
	return &PrivateKeySigner{
		privateKey: privateKey,
	}
}

func (s *PrivateKeySigner) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	signature, err := rsa.SignPKCS1v15(nil, s.privateKey, crypto.SHA256, digest)
	if err != nil {
		return nil, fmt.Errorf("error signing JWT: %w", err)
	}
	return signature, nil
}

type KMSSigner struct {
	client *kms.KeyManagementClient
	keyID  string
}

// Sign implements Signer.
func (s *KMSSigner) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	req := &kmspb.AsymmetricSignRequest{
		Name: s.keyID,
		Data: digest,
	}
	resp, err := s.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func NewKMSSigner(ctx context.Context, keyID string) Signer {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		log.Fatalf("failed to setup client: %v", err)
	}
	return &KMSSigner{
		client: client,
	}
}
