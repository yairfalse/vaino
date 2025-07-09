package workers

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// MemoryOptimizationConfig configures memory optimization strategies
type MemoryOptimizationConfig struct {
	// Memory limits
	MaxMemoryUsage        int64 `yaml:"max_memory_usage"`
	GCThreshold           int64 `yaml:"gc_threshold"`
	BackpressureThreshold int64 `yaml:"backpressure_threshold"`

	// Object pooling
	EnableObjectPooling bool          `yaml:"enable_object_pooling"`
	PoolSize            int           `yaml:"pool_size"`
	PoolCleanupInterval time.Duration `yaml:"pool_cleanup_interval"`

	// Memory management
	EnableMemoryMonitoring bool          `yaml:"enable_memory_monitoring"`
	MonitoringInterval     time.Duration `yaml:"monitoring_interval"`
	MemoryProfileEnabled   bool          `yaml:"memory_profile_enabled"`

	// Streaming
	StreamingThreshold int64 `yaml:"streaming_threshold"`
	ChunkSize          int64 `yaml:"chunk_size"`
	BufferSize         int   `yaml:"buffer_size"`

	// Garbage collection
	GCPercent       int           `yaml:"gc_percent"`
	ForceGCInterval time.Duration `yaml:"force_gc_interval"`

	// Resource limits
	MaxResourcesInMemory int `yaml:"max_resources_in_memory"`
	ResourceBatchSize    int `yaml:"resource_batch_size"`
}

// MemoryOptimizer manages memory optimization strategies
type MemoryOptimizer struct {
	config           MemoryOptimizationConfig
	memoryMonitor    *MemoryMonitor
	objectPools      *ObjectPoolManager
	streamingManager *StreamingManager
	gcManager        *GCManager
	resourceBatching *ResourceBatchingManager

	// State
	ctx     context.Context
	cancel  context.CancelFunc
	running int32
	mu      sync.RWMutex

	// Statistics
	stats MemoryOptimizationStats

	// Backpressure
	backpressureActive int32
	backpressureSignal chan struct{}
}

// MemoryOptimizationStats tracks memory optimization metrics
type MemoryOptimizationStats struct {
	TotalMemoryUsage    int64     `json:"total_memory_usage"`
	HeapMemoryUsage     int64     `json:"heap_memory_usage"`
	StackMemoryUsage    int64     `json:"stack_memory_usage"`
	GoroutineCount      int       `json:"goroutine_count"`
	GCRuns              int64     `json:"gc_runs"`
	GCPauseTime         int64     `json:"gc_pause_time"`
	ObjectPoolHits      int64     `json:"object_pool_hits"`
	ObjectPoolMisses    int64     `json:"object_pool_misses"`
	StreamingOperations int64     `json:"streaming_operations"`
	BackpressureEvents  int64     `json:"backpressure_events"`
	LastUpdated         time.Time `json:"last_updated"`
}

// MemoryMonitor monitors memory usage and triggers optimizations
type MemoryMonitor struct {
	optimizer       *MemoryOptimizer
	interval        time.Duration
	lastMemoryUsage int64
	ticker          *time.Ticker
	stopChan        chan struct{}
	mu              sync.RWMutex
}

// ObjectPoolManager manages object pools for memory efficiency
type ObjectPoolManager struct {
	resourcePool *sync.Pool
	snapshotPool *sync.Pool
	changePool   *sync.Pool
	bufferPool   *sync.Pool

	// Statistics
	hits   int64
	misses int64

	// Configuration
	maxPoolSize     int
	cleanupInterval time.Duration

	// Cleanup
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
}

// StreamingManager handles streaming operations for large datasets
type StreamingManager struct {
	threshold     int64
	chunkSize     int64
	bufferSize    int
	activeStreams map[string]*StreamingContext
	mu            sync.RWMutex
}

