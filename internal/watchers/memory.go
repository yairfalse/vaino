package watchers

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/yairfalse/wgo/pkg/types"
)

// MemoryManager manages memory optimization for long-running watch operations
type MemoryManager struct {
	mu                    sync.RWMutex
	objectPools           map[string]*ObjectPool
	memoryMonitor         *MemoryMonitor
	garbageCollector      *GarbageCollector
	compressionManager    *CompressionManager
	cacheManager          *CacheManager
	enabled               bool
	maxMemoryUsage        int64
	gcThreshold           float64
	compressionThreshold  int64
	cleanupInterval       time.Duration
	stats                 MemoryManagerStats
	ctx                   context.Context
	cancel                context.CancelFunc
}

// ObjectPool provides object pooling for frequently used objects
type ObjectPool struct {
	mu        sync.RWMutex
	name      string
	pool      sync.Pool
	stats     ObjectPoolStats
	maxSize   int
	itemSize  int64
	enabled   bool
	validator func(interface{}) bool
}

// MemoryMonitor monitors memory usage and triggers optimizations
type MemoryMonitor struct {
	mu                  sync.RWMutex
	enabled             bool
	samplingInterval    time.Duration
	memoryStats         MemoryStats
	alertThresholds     map[string]float64
	callbacks           []MemoryCallback
	historySize         int
	memoryHistory       []MemorySnapshot
	ctx                 context.Context
	cancel              context.CancelFunc
}

// GarbageCollector manages garbage collection optimization
type GarbageCollector struct {
	mu                 sync.RWMutex
	enabled            bool
	gcInterval         time.Duration
	forceGCThreshold   float64
	lastGC             time.Time
	gcCount            int64
	totalGCTime        time.Duration
	adaptiveGC         bool
	gcStats            GarbageCollectorStats
	optimizationRules  []GCOptimizationRule
}

// CompressionManager handles data compression for memory optimization
type CompressionManager struct {
	mu                sync.RWMutex
	enabled           bool
	compressionLevel  int
	compressionRatio  float64
	compressedData    map[string][]byte
	compressionStats  CompressionStats
	algorithm         string
}

// CacheManager manages various caches used by the watch system
type CacheManager struct {
	mu               sync.RWMutex
	caches           map[string]*Cache
	enabled          bool
	maxTotalSize     int64
	currentSize      int64
	evictionPolicy   string
	stats            CacheManagerStats
	cleanupInterval  time.Duration
}

// Cache represents a single cache instance
type Cache struct {
	mu         sync.RWMutex
	name       string
	items      map[string]*CacheItem
	maxSize    int64
	currentSize int64
	ttl        time.Duration
	stats      CacheStats
	enabled    bool
}

