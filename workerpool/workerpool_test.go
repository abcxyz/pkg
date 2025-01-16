// Copyright 2022 The Authors (see AUTHORS file)
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

package workerpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestWorker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool := New[*Void](&Config{
		Concurrency: 3,
	})

	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		if err := pool.Do(ctx, func() (*Void, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := pool.Done(ctx); err != nil {
		t.Fatal(err)
	}

	if got, want := time.Now().UTC().Sub(now), 40*time.Millisecond; got > want {
		t.Errorf("expected parallelism (took %s, expected less than %s)", got, want)
	}
}

func TestWorker_Do(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("stopped", func(t *testing.T) {
		t.Parallel()

		pool := New[*Void](&Config{
			Concurrency: 2,
		})
		if _, err := pool.Done(ctx); err != nil {
			t.Fatal(err)
		}

		if err := pool.Do(ctx, func() (*Void, error) {
			return nil, nil
		}); !errors.Is(err, ErrStopped) {
			t.Errorf("expected %q to be %q", err, ErrStopped)
		}
	})
}

func TestWorker_Done(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("stopped", func(t *testing.T) {
		t.Parallel()

		pool := New[*Void](&Config{
			Concurrency: 2,
		})
		if _, err := pool.Done(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error_results", func(t *testing.T) {
		t.Parallel()

		pool := New[*Void](&Config{
			Concurrency: 2,
		})

		for i := 0; i < 5; i++ {
			if err := pool.Do(ctx, func() (*Void, error) {
				time.Sleep(time.Duration(i) * time.Millisecond)
				return nil, fmt.Errorf("%d", i)
			}); err != nil {
				t.Fatal(err)
			}
		}

		results, err := pool.Done(ctx)
		if err == nil {
			t.Fatal("expected error, but got nothing")
		}
		if got, want := err.Error(), "0\n1\n2\n3\n4"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		want := []string{"0", "1", "2", "3", "4"}
		var got []string
		for _, result := range results {
			got = append(got, fmt.Sprintf("%s", result.Error))
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Error(diff)
		}
	})

	t.Run("ordered_results", func(t *testing.T) {
		t.Parallel()

		pool := New[int](&Config{
			Concurrency: 2,
		})

		for i := 0; i < 5; i++ {
			if err := pool.Do(ctx, func() (int, error) {
				time.Sleep(time.Duration(i) * time.Millisecond)
				return i, nil
			}); err != nil {
				t.Fatal(err)
			}
		}

		results, err := pool.Done(ctx)
		if err != nil {
			t.Fatal(err)
		}

		want := []int{0, 1, 2, 3, 4}
		var got []int
		for _, result := range results {
			got = append(got, result.Value)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("justs: diff (-want, +got):\n%s", diff)
		}
	})

	t.Run("stop_on_error", func(t *testing.T) {
		t.Parallel()

		sentinelError := fmt.Errorf("error from test")

		pool := New[int](&Config{
			Concurrency: 2,
			StopOnError: true,
		})

		for i := 0; i < 5; i++ {
			_ = pool.Do(ctx, func() (int, error) {
				if i < 2 {
					return i, nil
				}
				return 0, sentinelError
			})
		}

		results, err := pool.Done(ctx)
		if got, want := err, sentinelError; !errors.Is(got, want) {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := err, ErrStopped; !errors.Is(got, want) {
			t.Errorf("expected %q to be %q", got, want)
		}

		want := []*Result[int]{
			{Value: 0},
			{Value: 1},
			{Error: sentinelError},

			// These jobs could have queued before the other job returned as stopped,
			// so we assume any error is a good error. It could be the sentinel error
			// or it could be [ErrStopped].
			{Error: cmpopts.AnyError},
			{Error: cmpopts.AnyError},
		}
		if diff := cmp.Diff(want, results, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("justs: diff (-want, +got):\n%s", diff)
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, done := context.WithTimeout(context.Background(), 10*time.Millisecond)
		t.Cleanup(done)

		pool := New[int](&Config{
			Concurrency: 2,
		})

		for i := 0; i < 5; i++ {
			err := pool.Do(ctx, func() (int, error) {
				time.Sleep(100 * time.Millisecond)
				return i, nil
			})

			// The worker has a size of 2, so the first 2 should queue non-blocking
			// and sleep for 100ms. The third should block and then the context should
			// cancel at 10ms.
			if i < 2 {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("expected %v to be %v", err, context.DeadlineExceeded)
				}
			}
		}

		finishCtx := context.Background()
		results, err := pool.Done(finishCtx)
		if got, want := err, context.DeadlineExceeded; !errors.Is(got, want) {
			t.Errorf("expected %v to be %v", got, want)
		}

		want := []*Result[int]{
			{Value: 0},
			{Value: 1},
			{Error: context.DeadlineExceeded},
			{Error: context.DeadlineExceeded},
			{Error: context.DeadlineExceeded},
		}
		if diff := cmp.Diff(results, want, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("justs: diff (-want, +got):\n%s", diff)
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		t.Parallel()

		pool := New[int](&Config{
			Concurrency: 3,
		})
		var wg sync.WaitGroup

		for i := 0; i < 15; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				_ = pool.Do(ctx, func() (int, error) {
					time.Sleep(time.Duration(i) * time.Millisecond)
					return i, nil
				})
			}()
		}

		wg.Wait()

		results, err := pool.Done(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := len(results), 15; got != want {
			t.Errorf("expected %d to be %d: %v", got, want, results)
		}
	})
}
