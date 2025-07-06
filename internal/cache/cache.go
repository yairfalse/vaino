package cache

import (
	"sync"
	"time"
)

// Manager interface for cache operations
type Manager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear() error
	Stats() Stats
}

// Stats provides cache statistics
type Stats struct {
	Hits      int64
	Misses    int64
	Size      int64
	Evictions int64
}

// Config holds cache configuration
type Config struct {
	MemorySizeMB    int
	DiskSizeGB      int
	DefaultTTL      string
	CleanupInterval string
}

// NewManager creates a new cache manager
func NewManager(config Config) (Manager, error) {
	cache := &MemoryCache{
		items: make(map[string]*cacheItem),
		stats: Stats{},
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	return cache, nil
}

// NewMemoryCache creates a simple memory cache for testing
func NewMemoryCache() Manager {
	return &MemoryCache{
		items: make(map[string]*cacheItem),
		stats: Stats{},
	}
}

type cacheItem struct {
	value  interface{}
	expiry time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	stats Stats
}

func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	item, exists := mc.items[key]
	if !exists {
		mc.stats.Misses++
		return nil, false
	}
	
	if !item.expiry.IsZero() && time.Now().After(item.expiry) {
		mc.stats.Misses++
		delete(mc.items, key)
		return nil, false
	}
	
	mc.stats.Hits++
	return item.value, true
}

func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}
	
	mc.items[key] = &cacheItem{
		value:  value,
		expiry: expiry,
	}
}

func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	delete(mc.items, key)
}

func (mc *MemoryCache) Clear() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	evicted := int64(len(mc.items))
	mc.items = make(map[string]*cacheItem)
	mc.stats.Evictions += evicted
	
	return nil
}

func (mc *MemoryCache) Stats() Stats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	stats := mc.stats
	stats.Size = int64(len(mc.items))
	return stats
}

// cleanup periodically removes expired items
func (mc *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.mu.Lock()
		now := time.Now()
		var evicted int64
		
		for key, item := range mc.items {
			if !item.expiry.IsZero() && now.After(item.expiry) {
				delete(mc.items, key)
				evicted++
			}
		}
		
		mc.stats.Evictions += evicted
		mc.mu.Unlock()
	}
}