// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slices

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMap(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		slice  []string
		mapper func(string) int
		want   []int
	}{
		{
			name:   "nil_slice",
			slice:  nil,
			mapper: func(s string) int { return len(s) },
			want:   nil,
		},
		{
			name:   "empty_slice",
			slice:  []string{},
			mapper: func(s string) int { return len(s) },
			want:   []int{},
		},
		{
			name:   "one element",
			slice:  []string{"foo"},
			mapper: func(s string) int { return len(s) },
			want:   []int{3},
		},
		{
			name:   "many elements",
			slice:  []string{"foo", "bar", "baz", "bing", "bang", "bong"},
			mapper: func(s string) int { return len(s) },
			want:   []int{3, 3, 3, 4, 4, 4},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy, so we can verify the input slices were not modified.
			clone := testClone(tc.slice)

			got := Map(tc.slice, tc.mapper)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("incorrect Map (-want/+got):\n%s", diff)
			}

			// Ensure we allocated up as much memory as possible.
			if c, l := cap(got), len(got); c != l {
				t.Errorf("expected cap(%d) to equal len(%d)", c, l)
			}

			// Ensure the slice was not modified.
			if diff := cmp.Diff(clone, tc.slice); diff != "" {
				t.Errorf("Map modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		slice     []string
		predicate func(string) bool
		want      []string
	}{
		{
			name:      "nil_slice",
			slice:     nil,
			predicate: func(s string) bool { return len(s) == 0 },
			want:      nil,
		},
		{
			name:      "empty_slice",
			slice:     []string{},
			predicate: func(s string) bool { return len(s) == 0 },
			want:      []string{},
		},
		{
			name:      "one_element_match",
			slice:     []string{"foo"},
			predicate: func(s string) bool { return len(s) == 3 },
			want:      []string{"foo"},
		},
		{
			name:      "one_element_no_match",
			slice:     []string{"foo"},
			predicate: func(s string) bool { return len(s) == 0 },
			want:      []string{},
		},
		{
			name:      "many_elements_all_match",
			slice:     []string{"foo", "bar", "baz", "bing", "bang", "bong"},
			predicate: func(s string) bool { return len(s) > 0 },
			want:      []string{"foo", "bar", "baz", "bing", "bang", "bong"},
		},
		{
			name:      "many_elements_none_match",
			slice:     []string{"foo", "bar", "baz", "bing", "bang", "bong"},
			predicate: func(s string) bool { return len(s) == 0 },
			want:      []string{},
		},
		{
			name:      "many_elements_some_match",
			slice:     []string{"foo", "bar", "baz", "bing", "bang", "bong"},
			predicate: func(s string) bool { return len(s) == 3 },
			want:      []string{"foo", "bar", "baz"},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy, so we can verify the input slices were not modified.
			clone := testClone(tc.slice)

			got := Filter(tc.slice, tc.predicate)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("incorrect Filter (-want/+got):\n%s", diff)
			}

			// Ensure the slice was not modified.
			if diff := cmp.Diff(clone, tc.slice); diff != "" {
				t.Errorf("Filter modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestReduce(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		slice        []int
		initialValue int
		accumulator  func(int, int) int
		want         int
	}{
		{
			name:         "initial_value_returned_for_nil_slice",
			slice:        nil,
			initialValue: 10,
			accumulator:  func(a, b int) int { return a + b },
			want:         10,
		},
		{
			name:         "initial_value_returned_for_empty_slice",
			slice:        nil,
			initialValue: 4,
			accumulator:  func(a, b int) int { return a + b },
			want:         4,
		},
		{
			name:         "one_element",
			slice:        []int{2},
			initialValue: 0,
			accumulator:  func(a, b int) int { return a + b },
			want:         2,
		},
		{
			name:         "many_elements",
			slice:        []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			initialValue: 0,
			accumulator:  func(a, b int) int { return a + b },
			want:         55,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy, so we can verify the input slice was not modified.
			clone := testClone(tc.slice)

			got := Reduce(tc.slice, tc.initialValue, tc.accumulator)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("incorrect Reduce (-want/+got):\n%s", diff)
			}

			// Ensure the slice was not modified.
			if diff := cmp.Diff(clone, tc.slice); diff != "" {
				t.Errorf("Reduce modified slice (-want/+got):\n%s", diff)
			}
		})
	}
}

func testClone[T any](slice []T) []T {
	if slice == nil {
		return nil
	}
	clone := make([]T, len(slice))
	copy(clone, slice)
	return clone
}
