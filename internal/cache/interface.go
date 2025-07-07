package cache

import (
	"time"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Get retrieves a value from the cache
	Get(key string) (interface{}, bool)
	
	// Set stores a value in the cache with TTL
	Set(key string, value interface{}, ttl time.Duration) error
	
	// Delete removes a value from the cache
	Delete(key string) error
	
	// Clear removes all values from the cache
	Clear() error
	
	// Keys returns all cache keys (useful for debugging)
	Keys() []string
	
	// Size returns the number of items in cache
	Size() int
	
	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits     int64 `json:"hits"`
	Misses   int64 `json:"misses"`
	Sets     int64 `json:"sets"`
	Deletes  int64 `json:"deletes"`
	Evictions int64 `json:"evictions"`
	Size     int   `json:"size"`
	HitRatio float64 `json:"hit_ratio"`
}

// CacheItem represents an item stored in the cache
type CacheItem struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
}

// Config holds cache configuration
type Config struct {
	// MaxItems is the maximum number of items to store
	MaxItems int `json:"max_items"`
	
	// DefaultTTL is the default time-to-live for cache items
	DefaultTTL time.Duration `json:"default_ttl"`
	
	// CleanupInterval is how often to clean up expired items
	CleanupInterval time.Duration `json:"cleanup_interval"`
	
	// PersistToDisk enables disk persistence
	PersistToDisk bool `json:"persist_to_disk"`
	
	// PersistPath is the directory for disk cache files
	PersistPath string `json:"persist_path"`
}

// DefaultConfig returns a reasonable default cache configuration
func DefaultConfig() Config {
	return Config{
		MaxItems:        1000,
		DefaultTTL:      1 * time.Hour,
		CleanupInterval: 10 * time.Minute,
		PersistToDisk:   false,
		PersistPath:     "",
	}
}