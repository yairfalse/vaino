package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MemoryCache implements the Cache interface using in-memory storage
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	config   Config
	stats    CacheStats
	stopChan chan bool
	stopped  bool
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(config Config) *MemoryCache {
	cache := &MemoryCache{
		items:    make(map[string]*CacheItem),
		config:   config,
		stats:    CacheStats{},
		stopChan: make(chan bool),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	// Load from disk if persistence is enabled
	if config.PersistToDisk && config.PersistPath != "" {
		cache.loadFromDisk()
	}

	return cache
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		// Item expired, remove it
		delete(c.items, key)
		c.stats.Misses++
		c.stats.Evictions++
		return nil, false
	}

	// Update access statistics
	item.AccessCount++
	item.LastAccess = time.Now()
	c.stats.Hits++

	return item.Value, true
}

// Set stores a value in the cache with TTL
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	// Check if we need to evict items due to size limit
	if len(c.items) >= c.config.MaxItems {
		c.evictOldest()
	}

	item := &CacheItem{
		Key:         key,
		Value:       value,
		ExpiresAt:   time.Now().Add(ttl),
		CreatedAt:   time.Now(),
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	c.items[key] = item
	c.stats.Sets++

	// Persist to disk if enabled
	if c.config.PersistToDisk {
		go c.persistToDisk()
	}

	return nil
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; exists {
		delete(c.items, key)
		c.stats.Deletes++
	}

	return nil
}

// Clear removes all values from the cache
func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
	
	// Reset stats except for historical counters
	c.stats.Size = 0

	return nil
}

// Keys returns all cache keys
func (c *MemoryCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// Size returns the number of items in cache
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.items)
	
	// Calculate hit ratio
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}

	return stats
}

// Close stops the cache and cleanup goroutines
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		close(c.stopChan)
		c.stopped = true

		// Persist to disk before closing
		if c.config.PersistToDisk {
			c.persistToDisk()
		}
	}

	return nil
}

// cleanupLoop periodically removes expired items
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

// cleanup removes expired items
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
			c.stats.Evictions++
		}
	}
}

// evictOldest removes the oldest accessed item
func (c *MemoryCache) evictOldest() {
	if len(c.items) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range c.items {
		if first || item.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.LastAccess
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.stats.Evictions++
	}
}

// persistToDisk saves cache to disk
func (c *MemoryCache) persistToDisk() {
	if c.config.PersistPath == "" {
		return
	}

	c.mu.RLock()
	items := make(map[string]*CacheItem)
	for k, v := range c.items {
		items[k] = v
	}
	c.mu.RUnlock()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(c.config.PersistPath, 0o755); err != nil {
		return
	}

	cacheFile := filepath.Join(c.config.PersistPath, "cache.json")
	data, err := json.Marshal(items)
	if err != nil {
		return
	}

	_ = os.WriteFile(cacheFile, data, 0o600)
}

// loadFromDisk loads cache from disk
func (c *MemoryCache) loadFromDisk() {
	if c.config.PersistPath == "" {
		return
	}

	cacheFile := filepath.Join(c.config.PersistPath, "cache.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return
	}

	var items map[string]*CacheItem
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range items {
		// Only load non-expired items
		if now.Before(item.ExpiresAt) {
			c.items[key] = item
		}
	}
}

// GenerateKey creates a cache key from components
func GenerateKey(prefix string, components ...string) string {
	key := prefix
	for _, component := range components {
		key += ":" + component
	}
	return key
}