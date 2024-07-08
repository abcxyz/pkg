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
)

func TestIntersectMapKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		maps []map[string]string
		exp  map[string]string
	}{
		{
			name: "nil_maps",
			maps: nil,
			exp:  map[string]string{},
		},
		{
			name: "empty_maps",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				nil,
				{"foo": "bar", "zip": "zap"},
				{},
			},
			exp: map[string]string{},
		},
		{
			name: "none",
			maps: []map[string]string{
				{"foo": "bar"},
				{"zip": "zap"},
			},
			exp: map[string]string{},
		},
		{
			name: "one",
			maps: []map[string]string{
				{"foo": "bar"},
				{"foo": "bar"},
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "many",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap"},
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
			},
			exp: map[string]string{
				"foo": "bar", "zip": "zap",
			},
		},
		{
			name: "first_overwrites_decreasing_size",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"foo": "bar2", "zip": "zap", "fruit": "banana"},
				{"foo": "bar3", "zip": "zap"},
				{"foo": "bar4"},
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "all",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"foo": "bar", "zip": "zap"},
				{"foo": "bar"},
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input maps were not modified.
			mapsCopy := testDeepCopyMaps(t, tc.maps)

			got := IntersectMapKeys(tc.maps...)
			if diff := cmp.Diff(tc.exp, got); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure the maps were not modified.
			if diff := cmp.Diff(mapsCopy, tc.maps); diff != "" {
				t.Errorf("insersection modified map (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestUnionMapKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		maps []map[string]string
		exp  map[string]string
	}{
		{
			name: "nil_maps",
			maps: nil,
			exp:  map[string]string{},
		},
		{
			name: "empty_maps",
			maps: []map[string]string{
				{"foo": "bar"},
				nil,
				{"zip": "zap"},
				{},
			},
			exp: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
		},
		{
			name: "one",
			maps: []map[string]string{
				{"foo": "bar"},
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "many",
			maps: []map[string]string{
				{"foo": "bar"},
				{"zip": "zap"},
			},
			exp: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
		},
		{
			name: "first_overwrites_same_size",
			maps: []map[string]string{
				{"foo": "bar"},
				{"zip": "zap"},
				{"foo": "bar2"},
			},
			exp: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
		},
		{
			name: "all",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"foo": "bar", "zip": "zap"},
				{"foo": "bar"},
			},
			exp: map[string]string{
				"foo":   "bar",
				"zip":   "zap",
				"fruit": "banana",
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input maps were not modified.
			mapsCopy := testDeepCopyMaps(t, tc.maps)

			got := UnionMapKeys(tc.maps...)
			if diff := cmp.Diff(tc.exp, got); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure the maps were not modified.
			if diff := cmp.Diff(mapsCopy, tc.maps); diff != "" {
				t.Errorf("insersection modified map (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestSubtractMapKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		maps []map[string]string
		exp  map[string]string
	}{
		{
			name: "nil_maps",
			maps: nil,
			exp:  map[string]string{},
		},
		{
			name: "empty_maps",
			maps: []map[string]string{
				{"foo": "bar"},
				nil,
				{"zip": "zap"},
				{},
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "one",
			maps: []map[string]string{
				{"foo": "bar"},
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "many",
			maps: []map[string]string{
				{"foo": "bar"},
				{"foo": "bar", "zip": "zap"},
			},
			exp: map[string]string{},
		},
		{
			name: "remaining",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"zip": "zap"},
			},
			exp: map[string]string{
				"foo":   "bar",
				"fruit": "banana",
			},
		},
		{
			name: "all",
			maps: []map[string]string{
				{"foo": "bar", "zip": "zap", "fruit": "banana"},
				{"foo": "bar", "zip": "zap"},
				{"fruit": "banana"},
				{"fruit": "banana"},
			},
			exp: map[string]string{},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get a copy so we can verify the input maps were not modified.
			mapsCopy := testDeepCopyMaps(t, tc.maps)

			got := SubtractMapKeys(tc.maps...)
			if diff := cmp.Diff(tc.exp, got); diff != "" {
				t.Errorf("incorrect insersection (-want/+got):\n%s", diff)
			}

			// Ensure the maps were not modified.
			if diff := cmp.Diff(mapsCopy, tc.maps); diff != "" {
				t.Errorf("insersection modified map (-want/+got):\n%s", diff)
			}
		})
	}
}

func testDeepCopyMaps[K comparable, V any](tb testing.TB, maps []map[K]V) []map[K]V {
	tb.Helper()

	if len(maps) == 0 {
		return nil
	}

	final := make([]map[K]V, len(maps))
	for i, m := range maps {
		if m == nil {
			final[i] = nil
			continue
		}

		inner := make(map[K]V, len(m))
		for k, v := range m {
			inner[k] = v
		}
		final[i] = inner
	}
	return final
}
