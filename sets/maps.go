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
	gomaps "maps"
)

// IntersectMapKeys finds the intersection of all m keys, where intersection is
// defined as keys that exist in all m. In the case where duplicate keys exist
// across maps, the value corresponding to the key in the first map is used. It
// always returns an allocated map, even if the intersection is the empty set.
//
// It does not modify any of the given inputs, but also does not deep copy any
// values. That means the returned map may have keys and values that point to
// the same objects as in the original map.
func IntersectMapKeys[K comparable, V any](maps ...map[K]V) map[K]V {
	if len(maps) == 0 {
		return make(map[K]V)
	}

	// The only way we can do better than nested for loops in big-O runtime and
	// space usage would be to get fancy and do a hash join or index join. In
	// practice that would probably be slower for modestly-sized inputs.

	out := gomaps.Clone(maps[0])
	for _, m := range maps[1:] {
		for k := range out {
			if _, ok := m[k]; !ok {
				delete(out, k)
			}
		}
	}

	return out
}

// UnionMapKeys finds the union of all m, where union is defined as the
// combination of all keys in all m. In the case where duplicate keys exist
// across maps, the value corresponding to the key in the first map is used. It
// always returns an allocated map, even if the union is the empty set.
//
// It does not modify any of the given inputs, but also does not deep copy any
// values. That means the returned map may have keys and values that point to
// the same objects as in the original map.
func UnionMapKeys[K comparable, V any](maps ...map[K]V) map[K]V {
	if len(maps) == 0 {
		return make(map[K]V)
	}

	// Pre-compute the maximum possible allocation to minimize allocs.
	var alloc int
	for _, m := range maps {
		alloc += len(m)
	}
	final := make(map[K]V, alloc)

	// Iterate over all maps and combine elements.
	for _, m := range maps {
		for k, v := range m {
			// Only overwrite if we haven't previously seen the value.
			if _, ok := final[k]; !ok {
				final[k] = v
			}
		}
	}

	return final
}

// SubtractMapKeys finds the differece (subtraction), where difference is
// defined as the removal of all keys from m0 that exist in mn. It always
// returns an allocated map, even if the difference is the empty set.
//
// It does not modify any of the given inputs, but also does not deep copy any
// values. That means the returned map may have keys and values that point to
// the same objects as in the original map.
func SubtractMapKeys[K comparable, V any](maps ...map[K]V) map[K]V {
	if len(maps) == 0 {
		return make(map[K]V)
	}

	// Pre-compute the maximum possible allocation to minimize allocs.
	initialMap := maps[0]
	final := make(map[K]V, len(initialMap))
	for k, v := range initialMap {
		final[k] = v
	}

	for _, m := range maps[1:] {
		// Short-circuit: if the map is empty, there's nothing left to subtract.
		if len(final) == 0 {
			return make(map[K]V)
		}

		for k := range m {
			delete(final, k)
		}
	}

	return final
}
