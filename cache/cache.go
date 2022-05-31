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
	data        map[string]item[T]
	expireAfter time.Duration
	mu          sync.RWMutex
}

// item is an internal representation of a cached item. It stores the actual
// object and the upcoming expiration time.
type item[T any] struct {
	object    T
	expiresAt int64
}

// expired returns true if the given item has expired, or false otherwise.
func (c *item[T]) expired() bool {
	return c.expiresAt < time.Now().UnixNano()
}

// New creates a new in memory cache. Panics if expireAfter is 0 or negative.
func New[T any](expireAfter time.Duration) *Cache[T] {
	if expireAfter <= 0 {
		panic("expireAfter duration must be positive")
	}

	return &Cache[T]{
		data:        make(map[string]item[T]),
		expireAfter: expireAfter,
	}
}

// Removes an item by name and expiry time when the purge was scheduled.
// If there is a race, and the item has been refreshed, it will not be purged.
func (c *Cache[T]) purgeExpired(name string, expectedExpiryTime int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.data[name]; ok && item.expiresAt == expectedExpiryTime {
		// found, and the expiry time is still the same as when the purge was requested.
		delete(c.data, name)
	}
}

// Size returns the number of items in the cache.
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Clear removes all items from the cache, regardless of their expiration.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]item[T])
}

// WriteThruLookup checks the cache for the value associated with name,
// and if not found or expired, invokes the provided primaryLookup function
// to local the value.
func (c *Cache[T]) WriteThruLookup(name string, primaryLookup Func[T]) (T, error) {
	var nilT T

	c.mu.RLock()
	val, hit := c.lookup(name)
	if hit {
		c.mu.RUnlock()
		return val, nil
	}
	c.mu.RUnlock()

	// Ensure the value hasn't been set by another goroutine by escalating to a RW
	// lock. We need the W lock anyway if we're about to write.
	c.mu.Lock()
	defer c.mu.Unlock()
	val, hit = c.lookup(name)
	if hit {
		return val, nil
	}

	// If we got this far, it was either a miss, or hit w/ expired value, execute
	// the function.

	// Value does indeed need to be refreshed. Used the provided function.
	newData, err := primaryLookup()
	if err != nil {
		return nilT, err
	}

	// save the newData in the cache. newData may be nil, if that's what the WriteThruFunction provided.
	c.data[name] = item[T]{
		object:    newData,
		expiresAt: time.Now().Add(c.expireAfter).UnixNano(),
	}
	return newData, nil
}

// Lookup checks the cache for a non-expired object by the supplied key name.
// The bool return informs the caller if there was a cache hit or not.
// A return of nil, true means that nil is in the cache.
// Where nil, false indicates a cache miss or that the value is expired and should
// be refreshed.
func (c *Cache[T]) Lookup(name string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lookup(name)
}

// Set saves the current value of an object in the cache, with the supplied
// durintion until the object expires.
func (c *Cache[T]) Set(name string, object T) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[name] = item[T]{
		object:    object,
		expiresAt: time.Now().Add(c.expireAfter).UnixNano(),
	}

	return nil
}

// lookup finds an unexpired item at the given name. The bool indicates if a hit
// occurred. This is an internal API that is NOT thread-safe. Consumers must
// take out a read or read-write lock.
func (c *Cache[T]) lookup(name string) (T, bool) {
	var nilT T
	if item, ok := c.data[name]; ok && item.expired() {
		// Cache hit, but expired. The removal from the cache is deferred.
		go c.purgeExpired(name, item.expiresAt)
		return nilT, false
	} else if ok {
		// Cache hit, not expired.
		return item.object, true
	}

	// Cache miss.
	return nilT, false
}