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

package logging_test

import (
	"testing"

	"github.com/abcxyz/pkg/logging"
)

//nolint:thelper // These are examples
func ExampleTestLogger() {
	_ = func(t *testing.T) { // func TestMyThing(t *testing.T)
		logger = logging.TestLogger(t)
	}
}

//nolint:thelper // These are examples
func ExampleTestLogger_context() {
	_ = func(t *testing.T) { // func TestMyThing(t *testing.T)
		// Most tests rely on the logger in the context, so here's a fast way to
		// inject a test logger into the context.
		ctx := logging.WithLogger(t.Context(), logging.TestLogger(t))

		// Use ctx in tests. Anything that extracts a logger from the context will
		// get the test logger now.
		_ = ctx
	}
}
