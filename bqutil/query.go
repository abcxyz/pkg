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

// Package bqutil provides utils to interact with BigQuery.
package bqutil

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/abcxyz/pkg/logging"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
)

// Query is the interface to execute a query.
type Query[T any] interface {
	// Execute runs the query and returns the result in type []*T.
	Execute(context.Context) ([]*T, error)

	// String returns the underlying query string.
	String() string
}

type bqQuery[T any] struct {
	bqq *bigquery.Query
}

// NewQuery creates a new BigQuery query.
func NewQuery[T any](q *bigquery.Query) Query[T] {
	return &bqQuery[T]{bqq: q}
}

// String returns the BigQuery query string.
func (q *bqQuery[T]) String() string {
	return q.bqq.Q
}

// Execute runs the BigQuery query and get the result.
func (q *bqQuery[T]) Execute(ctx context.Context) ([]*T, error) {
	job, err := q.bqq.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	if status, err := job.Wait(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for query: %w", err)
	} else if status.Err() != nil {
		return nil, fmt.Errorf("query failed: %w", status.Err())
	}

	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read query result: %w", err)
	}

	var entries []*T
	for {
		var v T
		err := it.Next(&v)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get next entry: %w", err)
		}

		entries = append(entries, &v)
	}
	return entries, nil
}

// RetryQueryEntries retries the given [Query] until either the query result
// returns "wantCount" number of entries or the retry backoff is exhausted (and
// returns error).
//
// To collect result in custom Go struct:
//
//	type queryResult struct {
//		ColNameX, ColNameY string
//	}
//
//	q := NewQuery[queryResult](bigqueryQuery)
//	backoff := retry.WithMaxRetries(3, retry.NewConstant(time.Second))
//	result, err := RetryQueryEntries(context.Background(), q, 1, backoff)
//	// result will be in type []*queryResult
//
// To use this func in testing:
//
//	// Set the test logger in the context on the debug level.
//	ctx := logging.WithLogger(context.Background(),
//		logging.TestLogger(t, zaptest.Level(zapcore.DebugLevel)))
//
//	q := NewQuery[queryResult](bigqueryQuery)
//	backoff := retry.WithMaxRetries(3, retry.NewConstant(time.Second))
//	result, err := RetryQueryEntries(ctx, q, 1, backoff)
func RetryQueryEntries[T any](ctx context.Context, q Query[T], wantCount int, backoff retry.Backoff) ([]*T, error) {
	logger := logging.FromContext(ctx)
	logger.Debugw("Start retrying query", "query", q.String())

	var result []*T
	if err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		entries, err := q.Execute(ctx)
		if err != nil {
			logger.Debugw("Query failed; will retry", "error", err)
			return retry.RetryableError(err)
		}

		gotCount := len(entries)
		if gotCount >= wantCount {
			result = entries
			return nil
		}

		logger.Debugw("Not enough entries; will retry", "gotCount", gotCount, "wantCount", wantCount)
		return retry.RetryableError(fmt.Errorf("not enough entries"))

	}); err != nil {
		return nil, fmt.Errorf("retry backoff exhausted: %w", err)
	}

	return result, nil
}
