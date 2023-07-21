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

// Intersect finds the intersection of all slices, where intersection is defined
// as elements that exist in all slices. The elements in the returned slice will
// be in an undefined order - use [IntersectStable] to trade efficiency for
// ordering. Duplicate elements are removed. This function always returns an
// allocated slice, even if the intersection is the empty set.
//
// It does not modify any of the given slices, but also does not deep copy any
// values. That means the returned slice may have elements that point to the
// same objects as in the original slice.
func Intersect[T comparable](slices ...[]T) []T {
	if len(slices) == 0 {
		return []T{}
	}

	maps := make([]map[T]struct{}, len(slices))
	for i, s := range slices {
		// Short-circuit: if any of the slices are the empty set, we know the
		// intersection is the empty set.
		if len(s) == 0 {
			return []T{}
		}

		m := make(map[T]struct{}, len(s))
		for _, v := range s {
			m[v] = struct{}{}
		}
		maps[i] = m
	}

	result := IntersectMapKeys(maps...)
	final := make([]T, 0, len(result))
	for k := range result {
		final = append(final, k)
	}
	return final
}

// IntersectStable has the same invariants [Intersect], except it preserves the
// relative order of elements in the intersection. This function has a
// comparatively higher runtime complexity than [Intersect].
func IntersectStable[T comparable](slices ...[]T) []T {
	if len(slices) == 0 {
		return []T{}
	}

	// Make a copy of the first slice; we're going to gradually remove elements.
	final := append([]T{}, slices[0]...)

	// Remove duplicates
	finalSeen := make(map[T]struct{}, len(final))
	var i int
	for _, v := range final {
		if _, ok := finalSeen[v]; !ok {
			finalSeen[v] = struct{}{}
			final[i] = v
			i++
		}
	}
	for j := i; j < len(final); j++ {
		final[j] = *new(T)
	}
	final = final[:i:len(final)]

	for _, s := range slices[1:] {
		// Short-circuit: if our intersection is already the empty set, we can
		// return now.
		if len(final) == 0 {
			return final
		}

		var i int
		for _, v := range final {
			if sliceContains(s, v) {
				final[i] = v
				i++
			}
		}
		// Delete unused elements, nil out unused space so we don't leak.
		for j := i; j < len(final); j++ {
			final[j] = *new(T)
		}
		final = final[:i:len(final)]
	}
	return final
}

// Union finds the union of all slices, where union is defined as the
// combination of all elements in all slices. The elements are returned in the
// order in which they first appear in the slices. Duplicate elements are
// removed. This function always returns an allocated slice, even if the union
// is the empty set.
//
// It does not modify any of the given slices, but also does not deep copy any
// values. That means the returned slice may have elements that point to the
// same objects as in the original slice.
func Union[T comparable](slices ...[]T) []T {
	if len(slices) == 0 {
		return []T{}
	}

	// Pre-compute the maximum possible allocation to minimize allocs.
	var alloc int
	for _, s := range slices {
		alloc += len(s)
	}
	final := make([]T, 0, alloc)
	seen := make(map[T]struct{}, alloc)

	// Iterate over all maps and combine elements.
	for _, s := range slices {
		for _, v := range s {
			if _, ok := seen[v]; ok {
				// We already saw this element
				continue
			}

			final = append(final, v)
			seen[v] = struct{}{}
		}
	}
	return final[:len(final):len(final)]
}

// Subtract finds the differece (subtraction), where difference is defined as
// the removal of all elements from s0 that exist in sn. The elements are
// returned in the order in which they appeared in s0. This function always
// returns an allocated slice, even if the difference is the empty set.
//
// All duplicate elements that match a subtraction are removed. Duplicate
// elements that do not match a subtraction are preserved.
//
// It does not modify any of the given slices, but also does not deep copy any
// values. That means the returned slice may have elements that point to the
// same objects as in the original slice.
func Subtract[T comparable](slices ...[]T) []T {
	if len(slices) == 0 {
		return []T{}
	}

	// Make a copy of the first slice; we're going to gradually remove elements.
	final := append([]T{}, slices[0]...)

	var alloc int
	for _, s := range slices[1:] {
		alloc += len(s)
	}
	toDelete := make(map[T]struct{}, alloc)
	for _, s := range slices[1:] {
		for _, k := range s {
			toDelete[k] = struct{}{}
		}
	}

	var i int
	for _, v := range final {
		if _, ok := toDelete[v]; !ok {
			final[i] = v
			i++
		}
	}
	// Delete unused elements, nil out unused space so we don't leak.
	for j := i; j < len(final); j++ {
		final[j] = *new(T)
	}
	final = final[:i:len(final)]
	return final
}

func sliceIndex[T comparable](haystack []T, needle T) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}

func sliceContains[T comparable](haystack []T, needle T) bool {
	return sliceIndex(haystack, needle) != -1
}
