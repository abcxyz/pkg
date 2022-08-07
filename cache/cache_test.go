// Copyright 2022 The Authors (see AUTHORS file)
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

package cache

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type order struct {
	Burgers int
	Fries   int
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("new", func(t *testing.T) {
		t.Parallel()

		cache := New[*order](1 * time.Second)
		defer cache.Stop()

		if got, want := cache.Size(), 0; got != want {
			t.Errorf("expected %d to be %d", got, want)
		}
	})

	t.Run("panic_on_negative", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()

		cache := New[*order](-1 * time.Second)
		defer cache.Stop()

		t.Fatal("expected test to fail")
	})
}

func TestCache_Size(t *testing.T) {
	t.Parallel()

	cache := New[string](30 * time.Second)
	defer cache.Stop()

	cache.Set("foo", "bar")
	if got, hit := cache.Lookup("foo"); got == "" || !hit {
		t.Fatalf("lookup failed got %#v", got)
	}
	if got, want := cache.Size(), 1; got != want {
		t.Errorf("expected %d to be %d", got, want)
	}
}

func TestCache_Clear(t *testing.T) {
	t.Parallel()

	cache := New[string](30 * time.Second)
	defer cache.Stop()

	cache.Set("foo", "bar")
	cache.Clear()

	if got, ok := cache.Lookup("foo"); ok {
		t.Fatalf("lookup failed expected nil got %#v", got)
	}

	if got := cache.head; got != nil {
		t.Errorf("expected head to be nil, got %#v", got)
	}
	if got := cache.tail; got != nil {
		t.Errorf("expected tail to be nil, got %#v", got)
	}
	if got := len(cache.data); got != 0 {
		t.Errorf("expected map to be empty, got %#v", got)
	}
}

func TestCache_WriteThruLookup(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		cache := New[*order](time.Second)
		defer cache.Stop()

		lookupCount := 0
		want := &order{12, 34}
		lookerUpper := func() (*order, error) {
			lookupCount++
			return want, nil
		}

		for i := 0; i < 2; i++ {
			got, err := cache.WriteThruLookup("foo", lookerUpper)
			if err != nil {
				t.Fatalf("unexpected error on WriteThruLookup: %v", err)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		}

		if lookupCount != 1 {
			t.Fatalf("incorrect lookup count, want: 1, got: %v", lookupCount)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		cache := New[*order](time.Second)
		defer cache.Stop()

		lookerUpper := func() (*order, error) {
			return nil, fmt.Errorf("nope")
		}

		got, err := cache.WriteThruLookup("foo", lookerUpper)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "nope" {
			t.Errorf("incorrect error, want: `nope` got: %v", err.Error())
		}
		if got != nil {
			t.Errorf("unexpected cached item, want: nil, got %v", got)
		}
	})
}

func TestCache_Lookup(t *testing.T) {
	t.Parallel()

	t.Run("exists", func(t *testing.T) {
		t.Parallel()

		cache := New[string](30 * time.Second)
		defer cache.Stop()

		cache.Set("foo", "bar")
		if got, ok := cache.Lookup("foo"); got == "" || !ok {
			t.Errorf("lookup failed got %#v (%t)", got, ok)
		}
	})

	t.Run("no_exist", func(t *testing.T) {
		t.Parallel()

		cache := New[string](30 * time.Second)
		defer cache.Stop()

		if got, ok := cache.Lookup("baz"); ok {
			t.Errorf("expected lookup to fail, got %v", got)
		}
	})
}

func TestCache_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets", func(t *testing.T) {
		t.Parallel()

		cache := New[string](30 * time.Second)
		defer cache.Stop()

		cache.Set("foo", "bar")
		if got, _ := cache.Lookup("foo"); got != "bar" {
			t.Errorf("expected %q to be %q", got, "bar")
		}
	})

	t.Run("overwrites", func(t *testing.T) {
		t.Parallel()

		cache := New[string](30 * time.Second)
		defer cache.Stop()

		cache.Set("foo", "bar")
		if got, _ := cache.Lookup("foo"); got != "bar" {
			t.Errorf("expected %q to be %q", got, "bar")
		}

		cache.Set("foo", "baz")
		if got, _ := cache.Lookup("foo"); got != "baz" {
			t.Errorf("expected %q to be %q", got, "baz")
		}
	})

	t.Run("moves_to_tail", func(t *testing.T) {
		t.Parallel()

		cache := New[int](30 * time.Second)
		defer cache.Stop()

		cache.Set("foo", 1)
		if got, want := testPrintListFront(cache.head), "foo (1)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "foo (1)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		cache.Set("bar", 2)
		if got, want := testPrintListFront(cache.head), "foo (1) -> bar (2)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "bar (2) -> foo (1)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		cache.Set("baz", 3)
		if got, want := testPrintListFront(cache.head), "foo (1) -> bar (2) -> baz (3)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "baz (3) -> bar (2) -> foo (1)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Test moving head to tail.
		cache.Set("foo", 4)
		if got, want := testPrintListFront(cache.head), "bar (2) -> baz (3) -> foo (4)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "foo (4) -> baz (3) -> bar (2)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Test moving something in the middle to tail.
		cache.Set("baz", 5)
		if got, want := testPrintListFront(cache.head), "bar (2) -> foo (4) -> baz (5)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "baz (5) -> foo (4) -> bar (2)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Test moving tail to tail.
		cache.Set("baz", 6)
		if got, want := testPrintListFront(cache.head), "bar (2) -> foo (4) -> baz (6)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "baz (6) -> foo (4) -> bar (2)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
	})
}

