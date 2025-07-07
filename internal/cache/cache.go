package cache

import (
	"sync"
	"time"
)

type Manager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear() error
	Stats() Stats
}

type Stats struct {
	Hits   int64
	Misses int64
	Size   int64
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	stats Stats
}

type cacheItem struct {
	value  interface{}
	expiry time.Time
}

func NewManager() Manager {
	return &MemoryCache{
		items: make(map[string]*cacheItem),
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

	if !item.expiry.IsZero() && time.Now().After(item.expiry) {
		c.stats.Misses++
		delete(c.items, key)
		return nil, false
	}

	c.stats.Hits++
	return item.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}

	c.items[key] = &cacheItem{
		value:  value,
		expiry: expiry,
	}
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	return nil
}

func (c *MemoryCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = int64(len(c.items))
	return stats
}