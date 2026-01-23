package amadeus

import (
	"fmt"
	"sync"
	"time"
)

// SimpleCache is a basic thread-safe in-memory cache
type SimpleCache struct {
	data map[string]cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiryTime time.Time
}

// NewSimpleCache creates a new cache instance
func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		data: make(map[string]cacheItem),
	}
}

// Get retrieves a value from the cache
func (c *SimpleCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found {
		return nil, false
	}

	if time.Now().After(item.expiryTime) {
		return nil, false
	}

	return item.value, true
}

// Set adds a value to the cache with a TTL
func (c *SimpleCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheItem{
		value:      value,
		expiryTime: time.Now().Add(ttl),
	}
}

// GenerateCacheKey creates a unique key for caching based on inputs
func GenerateCacheKey(prefix string, params ...interface{}) string {
	return fmt.Sprintf("%s:%v", prefix, params)
}
