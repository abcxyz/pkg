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

	// Pre-compute the maximum possible allocation to minimize allocs. Here we
	// find the map with the least number of elements and allocate a map of that
	// size, since we know that is the maximum possible intersection size.
	var smallestIdx int
	for i, m := range maps {
		// Short-circuit: if any of the maps are the empty set, we know the
		// intersection is the empty set.
		if len(m) == 0 {
			return make(map[K]V)
		}

		if len(m) < len(maps[smallestIdx]) {
			smallestIdx = i
		}
	}
	smallestMap := maps[smallestIdx]
	final := make(map[K]V, len(smallestMap))
	for k, v := range smallestMap {
		final[k] = v
	}

	// Compute the intersection.
	for i, m := range maps {
		// Short-circuit: we've already got the smallest possible intersection
		// (empty set), so there's no point in continuing.
		if len(final) == 0 {
			return make(map[K]V)
		}

		// This is us, ignore
		if i == smallestIdx {
			continue
		}

		// For each key in our smallest set, check if the key exists in the current
		// map. If it does not, remove it from the map since it's not part of the
		// intersection.
		for k := range final {
			if _, ok := m[k]; !ok {
				delete(final, k)
			}
		}
	}

	return final
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
