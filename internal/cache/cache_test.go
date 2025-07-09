package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewManager()

	// Test setting and getting a value
	cache.Set("key1", "value1", time.Hour)
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1 in cache")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewManager()

	// Set value with very short TTL
	cache.Set("key2", "value2", time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	_, found := cache.Get("key2")
	if found {
		t.Error("Expected key2 to be expired")
	}
}

func TestManager_Delete(t *testing.T) {
	cache := NewManager()

	cache.Set("key3", "value3", time.Hour)
	cache.Delete("key3")

	_, found := cache.Get("key3")
	if found {
		t.Error("Expected key3 to be deleted")
	}
}

func TestManager_Stats(t *testing.T) {
	cache := NewManager()

	// Test initial stats
	stats := cache.Stats()
	if stats.Size != 0 || stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Initial stats should be zero")
	}

	// Add item and test stats
	cache.Set("key4", "value4", time.Hour)

	// Test hit
	cache.Get("key4")
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	// Test miss
	cache.Get("nonexistent")
	stats = cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestManager_Clear(t *testing.T) {
	cache := NewManager()

	cache.Set("key5", "value5", time.Hour)
	cache.Set("key6", "value6", time.Hour)

	err := cache.Clear()
	if err != nil {
		t.Errorf("Clear should not return error: %v", err)
	}

	_, found := cache.Get("key5")
	if found {
		t.Error("Expected cache to be cleared")
	}
}
