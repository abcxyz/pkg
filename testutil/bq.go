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
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
)

// QueryBQEntries queries the BQ and checks if a bigqury entry matched the query exists or not and return the results.
// [T any] needs to be bigquery iteration supported type (e.g. string, bigquery.Value, struct{}).
func QueryBQEntries[T any](ctx context.Context, query *bigquery.Query) ([]*T, error) {
	job, err := query.Run(ctx)
	if err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to run query: %w", err))
	}

	if status, err := job.Wait(ctx); err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to wait for query: %w", err))
	} else if err = status.Err(); err != nil {
		return nil, retry.RetryableError(fmt.Errorf("query failed: %w", err))
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to read job: %w", err))
	}

	var entries []*T
	for {
		var entry T
		err := it.Next(&entry)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, retry.RetryableError(fmt.Errorf("failed to get next entry: %w", err))
		}

		entries = append(entries, &entry)
	}
	return entries, nil
}
