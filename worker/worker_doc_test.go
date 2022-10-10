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

//nolint:all // This is sample code
package worker_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/abcxyz/pkg/worker"
)

func Example_sleep() {
	ctx := context.TODO()
	w := worker.New[*worker.Void](3)

	for i := 0; i < 5; i++ {
		if err := w.Do(ctx, func() (*worker.Void, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, nil
		}); err != nil {
			// TODO: check err
		}
	}

	results, err := w.Done(ctx)
	if err != nil {
		// TODO: check err
	}
	_ = results
}

func Example_hTTP() {
	ctx := context.TODO()
	w := worker.New[string](0)

	urls := []string{
		"https://apple.com",
		"https://example.com",
		"https://google.com",
	}

	for _, u := range urls {
		// Make a local copy for the closure.
		u := u

		if err := w.Do(ctx, func() (string, error) {
			resp, err := http.Get(u)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}); err != nil {
			// TODO: check err
		}
	}

	results, err := w.Done(ctx)
	if err != nil {
		// TODO: check err
	}

	for i, result := range results {
		fmt.Printf("%s: body(%d), err(%v)\n", urls[i], len(result.Value), result.Error)
	}
}
