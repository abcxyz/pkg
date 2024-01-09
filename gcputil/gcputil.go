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

// Package gcputil exposes helpers for interacting with Google Cloud.
package gcputil

import (
	"context"
	"os"

	"github.com/abcxyz/pkg/gcpmetadata"
	"github.com/abcxyz/pkg/logging"
)

// ProjectID attempts to get the project ID, starting with the fastest
// resolution ($PROJECT_ID, $GOOGLE_PROJECT, $GOOGLE_CLOUD_PROJECT) and then
// evetually querying the metadata server. If it fails to find a project ID, it
// logs an error.
//
// The results are not cached and could result in outbound HTTP calls. As such,
// callers should cache the response since it is unlikely to change.
func ProjectID(ctx context.Context) string {
	for _, name := range []string{
		"PROJECT_ID",
		"GOOGLE_PROJECT",
		"GOOGLE_CLOUD_PROJECT",
	} {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}

	v, err := gcpmetadata.NewClient().ProjectID(ctx)
	if err != nil {
		logging.FromContext(ctx).ErrorContext(ctx, "failed to get project id", "error", err)
		return ""
	}
	return v
}
