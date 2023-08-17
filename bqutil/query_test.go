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

package bqutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-retry"
)

type fakeQuery[T any] struct {
	count           int
	result          []T
	returnErrString string
}

func newFakeQuery[T any](result []T, returnErrString string) Query[T] {
	return &fakeQuery[T]{
		result:          result,
		returnErrString: returnErrString,
	}
}

func (q *fakeQuery[T]) String() string {
	return "fake query"
}

func (q *fakeQuery[T]) Execute(ctx context.Context) ([]T, error) {
	if q.returnErrString != "" {
		return nil, fmt.Errorf(q.returnErrString)
	}

	// Only return one more entry on each query.
	q.count += 1
	return q.result[:q.count], nil
}

type fakeQueryResult struct {
	Name string
}

func TestRetryQueryEntries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		wantEntries    []*fakeQueryResult
		wantQueryCount int
		wantErrString  string
	}{
		{
			name:           "success_one_entry",
			wantEntries:    []*fakeQueryResult{{Name: "one"}},
			wantQueryCount: 1,
		},
		{
			name: "success_multi_entries",
			wantEntries: []*fakeQueryResult{
				{Name: "one"},
				{Name: "two"},
				{Name: "three"},
			},
			wantQueryCount: 3,
		},
		{
			name:          "error_backoff_exhausted",
			wantErrString: "query failed",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

			wantCount := len(tc.wantEntries)

			backoff := retry.NewConstant(time.Millisecond)
			backoff = retry.WithMaxRetries(uint64(wantCount)+1, backoff)

			q := newFakeQuery(tc.wantEntries, tc.wantErrString)
			gotEntries, err := RetryQueryEntries(ctx, q, wantCount, backoff)

			if diff := testutil.DiffErrString(err, tc.wantErrString); diff != "" {
				t.Errorf("RetryQueryEntries unexpected error: %s", diff)
			}

			if diff := cmp.Diff(tc.wantEntries, gotEntries); diff != "" {
				t.Errorf("RetryQueryEntries result (-want,+got):\n%s", diff)
			}

			fq, ok := q.(*fakeQuery[*fakeQueryResult])
			if !ok {
				t.Fatalf("Wrong fake query type %T", q)
			}

			if got, want := fq.count, tc.wantQueryCount; got != want {
				t.Errorf("RetryQueryEntries query count got=%d, want=%d", got, want)
			}
		})
	}
}
