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

// Package workerpool defines abstractions for parallelizing tasks.
package workerpool

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// ErrStopped is the error returned when the worker is stopped.
var ErrStopped = fmt.Errorf("worker is stopped")

// Void is a convenience struct for workers that do not actually return values.
type Void struct{}

// WorkFunc is a function for executing work.
type WorkFunc[T any] func() (T, error)

// Pool represents an instance of a worker pool. It is same for concurrent use,
// but see function documentation for more specific semantics.
type Pool[T any] struct {
	size int64
	sem  *semaphore.Weighted

	i           int64
	results     []*result[T]
	resultsLock sync.Mutex

	stopped uint32
}

// result is the internal result representation. It is primarily used to
// maintain results ordering.
type result[T any] struct {
	idx    int64
	result *Result[T]
}

// Result is the final result returned to the caller.
type Result[T any] struct {
	Value T
	Error error
}

// New creates a new worker pool that executes work in parallel, up to the
// maximum provided concurrency. Work is guaranteed to be executed in the order
// in which it was enqueued, but is not guaranteed to complete in the order in
// which it was enqueued (i.e. this is not a pipeline).
//
// If the provided concurrency is less than 1, it defaults to the number of CPU
// cores.
func New[T any](concurrency int64) *Pool[T] {
	if concurrency < 1 {
		concurrency = int64(runtime.NumCPU())
	}
	if concurrency < 1 {
		concurrency = 1
	}

	return &Pool[T]{
		size:    concurrency,
		i:       -1,
		sem:     semaphore.NewWeighted(concurrency),
		results: make([]*result[T], 0, concurrency),
	}
}

// Do adds new work into the queue. If there are no available workers in the
// pool, it blocks until a worker becomes available or until the provided
// context is cancelled. The function returns when the work has been
// successfully scheduled.
//
// To wait for all work to be completed and read the results, call [Pool.Done].
// This function only returns an error on two conditions:
//
//   - The worker pool was stopped via a call to [Pool.Done]. You should not
//     enqueue more work. The error will be [ErrStopped].
//   - The incoming context was cancelled. You should probably not enqueue more
//     work, but this is an application-specific decision. The error will be
//     [context.DeadlineExceeded] or [context.Canceled].
//
// Never call Do from within a Do function because it will deadlock.
func (p *Pool[T]) Do(ctx context.Context, fn WorkFunc[T]) error {
	// Do not enqueue new work if the worker pool is stopped.
	if p.isStopped() {
		return ErrStopped
	}

	if err := p.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}

	// It's possible the worker pool was stopped while we were waiting for the
	// semaphore to acquire, but the worker pool is actually stopped.
	if p.isStopped() {
		p.sem.Release(1)
		return ErrStopped
	}

	i := atomic.AddInt64(&p.i, 1)

	go func() {
		defer p.sem.Release(1)
		t, err := fn()

		p.resultsLock.Lock()
		defer p.resultsLock.Unlock()
		p.results = append(p.results, &result[T]{
			idx: i,
			result: &Result[T]{
				Value: t,
				Error: err,
			},
		})
	}()

	return nil
}

// Done immediately stops the worker pool and prevents new work from being
// enqueued. Then it waits for all existing work to finish and results the
// results.
//
// The results are returned in the order in which jobs were enqueued into the
// worker pool. Each result will include a result value or corresponding error
// type.
//
// The function will return an error if:
//
//   - The pool is stopped. The error will be [ErrStopped].
//   - The incoming context is cancelled. The error will be
//     [context.DeadlineExceeded] or [context.Canceled].
//   - Any of the worker jobs returned a non-nil error. The error will be a
//     multi-error [errors.Unwrap].
//
// If the worker pool is already done, it returns [ErrStopped].
func (p *Pool[T]) Done(ctx context.Context) ([]*Result[T], error) {
	if !atomic.CompareAndSwapUint32(&p.stopped, 0, 1) {
		return nil, ErrStopped
	}

	if err := p.sem.Acquire(ctx, p.size); err != nil {
		return nil, fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer p.sem.Release(p.size)

	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	// Fix insertion order.
	final := make([]*Result[T], len(p.results))
	for _, v := range p.results {
		final[v.idx] = v.result
	}

	// Aggregate any errors into a multi-error. Individual errors are still
	// available on the specific result.
	var merr error
	for _, v := range final {
		merr = errors.Join(merr, v.Error)
	}

	return final, merr
}

// isStopped returns true if the worker pool is stopped, false otherwise. It is
// safe for concurrent use.
func (p *Pool[T]) isStopped() bool {
	return atomic.LoadUint32(&p.stopped) == 1
}