// StreamingContext represents an active streaming operation
type StreamingContext struct {
	ID            string
	TotalSize     int64
	ProcessedSize int64
	StartTime     time.Time
	Buffer        []byte
	ChunkChan     chan []byte
	ErrorChan     chan error
	DoneChan      chan struct{}
}

// GCManager manages garbage collection optimization
type GCManager struct {
	gcPercent       int
	forceGCInterval time.Duration
	lastGCTime      time.Time
	gcTicker        *time.Ticker
	stopChan        chan struct{}

	// Statistics
	gcRuns      int64
	totalGCTime int64
	mu          sync.RWMutex
}

// ResourceBatchingManager manages resource processing in batches
type ResourceBatchingManager struct {
	maxInMemory   int
	batchSize     int
	activeBatches map[string]*ResourceBatch
	mu            sync.RWMutex
}

// ResourceBatch represents a batch of resources being processed
type ResourceBatch struct {
	ID          string
	Resources   []types.Resource
	Size        int
	MaxSize     int
	ProcessedAt time.Time
	Status      BatchStatus
}

// BatchStatus represents the status of a resource batch
type BatchStatus int

const (
	BatchStatusPending BatchStatus = iota
	BatchStatusProcessing
	BatchStatusCompleted
	BatchStatusFailed
)

// NewMemoryOptimizer creates a new memory optimizer
func NewMemoryOptimizer(config MemoryOptimizationConfig) *MemoryOptimizer {
	// Set defaults
	if config.MaxMemoryUsage <= 0 {
		config.MaxMemoryUsage = 500 * 1024 * 1024 // 500MB
	}
	if config.GCThreshold <= 0 {
		config.GCThreshold = config.MaxMemoryUsage * 7 / 10 // 70% of max
	}
	if config.BackpressureThreshold <= 0 {
		config.BackpressureThreshold = config.MaxMemoryUsage * 8 / 10 // 80% of max
	}
	if config.MonitoringInterval <= 0 {
		config.MonitoringInterval = 5 * time.Second
	}
	if config.StreamingThreshold <= 0 {
		config.StreamingThreshold = 50 * 1024 * 1024 // 50MB
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1024 * 1024 // 1MB
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}
	if config.GCPercent <= 0 {
		config.GCPercent = 100
	}
	if config.ForceGCInterval <= 0 {
		config.ForceGCInterval = 30 * time.Second
	}
	if config.MaxResourcesInMemory <= 0 {
		config.MaxResourcesInMemory = 10000
	}
	if config.ResourceBatchSize <= 0 {
		config.ResourceBatchSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	mo := &MemoryOptimizer{
		config:             config,
		ctx:                ctx,
		cancel:             cancel,
		backpressureSignal: make(chan struct{}, 1),
	}

	// Initialize components
	mo.memoryMonitor = NewMemoryMonitor(mo, config.MonitoringInterval)
	mo.objectPools = NewObjectPoolManager(config.PoolSize, config.PoolCleanupInterval)
	mo.streamingManager = NewStreamingManager(config.StreamingThreshold, config.ChunkSize, config.BufferSize)
	mo.gcManager = NewGCManager(config.GCPercent, config.ForceGCInterval)
	mo.resourceBatching = NewResourceBatchingManager(config.MaxResourcesInMemory, config.ResourceBatchSize)

	return mo
}

// Start starts the memory optimizer
func (mo *MemoryOptimizer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&mo.running, 0, 1) {
		return fmt.Errorf("memory optimizer is already running")
	}

	// Start components
	if mo.config.EnableMemoryMonitoring {
		mo.memoryMonitor.Start(ctx)
	}

	if mo.config.EnableObjectPooling {
		mo.objectPools.Start(ctx)
	}

	mo.gcManager.Start(ctx)

	return nil
}

// Stop stops the memory optimizer
func (mo *MemoryOptimizer) Stop() error {
	if !atomic.CompareAndSwapInt32(&mo.running, 1, 0) {
		return fmt.Errorf("memory optimizer is not running")
	}

	mo.cancel()

	// Stop components
	mo.memoryMonitor.Stop()
	mo.objectPools.Stop()
	mo.gcManager.Stop()

	return nil
}

