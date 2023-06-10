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

// Package multicloser provides a convenient way to join multiple "close"
// functions together so they can be called together. This is especially useful
// to group multiple cleanup function calls and return it as a single "closer"
// to be called later.
package multicloser

import (
	"errors"
)

// Func is the type signature for a closing function. It accepts a function that
// returns an error or a void function.
type Func interface {
	func() error | func()
}

// Closer maintains the ordered list of closing functions. Functions will be run
// in the order in which they were inserted.
//
// It is not safe to use concurrently without locking.
type Closer struct {
	fns []func() error
}

// Append adds the given closer functions. It handles void and error signatures.
// Other signatures should use an anonymous function to match an expected
// signature.
func Append[T Func](c *Closer, fns ...T) *Closer {
	if c == nil {
		c = new(Closer)
	}

	for _, fn := range fns {
		if fn == nil {
			continue
		}

		switch typ := any(fn).(type) {
		case func() error:
			c.fns = append(c.fns, typ)
		case func():
			c.fns = append(c.fns, func() error {
				typ()
				return nil
			})
		default:
			panic("impossible")
		}
	}

	return c
}

// Close runs all closer functions. All closers are guaranteed to run, even if
// they panic. After all closers run, panics will propagate up the stack.
//
// [Close] also panics if it is called on an already-closed Closer.
func (c *Closer) Close() (err error) {
	if c == nil {
		return
	}

	for i := len(c.fns) - 1; i >= 0; i-- {
		fn := c.fns[i]
		if fn != nil {
			// We abuse defer's automatic panic recovery here a bit..
			defer func() {
				err = errors.Join(err, fn())
			}()
		}
	}
	return
}
