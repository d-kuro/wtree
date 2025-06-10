// Package cache provides a generic caching mechanism.
package cache

import (
	"sync"
	"time"
)

// Entry represents a cached value with expiration time.
type Entry[T any] struct {
	Value      T
	ExpiresAt  time.Time
}

// Cache provides a thread-safe generic cache implementation.
type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]Entry[V]
	ttl     time.Duration
}

// New creates a new cache with the specified time-to-live for entries.
func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		entries: make(map[K]Entry[V]),
		ttl:     ttl,
	}
}

// Get retrieves a value from the cache.
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

// Set stores a value in the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = Entry[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[K]Entry[V])
}

// CleanExpired removes all expired entries from the cache.
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

// GetOrCompute retrieves a value from the cache or computes it if not present.
func (c *Cache[K, V]) GetOrCompute(key K, compute func() (V, error)) (V, error) {
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	value, err := compute()
	if err != nil {
		var zero V
		return zero, err
	}

	c.Set(key, value)
	return value, nil
}