// GetResource gets a resource from the pool
func (mo *MemoryOptimizer) GetResource() *types.Resource {
	if mo.config.EnableObjectPooling {
		return mo.objectPools.GetResource()
	}
	return &types.Resource{}
}

// PutResource returns a resource to the pool
func (mo *MemoryOptimizer) PutResource(resource *types.Resource) {
	if mo.config.EnableObjectPooling {
		mo.objectPools.PutResource(resource)
	}
}

// GetSnapshot gets a snapshot from the pool
func (mo *MemoryOptimizer) GetSnapshot() *types.Snapshot {
	if mo.config.EnableObjectPooling {
		return mo.objectPools.GetSnapshot()
	}
	return &types.Snapshot{}
}

// PutSnapshot returns a snapshot to the pool
func (mo *MemoryOptimizer) PutSnapshot(snapshot *types.Snapshot) {
	if mo.config.EnableObjectPooling {
		mo.objectPools.PutSnapshot(snapshot)
	}
}

// ShouldUseStreaming determines if streaming should be used for a dataset
func (mo *MemoryOptimizer) ShouldUseStreaming(dataSize int64) bool {
	return dataSize > mo.config.StreamingThreshold
}

// CreateStreamingContext creates a new streaming context
func (mo *MemoryOptimizer) CreateStreamingContext(id string, totalSize int64) *StreamingContext {
	return mo.streamingManager.CreateContext(id, totalSize)
}

// IsBackpressureActive returns whether backpressure is currently active
func (mo *MemoryOptimizer) IsBackpressureActive() bool {
	return atomic.LoadInt32(&mo.backpressureActive) == 1
}

// WaitForBackpressure waits for backpressure to be released
func (mo *MemoryOptimizer) WaitForBackpressure(ctx context.Context) error {
	if !mo.IsBackpressureActive() {
		return nil
	}

	select {
	case <-mo.backpressureSignal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("backpressure timeout")
	}
}

// CreateResourceBatch creates a new resource batch
func (mo *MemoryOptimizer) CreateResourceBatch(id string) *ResourceBatch {
	return mo.resourceBatching.CreateBatch(id, mo.config.ResourceBatchSize)
}

// GetStats returns memory optimization statistics
func (mo *MemoryOptimizer) GetStats() MemoryOptimizationStats {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mo.stats.TotalMemoryUsage = int64(m.Alloc)
	mo.stats.HeapMemoryUsage = int64(m.HeapAlloc)
	mo.stats.StackMemoryUsage = int64(m.StackInuse)
	mo.stats.GoroutineCount = runtime.NumGoroutine()
	mo.stats.GCRuns = int64(m.NumGC)
	mo.stats.GCPauseTime = int64(m.PauseTotalNs)
	mo.stats.ObjectPoolHits = atomic.LoadInt64(&mo.objectPools.hits)
	mo.stats.ObjectPoolMisses = atomic.LoadInt64(&mo.objectPools.misses)
	mo.stats.LastUpdated = time.Now()

	return mo.stats
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(optimizer *MemoryOptimizer, interval time.Duration) *MemoryMonitor {
	return &MemoryMonitor{
		optimizer: optimizer,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

// Start starts the memory monitor
func (mm *MemoryMonitor) Start(ctx context.Context) {
	mm.ticker = time.NewTicker(mm.interval)

	go func() {
		defer mm.ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-mm.stopChan:
				return
			case <-mm.ticker.C:
				mm.checkMemoryUsage()
			}
		}
	}()
}

// Stop stops the memory monitor
func (mm *MemoryMonitor) Stop() {
	close(mm.stopChan)
}

// checkMemoryUsage checks current memory usage and triggers optimizations
func (mm *MemoryMonitor) checkMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentUsage := int64(m.Alloc)
	config := mm.optimizer.config

	// Check for backpressure threshold
	if currentUsage > config.BackpressureThreshold {
		atomic.StoreInt32(&mm.optimizer.backpressureActive, 1)
		atomic.AddInt64(&mm.optimizer.stats.BackpressureEvents, 1)
	} else {
		// Release backpressure
		if atomic.CompareAndSwapInt32(&mm.optimizer.backpressureActive, 1, 0) {
			// Signal backpressure release
			select {
			case mm.optimizer.backpressureSignal <- struct{}{}:
			default:
			}
		}
	}

	// Check for GC threshold
	if currentUsage > config.GCThreshold {
		mm.optimizer.gcManager.ForceGC()
	}

	// Update last usage
	mm.mu.Lock()
	mm.lastMemoryUsage = currentUsage
	mm.mu.Unlock()
}

