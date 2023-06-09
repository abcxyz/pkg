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
//
// Example:
//
//	func caller() {
//		closer, err := callee()
//		defer multicloser.Close(closer) // It's safe to do this first.
//		if err != nil {
//			// Handle err
//		}
//		// Do other stuff
//	}
//
//	func callee() (*multicloser.Closer, error) {
//		closer := multicloser.New()
//
//		client1 , err := newClient()
//		if err != nil {
//			return closer, err
//		}
//		closer.Append(client1.Close)
//
//		client2 , err := newClient()
//		if err != nil {
//			return closer, err
//		}
//		closer.Append(client2.Close)
//
//		return closer, nil
//	}
package multicloser

// CloseFunc is the general close function type.
type CloseFunc func()

// Closer keeps a slice of close functions.
type Closer struct {
	Closes []CloseFunc
}

// New creates a new multicloser.
func New() *Closer {
	return &Closer{}
}

// Append appends more close functions to the multicloser.
func (c *Closer) Append(closes ...CloseFunc) {
	c.Closes = append(c.Closes, closes...)
}

// Close calls all close functions in the multicloser.
func (c *Closer) Close() {
	for _, close := range c.Closes {
		close()
	}
}

// Close is a safer way to close a multicloser that handles the nil case.
func Close(closer *Closer) {
	if closer != nil {
		closer.Close()
	}
}