// CacheItem represents an item in the cache
type CacheItem struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Size       int64       `json:"size"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	AccessCount int64      `json:"access_count"`
	TTL        time.Duration `json:"ttl"`
}

// Statistics structures
type MemoryManagerStats struct {
	TotalMemoryUsage     int64                       `json:"total_memory_usage"`
	PoolMemoryUsage      int64                       `json:"pool_memory_usage"`
	CacheMemoryUsage     int64                       `json:"cache_memory_usage"`
	CompressedMemoryUsage int64                      `json:"compressed_memory_usage"`
	GCTriggerCount       int64                       `json:"gc_trigger_count"`
	MemoryOptimizations  int64                       `json:"memory_optimizations"`
	ObjectPoolStats      map[string]ObjectPoolStats `json:"object_pool_stats"`
	CacheStats           map[string]CacheStats       `json:"cache_stats"`
	CompressionRatio     float64                     `json:"compression_ratio"`
	LastOptimization     time.Time                   `json:"last_optimization"`
}

type ObjectPoolStats struct {
	TotalAllocated    int64     `json:"total_allocated"`
	TotalReturned     int64     `json:"total_returned"`
	CurrentActive     int64     `json:"current_active"`
	CurrentPooled     int64     `json:"current_pooled"`
	HitRate          float64   `json:"hit_rate"`
	MemoryUsage      int64     `json:"memory_usage"`
	LastUsed         time.Time `json:"last_used"`
}

type MemoryStats struct {
	HeapAlloc     int64     `json:"heap_alloc"`
	HeapSys       int64     `json:"heap_sys"`
	HeapIdle      int64     `json:"heap_idle"`
	HeapInuse     int64     `json:"heap_inuse"`
	HeapReleased  int64     `json:"heap_released"`
	HeapObjects   int64     `json:"heap_objects"`
	StackInuse    int64     `json:"stack_inuse"`
	StackSys      int64     `json:"stack_sys"`
	GCCalls       int64     `json:"gc_calls"`
	GCTime        time.Duration `json:"gc_time"`
	LastGC        time.Time `json:"last_gc"`
	Timestamp     time.Time `json:"timestamp"`
}

type MemorySnapshot struct {
	Stats      MemoryStats `json:"stats"`
	Timestamp  time.Time   `json:"timestamp"`
	Utilization float64    `json:"utilization"`
}

type GarbageCollectorStats struct {
	TotalGCRuns        int64         `json:"total_gc_runs"`
	TotalGCTime        time.Duration `json:"total_gc_time"`
	AverageGCTime      time.Duration `json:"average_gc_time"`
	LastGCTime         time.Time     `json:"last_gc_time"`
	MemoryFreed        int64         `json:"memory_freed"`
	OptimizationCount  int64         `json:"optimization_count"`
}

type CompressionStats struct {
	TotalCompressed     int64     `json:"total_compressed"`
	TotalUncompressed   int64     `json:"total_uncompressed"`
	CompressionRatio    float64   `json:"compression_ratio"`
	CompressionTime     time.Duration `json:"compression_time"`
	DecompressionTime   time.Duration `json:"decompression_time"`
	MemorySaved         int64     `json:"memory_saved"`
	LastCompression     time.Time `json:"last_compression"`
}

type CacheManagerStats struct {
	TotalCaches      int               `json:"total_caches"`
	TotalMemoryUsage int64             `json:"total_memory_usage"`
	TotalHits        int64             `json:"total_hits"`
	TotalMisses      int64             `json:"total_misses"`
	HitRate          float64           `json:"hit_rate"`
	EvictionCount    int64             `json:"eviction_count"`
	CacheStats       map[string]CacheStats `json:"cache_stats"`
}

type CacheStats struct {
	TotalItems      int64     `json:"total_items"`
	MemoryUsage     int64     `json:"memory_usage"`
	HitCount        int64     `json:"hit_count"`
	MissCount       int64     `json:"miss_count"`
	HitRate         float64   `json:"hit_rate"`
	EvictionCount   int64     `json:"eviction_count"`
	LastAccess      time.Time `json:"last_access"`
	LastEviction    time.Time `json:"last_eviction"`
}

// Callback types
type MemoryCallback func(stats MemoryStats, action string)
type GCOptimizationRule struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Condition func(MemoryStats) bool `json:"-"`
	Action    func() error           `json:"-"`
	Enabled   bool                   `json:"enabled"`
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager() *MemoryManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MemoryManager{
		objectPools:          make(map[string]*ObjectPool),
		memoryMonitor:        NewMemoryMonitor(),
		garbageCollector:     NewGarbageCollector(),
		compressionManager:   NewCompressionManager(),
		cacheManager:         NewCacheManager(),
		enabled:              true,
		maxMemoryUsage:       1024 * 1024 * 1024, // 1GB
		gcThreshold:          0.8,
		compressionThreshold: 10 * 1024 * 1024, // 10MB
		cleanupInterval:      5 * time.Minute,
		stats:                MemoryManagerStats{
			ObjectPoolStats: make(map[string]ObjectPoolStats),
			CacheStats:      make(map[string]CacheStats),
		},
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor() *MemoryMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MemoryMonitor{
		enabled:          true,
		samplingInterval: 10 * time.Second,
		memoryStats:      MemoryStats{},
		alertThresholds: map[string]float64{
			"heap_usage":   0.8,
			"gc_frequency": 0.9,
		},
		callbacks:     []MemoryCallback{},
		historySize:   100,
		memoryHistory: []MemorySnapshot{},
		ctx:           ctx,
		cancel:        cancel,
	}
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector() *GarbageCollector {
	return &GarbageCollector{
		enabled:           true,
		gcInterval:        30 * time.Second,
		forceGCThreshold:  0.85,
		lastGC:            time.Now(),
		gcCount:           0,
		totalGCTime:       0,
		adaptiveGC:        true,
		gcStats:           GarbageCollectorStats{},
		optimizationRules: []GCOptimizationRule{},
	}
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager() *CompressionManager {
	return &CompressionManager{
		enabled:          true,
		compressionLevel: 6,
		compressionRatio: 0.0,
		compressedData:   make(map[string][]byte),
		compressionStats: CompressionStats{},
		algorithm:        "gzip",
	}
}

// NewCacheManager creates a new cache manager
func NewCacheManager() *CacheManager {
	return &CacheManager{
		caches:          make(map[string]*Cache),
		enabled:         true,
		maxTotalSize:    512 * 1024 * 1024, // 512MB
		currentSize:     0,
		evictionPolicy:  "lru",
		stats:           CacheManagerStats{
			CacheStats: make(map[string]CacheStats),
		},
		cleanupInterval: 2 * time.Minute,
	}
}

// Start starts the memory manager
func (mm *MemoryManager) Start() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if !mm.enabled {
		return fmt.Errorf("memory manager is disabled")
	}
	
	// Start memory monitor
	if err := mm.memoryMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start memory monitor: %w", err)
	}
	
	// Start garbage collector
	if err := mm.garbageCollector.Start(); err != nil {
		return fmt.Errorf("failed to start garbage collector: %w", err)
	}
	
	// Start cache manager
	if err := mm.cacheManager.Start(); err != nil {
		return fmt.Errorf("failed to start cache manager: %w", err)
	}
	
	// Start cleanup loop
	go mm.cleanupLoop()
	
	return nil
}

// Stop stops the memory manager
func (mm *MemoryManager) Stop() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.cancel()
	
	// Stop components
	mm.memoryMonitor.Stop()
	mm.garbageCollector.Stop()
	mm.cacheManager.Stop()
	
	// Clear object pools
	for _, pool := range mm.objectPools {
		pool.Clear()
	}
	
	return nil
}

// GetPool returns an object pool
func (mm *MemoryManager) GetPool(name string) *ObjectPool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	return mm.objectPools[name]
}

// CreatePool creates a new object pool
func (mm *MemoryManager) CreatePool(name string, maxSize int, itemSize int64, factory func() interface{}) *ObjectPool {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	pool := &ObjectPool{
		name:    name,
		maxSize: maxSize,
		itemSize: itemSize,
		enabled: true,
		pool: sync.Pool{
			New: factory,
		},
		stats: ObjectPoolStats{},
	}
	
	mm.objectPools[name] = pool
	return pool
}

// GetStats returns memory manager statistics
func (mm *MemoryManager) GetStats() MemoryManagerStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	// Update pool stats
	for name, pool := range mm.objectPools {
		mm.stats.ObjectPoolStats[name] = pool.GetStats()
	}
	
	// Update cache stats
	for name, cache := range mm.cacheManager.caches {
		mm.stats.CacheStats[name] = cache.GetStats()
	}
	
	// Update memory usage
	mm.stats.TotalMemoryUsage = mm.calculateTotalMemoryUsage()
	mm.stats.CompressionRatio = mm.compressionManager.compressionStats.CompressionRatio
	
	return mm.stats
}

// ObjectPool methods
func (op *ObjectPool) Get() interface{} {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	if !op.enabled {
		return nil
	}
	
	obj := op.pool.Get()
	op.stats.TotalAllocated++
	op.stats.CurrentActive++
	op.stats.LastUsed = time.Now()
	
	return obj
}

func (op *ObjectPool) Put(obj interface{}) {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	if !op.enabled {
		return
	}
	
	// Validate object if validator is set
	if op.validator != nil && !op.validator(obj) {
		return
	}
	
	op.pool.Put(obj)
	op.stats.TotalReturned++
	op.stats.CurrentActive--
	op.stats.CurrentPooled++
	
	// Update hit rate
	if op.stats.TotalAllocated > 0 {
		op.stats.HitRate = float64(op.stats.TotalReturned) / float64(op.stats.TotalAllocated)
	}
}

func (op *ObjectPool) GetStats() ObjectPoolStats {
	op.mu.RLock()
	defer op.mu.RUnlock()
	
	op.stats.MemoryUsage = op.stats.CurrentPooled * op.itemSize
	return op.stats
}

func (op *ObjectPool) Clear() {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	// Reset the pool
	op.pool = sync.Pool{
		New: op.pool.New,
	}
	
	op.stats.CurrentPooled = 0
	op.stats.CurrentActive = 0
}

// MemoryMonitor methods
func (mm *MemoryMonitor) Start() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if !mm.enabled {
		return fmt.Errorf("memory monitor is disabled")
	}
	
	go mm.monitoringLoop()
	return nil
}

func (mm *MemoryMonitor) Stop() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if mm.cancel != nil {
		mm.cancel()
	}
}

func (mm *MemoryMonitor) monitoringLoop() {
	ticker := time.NewTicker(mm.samplingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.updateMemoryStats()
		}
	}
}

func (mm *MemoryMonitor) updateMemoryStats() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	mm.memoryStats = MemoryStats{
		HeapAlloc:    int64(m.HeapAlloc),
		HeapSys:      int64(m.HeapSys),
		HeapIdle:     int64(m.HeapIdle),
		HeapInuse:    int64(m.HeapInuse),
		HeapReleased: int64(m.HeapReleased),
		HeapObjects:  int64(m.HeapObjects),
		StackInuse:   int64(m.StackInuse),
		StackSys:     int64(m.StackSys),
		GCCalls:      int64(m.NumGC),
		GCTime:       time.Duration(m.PauseTotalNs),
		LastGC:       time.Unix(0, int64(m.LastGC)),
		Timestamp:    time.Now(),
	}
	
	// Add to history
	utilization := float64(mm.memoryStats.HeapInuse) / float64(mm.memoryStats.HeapSys)
	snapshot := MemorySnapshot{
		Stats:       mm.memoryStats,
		Timestamp:   time.Now(),
		Utilization: utilization,
	}
	
	mm.memoryHistory = append(mm.memoryHistory, snapshot)
	
	// Limit history size
	if len(mm.memoryHistory) > mm.historySize {
		mm.memoryHistory = mm.memoryHistory[len(mm.memoryHistory)-mm.historySize:]
	}
	
	// Check thresholds and trigger callbacks
	mm.checkThresholds()
}

func (mm *MemoryMonitor) checkThresholds() {
	utilization := float64(mm.memoryStats.HeapInuse) / float64(mm.memoryStats.HeapSys)
	
	if utilization > mm.alertThresholds["heap_usage"] {
		for _, callback := range mm.callbacks {
			callback(mm.memoryStats, "high_heap_usage")
		}
	}
}

// GarbageCollector methods
func (gc *GarbageCollector) Start() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	if !gc.enabled {
		return fmt.Errorf("garbage collector is disabled")
	}
	
	go gc.gcLoop()
	return nil
}

func (gc *GarbageCollector) Stop() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	gc.enabled = false
}

func (gc *GarbageCollector) gcLoop() {
	ticker := time.NewTicker(gc.gcInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			gc.checkAndRunGC()
		}
	}
}

func (gc *GarbageCollector) checkAndRunGC() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	if !gc.enabled {
		return
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	utilization := float64(m.HeapInuse) / float64(m.HeapSys)
	
	if utilization > gc.forceGCThreshold {
		gc.runGC()
	}
}

func (gc *GarbageCollector) runGC() {
	startTime := time.Now()
	
	runtime.GC()
	
	gcTime := time.Since(startTime)
	gc.gcCount++
	gc.totalGCTime += gcTime
	gc.lastGC = time.Now()
	
	// Update stats
	gc.gcStats.TotalGCRuns++
	gc.gcStats.TotalGCTime += gcTime
	gc.gcStats.LastGCTime = time.Now()
	
	if gc.gcStats.TotalGCRuns > 0 {
		gc.gcStats.AverageGCTime = gc.gcStats.TotalGCTime / time.Duration(gc.gcStats.TotalGCRuns)
	}
}

// CacheManager methods
func (cm *CacheManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if !cm.enabled {
		return fmt.Errorf("cache manager is disabled")
	}
	
	go cm.cleanupLoop()
	return nil
}

func (cm *CacheManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.enabled = false
	
	// Clear all caches
	for _, cache := range cm.caches {
		cache.Clear()
	}
}

func (cm *CacheManager) GetCache(name string) *Cache {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.caches[name]
}

func (cm *CacheManager) CreateCache(name string, maxSize int64, ttl time.Duration) *Cache {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cache := &Cache{
		name:        name,
		items:       make(map[string]*CacheItem),
		maxSize:     maxSize,
		currentSize: 0,
		ttl:         ttl,
		stats:       CacheStats{},
		enabled:     true,
	}
	
	cm.caches[name] = cache
	return cache
}

func (cm *CacheManager) cleanupLoop() {
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.cleanupExpiredItems()
		}
	}
}

func (cm *CacheManager) cleanupExpiredItems() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	for _, cache := range cm.caches {
		cache.CleanupExpired()
	}
}

// Cache methods
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.enabled {
		return nil, false
	}
	
	item, exists := c.items[key]
	if !exists {
		c.stats.MissCount++
		return nil, false
	}
	
	// Check if expired
	if c.ttl > 0 && time.Since(item.CreatedAt) > c.ttl {
		delete(c.items, key)
		c.currentSize -= item.Size
		c.stats.MissCount++
		return nil, false
	}
	
	// Update access info
	item.AccessedAt = time.Now()
	item.AccessCount++
	
	c.stats.HitCount++
	c.stats.LastAccess = time.Now()
	
	// Update hit rate
	totalAccess := c.stats.HitCount + c.stats.MissCount
	if totalAccess > 0 {
		c.stats.HitRate = float64(c.stats.HitCount) / float64(totalAccess)
	}
	
	return item.Value, true
}

func (c *Cache) Set(key string, value interface{}, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.enabled {
		return
	}
	
	// Check if we need to evict
	if c.currentSize+size > c.maxSize {
		c.evictItems(size)
	}
	
	item := &CacheItem{
		Key:         key,
		Value:       value,
		Size:        size,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		TTL:         c.ttl,
	}
	
	// Remove existing item if present
	if existing, exists := c.items[key]; exists {
		c.currentSize -= existing.Size
	}
	
	c.items[key] = item
	c.currentSize += size
	c.stats.TotalItems = int64(len(c.items))
	c.stats.MemoryUsage = c.currentSize
}

func (c *Cache) evictItems(spaceNeeded int64) {
	// Simple LRU eviction
	var itemsToEvict []*CacheItem
	
	for _, item := range c.items {
		itemsToEvict = append(itemsToEvict, item)
	}
	
	// Sort by access time (oldest first)
	for i := 0; i < len(itemsToEvict)-1; i++ {
		for j := i + 1; j < len(itemsToEvict); j++ {
			if itemsToEvict[i].AccessedAt.After(itemsToEvict[j].AccessedAt) {
				itemsToEvict[i], itemsToEvict[j] = itemsToEvict[j], itemsToEvict[i]
			}
		}
	}
	
	// Evict items until we have enough space
	spaceFreed := int64(0)
	for _, item := range itemsToEvict {
		if spaceFreed >= spaceNeeded {
			break
		}
		
		delete(c.items, item.Key)
		c.currentSize -= item.Size
		spaceFreed += item.Size
		c.stats.EvictionCount++
	}
	
	c.stats.LastEviction = time.Now()
}

func (c *Cache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.ttl <= 0 {
		return
	}
	
	now := time.Now()
	var keysToDelete []string
	
	for key, item := range c.items {
		if now.Sub(item.CreatedAt) > c.ttl {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		if item, exists := c.items[key]; exists {
			delete(c.items, key)
			c.currentSize -= item.Size
		}
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items = make(map[string]*CacheItem)
	c.currentSize = 0
	c.stats = CacheStats{}
}

func (c *Cache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	c.stats.TotalItems = int64(len(c.items))
	c.stats.MemoryUsage = c.currentSize
	
	return c.stats
}

// Helper methods
func (mm *MemoryManager) calculateTotalMemoryUsage() int64 {
	var total int64
	
	// Add pool memory usage
	for _, pool := range mm.objectPools {
		stats := pool.GetStats()
		total += stats.MemoryUsage
	}
	
	// Add cache memory usage
	total += mm.cacheManager.currentSize
	
	return total
}

func (mm *MemoryManager) cleanupLoop() {
	ticker := time.NewTicker(mm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.performCleanup()
		}
	}
}

func (mm *MemoryManager) performCleanup() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// Check if we need to optimize memory usage
	totalUsage := mm.calculateTotalMemoryUsage()
	
	if totalUsage > mm.maxMemoryUsage {
		// Trigger optimizations
		mm.optimizeMemoryUsage()
	}
	
	mm.stats.LastOptimization = time.Now()
}

func (mm *MemoryManager) optimizeMemoryUsage() {
	// Clear object pools if needed
	for _, pool := range mm.objectPools {
		pool.Clear()
	}
	
	// Force garbage collection
	mm.garbageCollector.runGC()
	
	// Clear less important caches
	mm.cacheManager.cleanupExpiredItems()
	
	mm.stats.MemoryOptimizations++
}

// Utility functions for creating pooled objects
func (mm *MemoryManager) GetWatchEventPool() *ObjectPool {
	pool := mm.GetPool("watch_event")
	if pool == nil {
		pool = mm.CreatePool("watch_event", 1000, int64(unsafe.Sizeof(WatchEvent{})), func() interface{} {
			return &WatchEvent{}
		})
	}
	return pool
}

func (mm *MemoryManager) GetResourcePool() *ObjectPool {
	pool := mm.GetPool("resource")
	if pool == nil {
		pool = mm.CreatePool("resource", 5000, int64(unsafe.Sizeof(types.Resource{})), func() interface{} {
			return &types.Resource{}
		})
	}
	return pool
}

func (mm *MemoryManager) GetResourceSnapshotPool() *ObjectPool {
	pool := mm.GetPool("resource_snapshot")
	if pool == nil {
		pool = mm.CreatePool("resource_snapshot", 5000, int64(unsafe.Sizeof(ResourceSnapshot{})), func() interface{} {
			return &ResourceSnapshot{}
		})
	}
	return pool
}

// IsEnabled returns whether the memory manager is enabled
func (mm *MemoryManager) IsEnabled() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.enabled
}

// SetEnabled enables or disables the memory manager
func (mm *MemoryManager) SetEnabled(enabled bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.enabled = enabled
}