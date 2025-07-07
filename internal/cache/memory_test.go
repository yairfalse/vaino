package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewMemoryCache(t *testing.T) {
	config := Config{
		MaxItems:        100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	if cache == nil {
		t.Fatal("expected cache to be created")
	}
	
	if cache.config.MaxItems != 100 {
		t.Errorf("expected MaxItems 100, got %d", cache.config.MaxItems)
	}
	
	// Test initial state
	if cache.Size() != 0 {
		t.Errorf("expected empty cache, got size %d", cache.Size())
	}
	
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("expected stats size 0, got %d", stats.Size)
	}
	
	// Cleanup
	cache.Close()
}

func TestMemoryCache_BasicOperations(t *testing.T) {
	config := Config{
		MaxItems:        10,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	defer cache.Close()
	
	// Test Set and Get
	err := cache.Set("key1", "value1", 0)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}
	
	value, exists := cache.Get("key1")
	if !exists {
		t.Error("expected key1 to exist")
	}
	
	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}
	
	// Test non-existent key
	_, exists = cache.Get("non-existent")
	if exists {
		t.Error("expected non-existent key to not exist")
	}
	
	// Test multiple keys
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)
	
	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}
	
	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TTL test in short mode")
	}
	
	config := Config{
		MaxItems:        10,
		DefaultTTL:      10 * time.Millisecond,
		CleanupInterval: 5 * time.Millisecond,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	defer cache.Close()
	
	// Set with short TTL
	err := cache.Set("temp-key", "temp-value", 8*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}
	
	// Should exist immediately
	value, exists := cache.Get("temp-key")
	if !exists {
		t.Error("expected temp-key to exist")
	}
	
	if value != "temp-value" {
		t.Errorf("expected temp-value, got %v", value)
	}
	
	// Wait for expiration
	time.Sleep(15 * time.Millisecond)
	
	// Should be expired
	_, exists = cache.Get("temp-key")
	if exists {
		t.Error("expected temp-key to be expired")
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	config := Config{
		MaxItems:        3,
		DefaultTTL:      time.Minute,
		CleanupInterval: 10 * time.Millisecond,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	defer cache.Close()
	
	// Fill cache to capacity
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)
	
	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}
	
	// Access key1 to make it recently used
	cache.Get("key1")
	
	// Add another item, should evict least recently used
	cache.Set("key4", "value4", 0)
	
	if cache.Size() != 3 {
		t.Errorf("expected size 3 after eviction, got %d", cache.Size())
	}
	
	// key1 should still exist (was accessed recently)
	_, exists := cache.Get("key1")
	if !exists {
		t.Error("expected key1 to still exist")
	}
	
	// key4 should exist (just added)
	_, exists = cache.Get("key4")
	if !exists {
		t.Error("expected key4 to exist")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	config := Config{
		MaxItems:        10,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	defer cache.Close()
	
	// Add some items
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	
	// Delete one
	err := cache.Delete("key1")
	if err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}
	
	// Verify deletion
	_, exists := cache.Get("key1")
	if exists {
		t.Error("expected key1 to be deleted")
	}
	
	// Other key should still exist
	_, exists = cache.Get("key2")
	if !exists {
		t.Error("expected key2 to still exist")
	}
	
	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	config := Config{
		MaxItems:        10,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	defer cache.Close()
	
	// Add some items
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)
	
	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}
	
	// Clear cache
	err := cache.Clear()
	if err != nil {
		t.Fatalf("failed to clear cache: %v", err)
	}
	
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}
	
	// Verify all keys are gone
	_, exists := cache.Get("key1")
	if exists {
		t.Error("expected key1 to be cleared")
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	config := Config{
		MaxItems:        10,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
		PersistToDisk:   false,
	}
	
	cache := NewMemoryCache(config)
	defer cache.Close()
	
	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 || stats.Hits != 0 || stats.Misses != 0 {
		t.Error("expected clean initial stats")
	}
	
	// Set some values
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	
	// Generate hits and misses
	cache.Get("key1")    // hit
	cache.Get("key1")    // hit
	cache.Get("missing") // miss
	cache.Get("key2")    // hit
	cache.Get("missing") // miss
	
	stats = cache.Stats()
	if stats.Size != 2 {
		t.Errorf("expected size 2, got %d", stats.Size)
	}
	
	if stats.Hits != 3 {
		t.Errorf("expected 3 hits, got %d", stats.Hits)
	}
	
	if stats.Misses != 2 {
		t.Errorf("expected 2 misses, got %d", stats.Misses)
	}
	
	expectedHitRatio := float64(3) / float64(5) // 3 hits out of 5 total accesses
	if stats.HitRatio != expectedHitRatio {
		t.Errorf("expected hit ratio %.2f, got %.2f", expectedHitRatio, stats.HitRatio)
	}
}

func TestMemoryCache_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := Config{
		MaxItems:        10,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
		PersistToDisk:   true,
		PersistPath:     tmpDir,
	}
	
	// Create cache and add some data
	cache1 := NewMemoryCache(config)
	cache1.Set("key1", "value1", 0)
	cache1.Set("key2", "value2", 0)
	cache1.Close() // This should persist to disk
	
	// Create new cache with same config - should load from disk
	cache2 := NewMemoryCache(config)
	defer cache2.Close()
	
	// Check if data was loaded
	value, exists := cache2.Get("key1")
	if !exists {
		t.Error("expected key1 to be loaded from persistence")
	}
	
	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}
	
	value, exists = cache2.Get("key2")
	if !exists {
		t.Error("expected key2 to be loaded from persistence")
	}
	
	if value != "value2" {
		t.Errorf("expected value2, got %v", value)
	}
	
	// Verify cache file was created
	cacheFile := filepath.Join(tmpDir, "cache.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Error("expected cache file to exist")
	}
}

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		prefix     string
		components []string
		expected   string
	}{
		{"prefix", []string{}, "prefix"},
		{"user", []string{"123"}, "user:123"},
		{"user", []string{"123", "profile"}, "user:123:profile"},
		{"cache", []string{"ns", "key", "version"}, "cache:ns:key:version"},
	}
	
	for _, test := range tests {
		result := GenerateKey(test.prefix, test.components...)
		if result != test.expected {
			t.Errorf("GenerateKey(%s, %v) = %s, expected %s",
				test.prefix, test.components, result, test.expected)
		}
	}
}