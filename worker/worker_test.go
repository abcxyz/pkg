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

package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestWorker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	w := New[*Void](3)

	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		if err := w.Do(ctx, func() (*Void, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := w.Done(ctx); err != nil {
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

		w := New[*Void](2)
		if _, err := w.Done(ctx); err != nil {
			t.Fatal(err)
		}

		if err := w.Do(ctx, func() (*Void, error) {
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

		w := New[*Void](2)
		if _, err := w.Done(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Done(ctx); !errors.Is(err, ErrStopped) {
			t.Errorf("expected %q to be %q", err, ErrStopped)
		}
	})

	t.Run("error_results", func(t *testing.T) {
		t.Parallel()

		w := New[*Void](2)

		for i := 0; i < 5; i++ {
			i := i

			if err := w.Do(ctx, func() (*Void, error) {
				time.Sleep(time.Duration(i) * time.Millisecond)
				return nil, fmt.Errorf("%d", i)
			}); err != nil {
				t.Fatal(err)
			}
		}

		results, err := w.Done(ctx)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{"0", "1", "2", "3", "4"}
		var got []string
		for _, result := range results {
			got = append(got, fmt.Sprintf("%s", result.Error))
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("justs: diff (-want, +got):\n%s", diff)
		}
	})

	t.Run("ordered_results", func(t *testing.T) {
		t.Parallel()

		w := New[int](2)

		for i := 0; i < 5; i++ {
			i := i

			if err := w.Do(ctx, func() (int, error) {
				time.Sleep(time.Duration(i) * time.Millisecond)
				return i, nil
			}); err != nil {
				t.Fatal(err)
			}
		}

		results, err := w.Done(ctx)
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

	t.Run("cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, done := context.WithTimeout(context.Background(), 10*time.Millisecond)
		t.Cleanup(done)

		w := New[int](2)

		for i := 0; i < 5; i++ {
			i := i

			err := w.Do(ctx, func() (int, error) {
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
		results, err := w.Done(finishCtx)
		if err != nil {
			t.Fatal(err)
		}
		want := []int{0, 1}
		var got []int
		for _, result := range results {
			got = append(got, result.Value)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("justs: diff (-want, +got):\n%s", diff)
		}
	})

	t.Run("concurreny", func(t *testing.T) {
		t.Parallel()

		w := New[int](3)
		var wg sync.WaitGroup

		for i := 0; i < 15; i++ {
			i := i
			wg.Add(1)

			go func() {
				defer wg.Done()
				_ = w.Do(ctx, func() (int, error) {
					time.Sleep(time.Duration(i) * time.Millisecond)
					return i, nil
				})
			}()
		}

		wg.Wait()

		results, err := w.Done(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := len(results), 15; got != want {
			t.Errorf("expected %d to be %d: %v", got, want, results)
		}
	})
}
