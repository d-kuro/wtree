// Package cache provides a thread-safe generic caching mechanism with TTL (time-to-live) support.
// It supports automatic expiration of entries and provides methods for manual cleanup.
// The cache is implemented using Go generics for type safety.
package cache

import (
	"sync"
	"time"
)

// Entry represents a cached value with its expiration time.
// This is an internal structure used by the Cache to track when values expire.
type Entry[T any] struct {
	Value     T         // The cached value
	ExpiresAt time.Time // When this entry expires
}

// Cache provides a thread-safe generic cache implementation.
type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]Entry[V]
	ttl     time.Duration

	// computeMu protects the inFlight map to prevent race conditions during computation
	computeMu sync.Mutex
	inFlight  map[K]*sync.WaitGroup
}

// New creates a new cache with the specified default time-to-live for entries.
// The ttl parameter sets the default expiration duration for all entries added with Set().
// Individual entries can override this TTL by using SetWithTTL().
func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		entries:  make(map[K]Entry[V]),
		ttl:      ttl,
		inFlight: make(map[K]*sync.WaitGroup),
	}
}

// Get retrieves a value from the cache by key.
// Returns the value and true if found and not expired, otherwise returns the zero value and false.
// This method is thread-safe and uses a read lock for optimal concurrent access.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		var zero V
		return zero, false
	}

	return entry.Value, true
}

// Set stores a value in the cache with the default TTL.
// The value will expire after the TTL duration specified during cache creation.
// If a value with the same key already exists, it will be replaced.
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL.
// This allows overriding the default TTL for specific entries.
// The value will expire after the specified TTL duration from now.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = Entry[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache by key.
// No error is returned if the key doesn't exist.
// This operation is thread-safe and atomic.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache.
// This operation is thread-safe and atomic.
// After calling Clear, the cache will be empty regardless of expiration times.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.entries)
}

// CleanExpired removes all expired entries from the cache.
// This can be called periodically to free memory from expired entries.
// The method iterates through all entries and removes those past their expiration time.
func (c *Cache[K, V]) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// GetOrCompute retrieves a value from the cache or computes it if not present or expired.
// If the compute function returns an error, the value is not cached and the error is returned.
// This method is useful for lazy loading patterns where expensive computations should be cached.
// This method prevents race conditions by ensuring only one computation per key executes at a time.
func (c *Cache[K, V]) GetOrCompute(key K, compute func() (V, error)) (V, error) {
	// First, try to get from cache
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	// Use singleflight pattern to prevent duplicate computations
	c.computeMu.Lock()

	// Double-check pattern: value might have been computed while waiting for lock
	if value, ok := c.Get(key); ok {
		c.computeMu.Unlock()
		return value, nil
	}

	// Check if computation is already in flight
	if wg, exists := c.inFlight[key]; exists {
		c.computeMu.Unlock()
		wg.Wait() // Wait for ongoing computation

		// After waiting, try to get the value again
		if value, ok := c.Get(key); ok {
			return value, nil
		}
		// If still not found, the computation may have failed, so we'll compute again
	} else {
		// Start new computation
		wg := &sync.WaitGroup{}
		wg.Add(1)
		c.inFlight[key] = wg
		c.computeMu.Unlock()

		// Compute the value
		value, err := compute()

		// Clean up the in-flight entry
		c.computeMu.Lock()
		delete(c.inFlight, key)
		c.computeMu.Unlock()
		wg.Done()

		if err != nil {
			var zero V
			return zero, err
		}

		c.Set(key, value)
		return value, nil
	}

	// Fallback: compute again if value still not available
	value, err := compute()
	if err != nil {
		var zero V
		return zero, err
	}

	c.Set(key, value)
	return value, nil
}
