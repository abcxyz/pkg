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

package sets

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestIntersect(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		slices [][]string
		exp    []string
	}{
		{
			name:   "nil_slices",
			slices: nil,
			exp:    []string{},
		},
		{
			name: "empty_slices",
			slices: [][]string{
				{"foo", "bar", "zip", "zap"},
				nil,
				{"foo", "bar"},
				{},
			},
			exp: []string{},
		},
		{
			name: "none",
			slices: [][]string{
				{"foo", "bar"},
				{"zip", "zap"},
			},
			exp: []string{},
		},
		{
			name: "one",
			slices: [][]string{
				{"foo", "bar"},
				{"foo", "baz"},
			},
			exp: []string{"foo"},
		},
		{
			name: "many",
			slices: [][]string{
				{"foo", "bar", "zip", "zap"},
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
			},
			exp: []string{"foo", "bar", "zip", "zap"},
		},
		{
			name: "all",
			slices: [][]string{
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
				{"foo", "bar", "zip", "zap"},
				{"foo", "bar"},
			},
			exp: []string{"foo", "bar"},
		},
		{
			name: "duplicates",
			slices: [][]string{
				{"foo", "foo", "foo", "bar"},
				{"foo", "bar"},
			},
			exp: []string{"foo", "bar"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input slices were not modified.
			slicesCopy := testDeepCopySlices(t, tc.slices)

			got := Intersect(tc.slices...)
			if diff := cmp.Diff(tc.exp, got, cmpopts.SortSlices(func(x, y string) bool {
				return x < y
			})); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure we freed up as much memory as possible.
			if c, l := cap(got), len(got); c != l {
				t.Errorf("expected cap(%d) to equal len(%d)", c, l)
			}

			// Ensure the slices were not modified.
			if diff := cmp.Diff(slicesCopy, tc.slices); diff != "" {
				t.Errorf("insersection modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestIntersectStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		slices [][]string
		exp    []string
	}{
		{
			name:   "nil_slices",
			slices: nil,
			exp:    []string{},
		},
		{
			name: "empty_slices",
			slices: [][]string{
				{"foo", "bar", "zip", "zap"},
				nil,
				{"foo", "bar"},
				{},
			},
			exp: []string{},
		},
		{
			name: "none",
			slices: [][]string{
				{"foo", "bar"},
				{"zip", "zap"},
			},
			exp: []string{},
		},
		{
			name: "one",
			slices: [][]string{
				{"foo", "bar"},
				{"foo", "baz"},
			},
			exp: []string{"foo"},
		},
		{
			name: "many",
			slices: [][]string{
				{"foo", "bar", "zip", "zap", "zonk", "zink", "apple", "banana"},
				{"apple", "bar", "foo", "bar", "zip", "zap", "apple", "fruit", "banana", "zink", "zonk", "apple"},
			},
			exp: []string{"foo", "bar", "zip", "zap", "zonk", "zink", "apple", "banana"},
		},
		{
			name: "all",
			slices: [][]string{
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
				{"foo", "bar", "zip", "zap"},
				{"foo", "bar"},
			},
			exp: []string{"foo", "bar"},
		},
		{
			name: "duplicates",
			slices: [][]string{
				{"foo", "foo", "foo", "bar"},
				{"foo", "bar"},
			},
			exp: []string{"foo", "bar"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input slices were not modified.
			slicesCopy := testDeepCopySlices(t, tc.slices)

			got := IntersectStable(tc.slices...)
			if diff := cmp.Diff(tc.exp, got); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure we freed up as much memory as possible.
			if c, l := cap(got), len(got); c != l {
				t.Errorf("expected cap(%d) to equal len(%d)", c, l)
			}

			// Ensure the slices were not modified.
			if diff := cmp.Diff(slicesCopy, tc.slices); diff != "" {
				t.Errorf("insersection modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestUnion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		slices [][]string
		exp    []string
	}{
		{
			name:   "nil_slices",
			slices: nil,
			exp:    []string{},
		},
		{
			name: "empty_slices",
			slices: [][]string{
				{"foo"},
				nil,
				{"bar"},
				{},
			},
			exp: []string{"foo", "bar"},
		},
		{
			name: "one",
			slices: [][]string{
				{"foo", "bar"},
			},
			exp: []string{"foo", "bar"},
		},
		{
			name: "many",
			slices: [][]string{
				{"foo", "bar"},
				{"zip", "zap"},
			},
			exp: []string{"foo", "bar", "zip", "zap"},
		},
		{
			name: "all",
			slices: [][]string{
				{"foo", "bar", "zip", "zap"},
				{"foo", "bar"},
				{"apple", "banana"},
			},
			exp: []string{"foo", "bar", "zip", "zap", "apple", "banana"},
		},
		{
			name: "duplicates",
			slices: [][]string{
				{"bar", "foo", "foo", "foo", "bar"},
				{"foo", "bar"},
			},
			exp: []string{"bar", "foo"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input slices were not modified.
			slicesCopy := testDeepCopySlices(t, tc.slices)

			got := Union(tc.slices...)
			if diff := cmp.Diff(tc.exp, got); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure the slices were not modified.
			if diff := cmp.Diff(slicesCopy, tc.slices); diff != "" {
				t.Errorf("insersection modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		slices [][]string
		exp    []string
	}{
		{
			name:   "nil_slices",
			slices: nil,
			exp:    []string{},
		},
		{
			name: "empty_slices",
			slices: [][]string{
				{"foo", "bar"},
				nil,
				{"zip", "zap"},
				{},
			},
			exp: []string{"foo", "bar"},
		},
		{
			name: "one",
			slices: [][]string{
				{"foo"},
				{"foo"},
			},
			exp: []string{},
		},
		{
			name: "many",
			slices: [][]string{
				{"foo", "bar"},
				{"foo", "bar", "zip", "zap"},
			},
			exp: []string{},
		},
		{
			name: "remaining",
			slices: [][]string{
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
				{"zip", "zap"},
			},
			exp: []string{"foo", "bar", "fruit", "banana"},
		},
		{
			name: "all",
			slices: [][]string{
				{"foo", "bar", "zip", "zap", "fruit", "banana"},
				{"foo", "bar", "zip", "zap"},
				{"fruit", "banana"},
				{"fruit", "banana"},
			},
			exp: []string{},
		},
		{
			name: "duplicates",
			slices: [][]string{
				{"bar", "foo", "foo", "foo", "bar"},
				{"bar"},
			},
			exp: []string{"foo", "foo", "foo"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input slices were not modified.
			slicesCopy := testDeepCopySlices(t, tc.slices)

			got := Subtract(tc.slices...)
			if diff := cmp.Diff(tc.exp, got); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure we freed up as much memory as possible.
			if c, l := cap(got), len(got); c != l {
				t.Errorf("expected cap(%d) to equal len(%d)", c, l)
			}

			// Ensure the slices were not modified.
			if diff := cmp.Diff(slicesCopy, tc.slices); diff != "" {
				t.Errorf("insersection modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func testDeepCopySlices[T comparable](tb testing.TB, slices [][]T) [][]T {
	tb.Helper()

	if len(slices) == 0 {
		return nil
	}

	final := make([][]T, len(slices))
	for i, s := range slices {
		if s == nil {
			final[i] = nil
			continue
		}
		final[i] = append([]T{}, s...)
	}
	return final
}
