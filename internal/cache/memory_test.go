package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache()
	
	// Test basic set and get
	cache.Set("key1", "value1", time.Hour)
	
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}
}

func TestMemoryCache_GetNonExistent(t *testing.T) {
	cache := NewMemoryCache()
	
	value, found := cache.Get("nonexistent")
	if found {
		t.Error("Expected not to find nonexistent key")
	}
	if value != nil {
		t.Errorf("Expected nil value, got %v", value)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache()
	
	// Set with very short expiration
	cache.Set("expiring", "value", 10*time.Millisecond)
	
	// Should exist immediately
	value, found := cache.Get("expiring")
	if !found || value != "value" {
		t.Error("Expected to find key immediately after setting")
	}
	
	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	
	// Should be expired now
	value, found = cache.Get("expiring")
	if found {
		t.Error("Expected key to be expired")
	}
	if value != nil {
		t.Errorf("Expected nil value for expired key, got %v", value)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
	
	cache.Set("key1", "value1", time.Hour)
	cache.Delete("key1")
	
	value, found := cache.Get("key1")
	if found {
		t.Error("Expected key to be deleted")
	}
	if value != nil {
		t.Errorf("Expected nil value for deleted key, got %v", value)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache()
	
	cache.Set("key1", "value1", time.Hour)
	cache.Set("key2", "value2", time.Hour)
	
	err := cache.Clear()
	if err != nil {
		t.Errorf("Clear should not return error, got %v", err)
	}
	
	// Both keys should be gone
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	
	if found1 || found2 {
		t.Error("Expected all keys to be cleared")
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache()
	
	// Initial stats
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Size != 0 {
		t.Error("Expected zero initial stats")
	}
	
	// Add items and check stats
	cache.Set("key1", "value1", time.Hour)
	cache.Set("key2", "value2", time.Hour)
	
	stats = cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected size 2, got %d", stats.Size)
	}
	
	// Generate hits and misses
	cache.Get("key1")    // hit
	cache.Get("missing") // miss
	
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache()
	
	// Test concurrent set and get operations
	done := make(chan bool)
	
	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Set("key", i, time.Hour)
		}
		done <- true
	}()
	
	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("key")
		}
		done <- true
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Should not panic or deadlock
	stats := cache.Stats()
	if stats.Size != 1 {
		t.Errorf("Expected final size 1, got %d", stats.Size)
	}
}