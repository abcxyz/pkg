// Copyright 2024 The Authors (see AUTHORS file)
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

package pointer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
	}{
		{
			name: "string",
			in:   "hello",
		},
		{
			name: "empty_string",
			in:   "",
		},
		{
			name: "bool",
			in:   true,
		},
		{
			name: "nil",
			in:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := To(tc.in)
			if got == nil {
				t.Fatal("pointer should not be nil")
			}

			deref := *got
			if diff := cmp.Diff(tc.in, deref); diff != "" {
				t.Errorf("returned diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestDeref(t *testing.T) {
	t.Parallel()

	if got, want := Deref(To("hello")), "hello"; got != want {
		t.Errorf("expected %v to be %v", got, want)
	}

	if got, want := Deref(To(true)), true; got != want {
		t.Errorf("expected %v to be %v", got, want)
	}

	if got, want := Deref(To(any(nil))), any(nil); got != want {
		t.Errorf("expected %v to be %v", got, want)
	}
}
