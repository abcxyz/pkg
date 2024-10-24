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
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

type KeyManagementClient interface {
	AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error)
}

type CloudKmsClient struct {
	client *kms.KeyManagementClient
}

func (c *CloudKmsClient) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	resp, err := c.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error response from AsymmetricSign: %w", err)
	}
	return resp, nil
}

func NewCloudKmsClient(ctx context.Context) (KeyManagementClient, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating cloud kms client: %w", err)
	}
	return &CloudKmsClient{
		client: client,
	}, nil
}
