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

// Map returns a slice consisting of the results of applying the given function
// to the elements of the given slice.
func Map[T1, T2 any](slice []T1, mapper func(T1) T2) []T2 {
	if mapper == nil {
		panic("nil mapping function provided")
	}
	if slice == nil {
		return nil
	}
	result := make([]T2, len(slice))
	for index, value := range slice {
		result[index] = mapper(value)
	}
	return result
}

// Reduce performs a reduction on the elements of the given slice, using the
// given initial value and an accumulator function, and returns the reduced value.
func Reduce[T1, T2 any](slice []T1, initialValue T2, accumulator func(T2, T1) T2) T2 {
	if accumulator == nil {
		panic("nil accumulator function provided")
	}
	result := initialValue
	for _, value := range slice {
		result = accumulator(result, value)
	}
	return result
}

// Filter returns a slice consisting of the elements of the given slice that
// match the given predicate.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	if predicate == nil {
		panic("nil predicate provided")
	}
	if slice == nil {
		return nil
	}
	result := make([]T, 0)
	for _, value := range slice {
		if predicate(value) {
			result = append(result, value)
		}
	}
	return result
}