// NewObjectPoolManager creates a new object pool manager
func NewObjectPoolManager(poolSize int, cleanupInterval time.Duration) *ObjectPoolManager {
	opm := &ObjectPoolManager{
		maxPoolSize:     poolSize,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}

	// Initialize pools
	opm.resourcePool = &sync.Pool{
		New: func() interface{} {
			return &types.Resource{}
		},
	}

	opm.snapshotPool = &sync.Pool{
		New: func() interface{} {
			return &types.Snapshot{}
		},
	}

	opm.changePool = &sync.Pool{
		New: func() interface{} {
			return &types.Change{}
		},
	}

	opm.bufferPool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 64*1024) // 64KB buffer
		},
	}

	return opm
}

// Start starts the object pool manager
func (opm *ObjectPoolManager) Start(ctx context.Context) {
	opm.cleanupTicker = time.NewTicker(opm.cleanupInterval)

	go func() {
		defer opm.cleanupTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-opm.stopChan:
				return
			case <-opm.cleanupTicker.C:
				opm.cleanup()
			}
		}
	}()
}

// Stop stops the object pool manager
func (opm *ObjectPoolManager) Stop() {
	close(opm.stopChan)
}

// GetResource gets a resource from the pool
func (opm *ObjectPoolManager) GetResource() *types.Resource {
	resource := opm.resourcePool.Get().(*types.Resource)
	// Reset resource
	*resource = types.Resource{}
	atomic.AddInt64(&opm.hits, 1)
	return resource
}

// PutResource returns a resource to the pool
func (opm *ObjectPoolManager) PutResource(resource *types.Resource) {
	opm.resourcePool.Put(resource)
}

// GetSnapshot gets a snapshot from the pool
func (opm *ObjectPoolManager) GetSnapshot() *types.Snapshot {
	snapshot := opm.snapshotPool.Get().(*types.Snapshot)
	// Reset snapshot
	*snapshot = types.Snapshot{}
	atomic.AddInt64(&opm.hits, 1)
	return snapshot
}

// PutSnapshot returns a snapshot to the pool
func (opm *ObjectPoolManager) PutSnapshot(snapshot *types.Snapshot) {
	opm.snapshotPool.Put(snapshot)
}

// GetBuffer gets a buffer from the pool
func (opm *ObjectPoolManager) GetBuffer() []byte {
	buffer := opm.bufferPool.Get().([]byte)
	atomic.AddInt64(&opm.hits, 1)
	return buffer
}

// PutBuffer returns a buffer to the pool
func (opm *ObjectPoolManager) PutBuffer(buffer []byte) {
	opm.bufferPool.Put(buffer)
}

// cleanup performs periodic cleanup of pools
func (opm *ObjectPoolManager) cleanup() {
	// Pool cleanup is handled automatically by sync.Pool
	// This method can be used for custom cleanup logic
}

// NewStreamingManager creates a new streaming manager
func NewStreamingManager(threshold, chunkSize int64, bufferSize int) *StreamingManager {
	return &StreamingManager{
		threshold:     threshold,
		chunkSize:     chunkSize,
		bufferSize:    bufferSize,
		activeStreams: make(map[string]*StreamingContext),
	}
}

