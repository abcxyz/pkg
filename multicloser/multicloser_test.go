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

package multicloser

import (
	"fmt"
	"testing"

	"github.com/abcxyz/pkg/testutil"
)

func TestAppend(t *testing.T) {
	t.Parallel()

	t.Run("nil_closer", func(t *testing.T) {
		t.Parallel()

		c := Append(nil, func() {})
		if c == nil {
			t.Error("expected not nil")
		}
	})

	t.Run("nil_func", func(t *testing.T) {
		t.Parallel()

		var c *Closer
		c = Append(c, (func())(nil))
		if got, want := len(c.fns), 0; got != want {
			t.Errorf("expected %d to be %d: %v", got, want, c.fns)
		}
	})

	t.Run("variadic", func(t *testing.T) {
		t.Parallel()

		var c *Closer
		c = Append(c, func() {}, func() {})
		c = Append(c, func() error { return nil }, func() error { return nil })
		if got, want := len(c.fns), 4; got != want {
			t.Errorf("expected %d to be %d: %v", got, want, c.fns)
		}
	})
}

func TestClose(t *testing.T) {
	t.Parallel()

	t.Run("nil_closer", func(t *testing.T) {
		t.Parallel()

		// This test is mostly checking to ensure we don't panic.
		var c *Closer
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("nil_func", func(t *testing.T) {
		t.Parallel()

		// We have to write directly to the slice to bypass the validation in Append.
		c := &Closer{}
		c.fns = append(c.fns, nil, nil)
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ordered", func(t *testing.T) {
		t.Parallel()

		var c *Closer
		for i := 0; i < 5; i++ {
			i := i
			c = Append(c, func() error {
				return fmt.Errorf("%d", i)
			})
		}

		got := c.Close()
		want := "0\n1\n2\n3\n4"
		if diff := testutil.DiffErrString(got, want); diff != "" {
			t.Errorf(diff)
		}
	})
}
