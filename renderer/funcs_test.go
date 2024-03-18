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

package renderer

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/pointer"
	"github.com/abcxyz/pkg/testutil"
)

func TestToStringSlice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		exp  []string
		err  string
	}{
		{
			name: "nil",
			in:   nil,
			exp:  nil,
		},
		{
			name: "not_slice",
			in:   3,
			err:  "value is not a slice",
		},
		{
			name: "pointers",
			in:   []*int{pointer.To(2), pointer.To(3), pointer.To(4)},
			exp:  []string{"2", "3", "4"},
		},
		{
			name: "stringer",
			in:   []time.Duration{10 * time.Second, 20 * time.Hour},
			exp:  []string{"10s", "20h0m0s"},
		},
		{
			name: "error",
			in:   []error{fmt.Errorf("one"), fmt.Errorf("two")},
			exp:  []string{"one", "two"},
		},
		{
			name: "string",
			in:   []string{"one", "two"},
			exp:  []string{"one", "two"},
		},
		{
			name: "int",
			in:   []int{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "int8",
			in:   []int8{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "int16",
			in:   []int16{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "int32",
			in:   []int32{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "int64",
			in:   []int64{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "uint",
			in:   []uint{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "uint8",
			in:   []uint8{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "uint16",
			in:   []uint16{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "uint32",
			in:   []uint32{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "uint64",
			in:   []uint64{1, 2},
			exp:  []string{"1", "2"},
		},
		{
			name: "mixed",
			in:   []any{5, 10 * time.Second, fmt.Errorf("hello"), "world"},
			exp:  []string{"5", "10s", "hello", "world"},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			list, err := toStringSlice(tc.in)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Fatal(err)
			}
			if diff := cmp.Diff(list, tc.exp); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		exp  string
	}{
		{
			name: "nil",
			in:   nil,
			exp:  "",
		},
		{
			name: "single",
			in:   []string{"a"},
			exp:  "a",
		},
		{
			name: "multi",
			in:   []string{"a", "b", "c"},
			exp:  "a,b,c",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := joinStrings(tc.in, ",")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(result, tc.exp); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestToSentence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		exp  string
	}{
		{
			name: "nil",
			in:   nil,
			exp:  "",
		},
		{
			name: "single",
			in:   []string{"a"},
			exp:  "a",
		},
		{
			name: "double",
			in:   []string{"a", "b"},
			exp:  "a and b",
		},
		{
			name: "multi",
			in:   []string{"a", "b", "c"},
			exp:  "a, b, and c",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := toSentence(tc.in, "and")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(result, tc.exp); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestToPercent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   float64
		exp  string
	}{
		{
			name: "zero",
			in:   0,
			exp:  "0.00%",
		},
		{
			name: "long",
			in:   3.14159,
			exp:  "314.16%",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := toPercent(tc.in)
			if diff := cmp.Diff(result, tc.exp); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestTrimSpace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "empty",
			in:   "",
			exp:  "",
		},
		{
			name: "unicode",
			in:   "val12!\uFEFF",
			exp:  "val12!",
		},
		{
			name: "unicode",
			in:   " val12!  \r\t",
			exp:  "val12!",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := trimSpace(tc.in)
			if diff := cmp.Diff(result, tc.exp); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