// CreateContext creates a new streaming context
func (sm *StreamingManager) CreateContext(id string, totalSize int64) *StreamingContext {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	ctx := &StreamingContext{
		ID:        id,
		TotalSize: totalSize,
		StartTime: time.Now(),
		Buffer:    make([]byte, sm.chunkSize),
		ChunkChan: make(chan []byte, sm.bufferSize),
		ErrorChan: make(chan error, 1),
		DoneChan:  make(chan struct{}),
	}

	sm.activeStreams[id] = ctx
	return ctx
}

// NewGCManager creates a new GC manager
func NewGCManager(gcPercent int, forceGCInterval time.Duration) *GCManager {
	return &GCManager{
		gcPercent:       gcPercent,
		forceGCInterval: forceGCInterval,
		stopChan:        make(chan struct{}),
	}
}

// Start starts the GC manager
func (gcm *GCManager) Start(ctx context.Context) {
	// Set GC percent
	debug.SetGCPercent(gcm.gcPercent)

	gcm.gcTicker = time.NewTicker(gcm.forceGCInterval)

	go func() {
		defer gcm.gcTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-gcm.stopChan:
				return
			case <-gcm.gcTicker.C:
				gcm.ForceGC()
			}
		}
	}()
}

// Stop stops the GC manager
func (gcm *GCManager) Stop() {
	close(gcm.stopChan)
}

// ForceGC forces garbage collection
func (gcm *GCManager) ForceGC() {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	startTime := time.Now()
	runtime.GC()
	gcTime := time.Since(startTime)

	atomic.AddInt64(&gcm.gcRuns, 1)
	atomic.AddInt64(&gcm.totalGCTime, gcTime.Nanoseconds())
	gcm.lastGCTime = time.Now()
}

// NewResourceBatchingManager creates a new resource batching manager
func NewResourceBatchingManager(maxInMemory, batchSize int) *ResourceBatchingManager {
	return &ResourceBatchingManager{
		maxInMemory:   maxInMemory,
		batchSize:     batchSize,
		activeBatches: make(map[string]*ResourceBatch),
	}
}

// CreateBatch creates a new resource batch
func (rbm *ResourceBatchingManager) CreateBatch(id string, maxSize int) *ResourceBatch {
	rbm.mu.Lock()
	defer rbm.mu.Unlock()

	batch := &ResourceBatch{
		ID:          id,
		Resources:   make([]types.Resource, 0, maxSize),
		MaxSize:     maxSize,
		ProcessedAt: time.Now(),
		Status:      BatchStatusPending,
	}

	rbm.activeBatches[id] = batch
	return batch
}

// GetBatch gets a batch by ID
func (rbm *ResourceBatchingManager) GetBatch(id string) *ResourceBatch {
	rbm.mu.RLock()
	defer rbm.mu.RUnlock()

	return rbm.activeBatches[id]
}

// DefaultMemoryOptimizationConfig returns a default memory optimization configuration
func DefaultMemoryOptimizationConfig() MemoryOptimizationConfig {
	return MemoryOptimizationConfig{
		MaxMemoryUsage:         500 * 1024 * 1024, // 500MB
		GCThreshold:            350 * 1024 * 1024, // 350MB
		BackpressureThreshold:  400 * 1024 * 1024, // 400MB
		EnableObjectPooling:    true,
		PoolSize:               1000,
		PoolCleanupInterval:    5 * time.Minute,
		EnableMemoryMonitoring: true,
		MonitoringInterval:     5 * time.Second,
		MemoryProfileEnabled:   false,
		StreamingThreshold:     50 * 1024 * 1024, // 50MB
		ChunkSize:              1024 * 1024,      // 1MB
		BufferSize:             100,
		GCPercent:              100,
		ForceGCInterval:        30 * time.Second,
		MaxResourcesInMemory:   10000,
		ResourceBatchSize:      100,
	}
}