func TestCache_Concurrent(t *testing.T) {
	t.Parallel()

	cache := New[*order](time.Second * 5)
	defer cache.Stop()

	lookupCount := 0
	want := &order{12, 34}
	lookerUpper := func() (*order, error) {
		// The sleep here, reliably triggers a race condition on multiple entrants attempting
		// to lookup the cache miss to primary storage. Only one will win!
		time.Sleep(250 * time.Millisecond)
		lookupCount++
		return want, nil
	}

	parallel := 10
	done := make(chan error, parallel)
	for i := 0; i < parallel; i++ {
		ver := i
		go func() {
			got, err := cache.WriteThruLookup("foo", lookerUpper)
			if err != nil {
				done <- fmt.Errorf("routine: %v got unexpected error: %w", ver, err)
				return
			}
			if diff := cmp.Diff(want, got); diff != "" {
				done <- fmt.Errorf("routine: %v mismatch (-want, +got):\n%s", ver, diff)
			}
			done <- nil
		}()
	}

	for i := 0; i < parallel; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("goroutines did not terminate fast enough")
		}
	}

	if lookupCount != 1 {
		t.Errorf("unexpected lookupCount, want: 1, got: %v", lookupCount)
	}
}

func TestTTL_Stop(t *testing.T) {
	t.Parallel()

	t.Run("deletes_all_entries", func(t *testing.T) {
		t.Parallel()

		cache := New[int](5 * time.Minute)
		cache.Set("foo", 5)
		cache.Set("bar", 10)
		cache.Set("baz", 15)

		cache.Stop()

		if cache.data != nil {
			t.Errorf("expected %#v to be nil", cache.data)
		}
	})

	t.Run("panics_writethrulookup", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "cache is stopped"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := New[int](5 * time.Minute)
		cache.Stop()
		if _, err := cache.WriteThruLookup("foo", func() (int, error) {
			return 5, nil
		}); err != nil {
			t.Fatal(err)
		}
		t.Errorf("did not panic")
	})

	t.Run("panics_lookup", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "cache is stopped"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := New[int](5 * time.Minute)
		cache.Stop()
		cache.Lookup("foo")
		t.Errorf("did not panic")
	})

	t.Run("panics_set", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "cache is stopped"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := New[int](5 * time.Minute)
		cache.Stop()
		cache.Set("foo", 5)
		t.Errorf("did not panic")
	})
}

func TestCache_Expires(t *testing.T) {
	t.Parallel()

	t.Run("after_duration", func(t *testing.T) {
		t.Parallel()

		cache := New[string](50 * time.Millisecond)
		defer cache.Stop()

		cache.Set("foo", "bar")
		if got, _ := cache.Lookup("foo"); got != "bar" {
			t.Errorf("expected %q to be %q", got, "bar")
		}

		time.Sleep(55 * time.Millisecond) // allow for some clock skew

		if got, ok := cache.Lookup("foo"); ok {
			t.Errorf("expected %q to not exist", got)
		}
	})

	t.Run("after_duration", func(t *testing.T) {
		t.Parallel()

		// Cache time doesn't really matter because we're manually invoking.
		cache := New[string](30 * time.Minute)
		defer cache.Stop()

		// Manually override expireAfter to get deterministic times.
		cache.expireAfter = 0

		now := time.Unix(0, 0).UTC()
		cache.set("foo", "bar", now)
		cache.set("baz", "bap", now.Add(5*time.Second))
		cache.set("apple", "banana", now.Add(10*time.Second))
		cache.set("kiwi", "pear", now.Add(10*time.Second))

		if got, want := testPrintListFront(cache.head), "foo (bar) -> baz (bap) -> apple (banana) -> kiwi (pear)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "kiwi (pear) -> apple (banana) -> baz (bap) -> foo (bar)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Cleaning in the past shouldn't do anything.
		cache.cleanUntil(now.Add(-5 * time.Second))
		if got, want := testPrintListFront(cache.head), "foo (bar) -> baz (bap) -> apple (banana) -> kiwi (pear)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "kiwi (pear) -> apple (banana) -> baz (bap) -> foo (bar)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Cleaning after a few seconds should clear the first entry.
		cache.cleanUntil(now.Add(3 * time.Second))
		if got, want := testPrintListFront(cache.head), "baz (bap) -> apple (banana) -> kiwi (pear)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "kiwi (pear) -> apple (banana) -> baz (bap)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Cleaning after a few seconds should clear the next entry.
		cache.cleanUntil(now.Add(7 * time.Second))
		if got, want := testPrintListFront(cache.head), "apple (banana) -> kiwi (pear)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), "kiwi (pear) -> apple (banana)"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		// Cleaning after a long time should remove all entries
		cache.cleanUntil(now.Add(15 * time.Second))
		if got, want := testPrintListFront(cache.head), ""; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := testPrintListBack(cache.tail), ""; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got := cache.head; got != nil {
			t.Errorf("expected head to be nil, got %#v", got)
		}
		if got := cache.tail; got != nil {
			t.Errorf("expected tail to be nil, got %#v", got)
		}
		if got, want := len(cache.data), 0; got != want {
			t.Errorf("expected %#v to be empty", cache.data)
		}
	})
}

func testPrintListFront[T any](node *cacheListItem[T]) string {
	list := make([]string, 0)
	for node != nil {
		list = append(list, fmt.Sprintf("%s (%v)", *node.key, node.value))
		node = node.next
	}
	return strings.Join(list, " -> ")
}

func testPrintListBack[T any](node *cacheListItem[T]) string {
	list := make([]string, 0)
	for node != nil {
		list = append(list, fmt.Sprintf("%s (%v)", *node.key, node.value))
		node = node.prev
	}
	return strings.Join(list, " -> ")
}
