package cache

import (
	"sync"
	"time"
)

type Stats struct {
	Items  int64
	Hits   int64
	Misses int64
}

type Manager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear() error
	Stats() Stats
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	stats Stats
}

type cacheItem struct {
	value  interface{}
	expiry time.Time
}

func NewManager() Manager {
	return &MemoryCache{
		items: make(map[string]cacheItem),
		stats: Stats{},
	}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	if time.Now().After(item.expiry) {
		c.stats.Misses++
		return nil, false
	}

	c.stats.Hits++
	return item.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; !exists {
		c.stats.Items++
	}

	c.items[key] = cacheItem{
		value:  value,
		expiry: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; exists {
		delete(c.items, key)
		c.stats.Items--
	}
}

func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
	c.stats = Stats{}
	return nil
}

func (c *MemoryCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.stats
}