// Copyright 2023 The Authors (see AUTHORS file)
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

// Package testutil contains common util functions to facilitate tests.
package testutil

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// UploadGCSFiles uploads an object to GCS bucket.
func UploadGCSFile(ctx context.Context, gcsClient *storage.Client, bucket, object string, data io.Reader, metadata map[string]string, conditions *storage.Conditions) error {
	o := gcsClient.Bucket(bucket).Object(object)

	// Set the upload conditions.
	// See: https://pkg.go.dev/cloud.google.com/go/storage#Conditions
	o = o.If(*conditions)

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	wc.Metadata = metadata

	if _, err := io.Copy(wc, data); err != nil {
		return fmt.Errorf("failed to copy bytes: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}
