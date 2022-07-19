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

// Package cache implements an inmemory cache for any object.
//
// This package was created and adapted from:
//
//     https://github.com/google/exposure-notifications-server/blob/main/pkg/cache/cache.go
//
// This package assumes the system time has minimal skew. In case of major clock
// skew or system clock reset, cache expirations could occur out of order.
package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// Func is a generic-based function that returns T, or an error if creating T
// failed. This function is used as part of the WriteThruCache call.
type Func[T any] func() (T, error)

// Cache represents a generic cacher. All items in the cache must be of the same
// type T, and all items in the cache share the same expiration duration
// (expiration is not configurable per-item).
//
// For performance, it's strongly recommended that you store pointers to objects
// instead of actual objects.
type Cache[T any] struct {
	// data is the actual internal cache storage.
	data map[string]*cacheListItem[T]

	// head points to the head of the linked list, tail points to the tail.
	head, tail *cacheListItem[T]

	// expireAfter is the global TTL value.
	expireAfter time.Duration

	// stopped indicates whether the cache is stopped. stopCh is a channel used to
	// control cancellation.
	stopped uint32
	stopCh  chan struct{}

	// mu is the internal lock to allow for concurrent operations.
	mu sync.RWMutex
}

// New creates a new in memory cache. Panics if expireAfter is 0 or negative.
func New[T any](expireAfter time.Duration) *Cache[T] {
	if expireAfter <= 0 {
		panic("expireAfter duration must be positive")
	}

	c := &Cache[T]{
		data:        make(map[string]*cacheListItem[T]),
		expireAfter: expireAfter,
		stopCh:      make(chan struct{}),
	}

	// Start the sweep, with a minimum sweep of 50ms.
	sweep := expireAfter / 4.0
	if min := 50 * time.Millisecond; sweep < min {
		sweep = min
	}
	go c.start(sweep)

	return c
}

// Size returns the current number of items in the cache.
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isStopped() {
		panic("cache is stopped")
	}

	return len(c.data)
}

// Clear removes all items from the cache, regardless of their expiration. Note
// this is different from Stop() which deletes all cached items and prevents new
// items from being added.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isStopped() {
		panic("cache is stopped")
	}

	c.clear()
}

// clear removes all items from the cache, zeroing out as many items as possible
// for efficient GC. Callers must check if the cache is stopped and acquire a
// full lock before calling this function.
func (c *Cache[T]) clear() {
	var zeroV T

	for k, v := range c.data {
		v.key = nil
		v.value = zeroV
		v.expiresAt = nil
		delete(c.data, k)
	}
	c.data = nil

	node := c.head
	for node != nil {
		node.key = nil
		node.value = zeroV
		node.prev = nil
		node, node.next = node.next, nil
	}
	c.head = nil
	c.tail = nil
}

// WriteThruLookup checks the cache for the value associated with name, and if
// not found or expired, invokes the provided lookup function to resolve the
// value.
func (c *Cache[T]) WriteThruLookup(name string, fn Func[T]) (T, error) {
	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isStopped() {
		panic("cache is stopped")
	}

	if v, ok := c.lookup(name, now); ok {
		return v, nil
	}

	v, err := fn()
	if err != nil {
		var zeroV T
		return zeroV, err
	}

	c.set(name, v, now)
	return v, nil
}

// Lookup checks the cache for a non-expired object by the supplied key name.
// The bool return informs the caller if there was a cache hit or not.
// A return of nil, true means that nil is in the cache.
// Where nil, false indicates a cache miss or that the value is expired and should
// be refreshed.
func (c *Cache[T]) Lookup(name string) (T, bool) {
	now := time.Now().UTC()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isStopped() {
		panic("cache is stopped")
	}

	return c.lookup(name, now)
}

// lookup is the internal implementation of Lookup. Callers are responsible for
// acquring a lock and checking whether the cache is stopped.
func (c *Cache[T]) lookup(name string, now time.Time) (T, bool) {
	v, ok := c.data[name]
	if !ok || v.expiresAt.Before(now) {
		var zeroV T
		return zeroV, false
	}
	return v.value, true
}

// Set saves the current value of an object in the cache, with the supplied
// durintion until the object expires.
func (c *Cache[T]) Set(name string, object T) {
	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isStopped() {
		panic("cache is stopped")
	}

	c.set(name, object, now)
}

func (c *Cache[T]) set(name string, object T, now time.Time) {
	// Calculate expiration after acquiring a lock. The item is valid from when
	// insertion started, not when insertion finishes.
	exp := now.Add(c.expireAfter)

	node, ok := c.data[name]
	if !ok {
		node = &cacheListItem[T]{
			key: &name,
		}
		c.data[name] = node
	}
	node.value = object
	node.expiresAt = &exp

	// Remove the item from the list.
	if node == c.head {
		c.head = node.next
	}
	if node == c.tail {
		c.tail = node.prev
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	node.next = nil
	node.prev = nil

	// If this is the first entry in the cache, update the head.
	if c.head == nil {
		c.head = node
	}

	// Move to the end of the list.
	if c.tail != nil {
		c.tail.next = node
		node.prev = c.tail
	}
	c.tail = node
}

// Stop clears the cache and prevents new entries from being added and
// retrieved.
func (c *Cache[T]) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !atomic.CompareAndSwapUint32(&c.stopped, 0, 1) {
		return
	}
	close(c.stopCh)

	c.clear()
}

// start begins the background reaping process for expired entries. It runs
// until stopped via Stop() and is intended to be called as a goroutine.
func (c *Cache[T]) start(sweep time.Duration) {
	ticker := time.NewTicker(sweep)
	defer ticker.Stop()

	for {
		// Check if we're stopped first to prevent entering a race between a short
		// time ticker and the stop channel.
		if c.isStopped() {
			return
		}

		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			func() {
				now := time.Now().UTC()

				c.mu.Lock()
				defer c.mu.Unlock()

				c.cleanUntil(now)
			}()
		}
	}
}

// cleanUntil deletes entries from the linked list and map until the expiration
// is greater than the given time.
func (c *Cache[T]) cleanUntil(when time.Time) {
	// Walk the LinkedList from the front, since those are the oldest items.
	node := c.head
	for node != nil {
		// If this item isn't a candidate for expiration, then no future items
		// will be a candidate either, since they are in increasing order.
		if node.expiresAt.After(when) {
			break
		}

		delete(c.data, *node.key)

		var zeroV T
		node.key = nil
		node.value = zeroV
		node.expiresAt = nil
		node.prev = nil
		node, node.next = node.next, nil
	}

	c.head = node
	if node != nil {
		// reset if the node pointed to its previous; that node is now gone.
		node.prev = nil
	} else {
		// deleted everything, delete the tail too.
		c.tail = nil
	}
}

// isStopped is a helper for checking if the queue is stopped.
func (c *Cache[T]) isStopped() bool {
	return atomic.LoadUint32(&c.stopped) == 1
}

// cacheListItem represents an entry in the linked list.
type cacheListItem[T any] struct {
	next, prev *cacheListItem[T]
	key        *string
	value      T
	expiresAt  *time.Time
}
