package workers

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// WorkerPoolConfig contains configuration for all worker pools
type WorkerPoolConfig struct {
	ResourceProcessor ResourceProcessorConfig `yaml:"resource_processor"`
	TerraformParser   TerraformParserConfig   `yaml:"terraform_parser"`
	DiffWorker        DiffWorkerConfig        `yaml:"diff_worker"`
	StorageManager    StorageManagerConfig    `yaml:"storage_manager"`
	
	// Global settings
	MaxConcurrency    int           `yaml:"max_concurrency"`
	DefaultTimeout    time.Duration `yaml:"default_timeout"`
	MemoryLimit       int64         `yaml:"memory_limit"`
	EnableMetrics     bool          `yaml:"enable_metrics"`
	MetricsInterval   time.Duration `yaml:"metrics_interval"`
	HealthCheckPort   int           `yaml:"health_check_port"`
}

// ResourceProcessorConfig configures the resource processor
type ResourceProcessorConfig struct {
	Enabled       bool          `yaml:"enabled"`
	WorkerCount   int           `yaml:"worker_count"`
	BufferSize    int           `yaml:"buffer_size"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
	Timeout       time.Duration `yaml:"timeout"`
	MemoryLimit   int64         `yaml:"memory_limit"`
	RateLimit     int           `yaml:"rate_limit"`
	RateBurst     int           `yaml:"rate_burst"`
}

// TerraformParserConfig configures the Terraform parser
type TerraformParserConfig struct {
	Enabled       bool          `yaml:"enabled"`
	WorkerCount   int           `yaml:"worker_count"`
	BufferSize    int           `yaml:"buffer_size"`
	MaxFileSize   int64         `yaml:"max_file_size"`
	Timeout       time.Duration `yaml:"timeout"`
	StreamingMode bool          `yaml:"streaming_mode"`
}

// DiffWorkerConfig configures the diff worker
type DiffWorkerConfig struct {
	Enabled       bool          `yaml:"enabled"`
	WorkerCount   int           `yaml:"worker_count"`
	BufferSize    int           `yaml:"buffer_size"`
	BatchSize     int           `yaml:"batch_size"`
	Timeout       time.Duration `yaml:"timeout"`
	CacheEnabled  bool          `yaml:"cache_enabled"`
	CacheTTL      time.Duration `yaml:"cache_ttl"`
}

// StorageManagerConfig configures the storage manager
type StorageManagerConfig struct {
	Enabled       bool          `yaml:"enabled"`
	WorkerCount   int           `yaml:"worker_count"`
	BufferSize    int           `yaml:"buffer_size"`
	Timeout       time.Duration `yaml:"timeout"`
	MaxFileSize   int64         `yaml:"max_file_size"`
	Compression   bool          `yaml:"compression"`
	Encryption    bool          `yaml:"encryption"`
	Backup        bool          `yaml:"backup"`
}

// WorkerPoolManager manages all worker pools
type WorkerPoolManager struct {
	config           WorkerPoolConfig
	resourceProcessor *ResourceProcessor
	terraformParser  *ConcurrentTerraformParser
	diffWorker       *DiffWorker
	storageManager   *ConcurrentStorageManager
	
	// State management
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	running        bool
	
	// Metrics
	metricsCollector *MetricsCollector
	healthMonitor    *HealthMonitor
	
	// Lifecycle management
	shutdownTimeout time.Duration
	gracefulStop    chan struct{}
}

// MetricsCollector collects metrics from all worker pools
type MetricsCollector struct {
	interval        time.Duration
	enabled         bool
	lastCollection  time.Time
	metrics         WorkerPoolMetrics
	mu              sync.RWMutex
}

// WorkerPoolMetrics contains metrics from all worker pools
type WorkerPoolMetrics struct {
	Timestamp         time.Time                `json:"timestamp"`
	ResourceProcessor ProcessingStats          `json:"resource_processor"`
	TerraformParser   TerraformParsingStats    `json:"terraform_parser"`
	DiffWorker        DiffWorkerStats          `json:"diff_worker"`
	StorageManager    StorageManagerStats      `json:"storage_manager"`
	SystemMetrics     SystemMetrics            `json:"system_metrics"`
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     int64   `json:"memory_usage"`
	MemoryLimit     int64   `json:"memory_limit"`
	GoroutineCount  int     `json:"goroutine_count"`
	GCPauses        int64   `json:"gc_pauses"`
	HeapSize        int64   `json:"heap_size"`
	StackSize       int64   `json:"stack_size"`
}

// HealthMonitor monitors the health of worker pools
type HealthMonitor struct {
	port           int
	checkInterval  time.Duration
	healthChecks   []HealthCheck
	status         HealthStatus
	mu             sync.RWMutex
}

// HealthCheck represents a health check
type HealthCheck struct {
	Name        string
	Check       func() error
	LastResult  error
	LastChecked time.Time
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Healthy    bool      `json:"healthy"`
	Timestamp  time.Time `json:"timestamp"`
	Checks     []HealthCheckResult `json:"checks"`
	Uptime     time.Duration `json:"uptime"`
	StartTime  time.Time `json:"start_time"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Name      string    `json:"name"`
	Healthy   bool      `json:"healthy"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewWorkerPoolManager creates a new worker pool manager
func NewWorkerPoolManager(config WorkerPoolConfig) *WorkerPoolManager {
	// Set defaults
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = runtime.NumCPU() * 2
	}
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.MemoryLimit <= 0 {
		config.MemoryLimit = 500 * 1024 * 1024 // 500MB
	}
	if config.MetricsInterval <= 0 {
		config.MetricsInterval = 30 * time.Second
	}
	if config.HealthCheckPort <= 0 {
		config.HealthCheckPort = 8080
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	wpm := &WorkerPoolManager{
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		shutdownTimeout: 30 * time.Second,
		gracefulStop:    make(chan struct{}),
	}
	
	// Initialize metrics collector
	if config.EnableMetrics {
		wpm.metricsCollector = &MetricsCollector{
			interval: config.MetricsInterval,
			enabled:  true,
		}
	}
	
	// Initialize health monitor
	wpm.healthMonitor = &HealthMonitor{
		port:          config.HealthCheckPort,
		checkInterval: 10 * time.Second,
		status: HealthStatus{
			StartTime: time.Now(),
		},
	}
	
	return wpm
}

// Start starts all configured worker pools
func (wpm *WorkerPoolManager) Start(ctx context.Context) error {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()
	
	if wpm.running {
		return fmt.Errorf("worker pool manager is already running")
	}
	
	// Start resource processor
	if wpm.config.ResourceProcessor.Enabled {
		if err := wpm.startResourceProcessor(); err != nil {
			return fmt.Errorf("failed to start resource processor: %w", err)
		}
	}
	
	// Start Terraform parser
	if wpm.config.TerraformParser.Enabled {
		if err := wpm.startTerraformParser(); err != nil {
			return fmt.Errorf("failed to start Terraform parser: %w", err)
		}
	}
	
	// Start diff worker
	if wpm.config.DiffWorker.Enabled {
		if err := wpm.startDiffWorker(); err != nil {
			return fmt.Errorf("failed to start diff worker: %w", err)
		}
	}
	
	// Start storage manager
	if wpm.config.StorageManager.Enabled {
		if err := wpm.startStorageManager(); err != nil {
			return fmt.Errorf("failed to start storage manager: %w", err)
		}
	}
	
	// Start metrics collector
	if wpm.metricsCollector != nil {
		go wpm.metricsCollector.Start(wpm.ctx)
	}
	
	// Start health monitor
	go wpm.healthMonitor.Start(wpm.ctx)
	
	wpm.running = true
	return nil
}

// Stop stops all worker pools gracefully
func (wpm *WorkerPoolManager) Stop() error {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()
	
	if !wpm.running {
		return fmt.Errorf("worker pool manager is not running")
	}
	
	// Signal graceful shutdown
	close(wpm.gracefulStop)
	
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), wpm.shutdownTimeout)
	defer shutdownCancel()
	
	// Stop all worker pools
	var errors []error
	
	// Stop resource processor
	if wpm.resourceProcessor != nil {
		if err := wpm.resourceProcessor.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("resource processor stop error: %w", err))
		}
	}
	
	// Stop Terraform parser
	if wpm.terraformParser != nil {
		wpm.terraformParser.stop()
	}
	
	// Stop diff worker
	if wpm.diffWorker != nil {
		wpm.diffWorker.stop()
	}
	
	// Stop storage manager
	if wpm.storageManager != nil {
		wpm.storageManager.stop()
	}
	
	// Cancel main context
	wpm.cancel()
	
	// Wait for shutdown completion or timeout
	done := make(chan struct{})
	go func() {
		// Wait for all components to shut down
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()
	
	select {
	case <-done:
		// Normal shutdown
	case <-shutdownCtx.Done():
		errors = append(errors, fmt.Errorf("shutdown timeout exceeded"))
	}
	
	wpm.running = false
	
	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	
	return nil
}

// startResourceProcessor starts the resource processor
func (wpm *WorkerPoolManager) startResourceProcessor() error {
	config := wpm.config.ResourceProcessor
	
	options := []ProcessorOption{
		WithWorkerCount(config.WorkerCount),
		WithBufferSize(config.BufferSize),
		WithRetryConfig(config.MaxRetries, config.RetryDelay),
		WithTimeout(config.Timeout),
		WithMemoryLimit(config.MemoryLimit),
		WithRateLimit(config.RateLimit, config.RateBurst),
	}
	
	wpm.resourceProcessor = NewResourceProcessor(options...)
	
	return wpm.resourceProcessor.Start(wpm.ctx)
}

// startTerraformParser starts the Terraform parser
func (wpm *WorkerPoolManager) startTerraformParser() error {
	config := wpm.config.TerraformParser
	
	options := []TerraformParserOption{
		WithTerraformWorkerCount(config.WorkerCount),
		WithTerraformBufferSize(config.BufferSize),
		WithTerraformMaxFileSize(config.MaxFileSize),
		WithTerraformTimeout(config.Timeout),
	}
	
	wpm.terraformParser = NewConcurrentTerraformParser(options...)
	
	return wpm.terraformParser.start()
}

// startDiffWorker starts the diff worker
func (wpm *WorkerPoolManager) startDiffWorker() error {
	config := wpm.config.DiffWorker
	
	options := []DiffWorkerOption{
		WithDiffWorkerCount(config.WorkerCount),
		WithDiffBufferSize(config.BufferSize),
		WithDiffBatchSize(config.BatchSize),
		WithDiffTimeout(config.Timeout),
	}
	
	if config.CacheEnabled {
		options = append(options, WithComparisonCache(config.CacheTTL))
	}
	
	wpm.diffWorker = NewDiffWorker(options...)
	
	return wpm.diffWorker.start()
}

// startStorageManager starts the storage manager
func (wpm *WorkerPoolManager) startStorageManager() error {
	config := wpm.config.StorageManager
	
	options := []StorageManagerOption{
		WithStorageWorkerCount(config.WorkerCount),
		WithStorageBufferSize(config.BufferSize),
		WithStorageTimeout(config.Timeout),
		WithStorageMaxFileSize(config.MaxFileSize),
	}
	
	wpm.storageManager = NewConcurrentStorageManager(options...)
	
	return wpm.storageManager.start()
}

// GetResourceProcessor returns the resource processor
func (wpm *WorkerPoolManager) GetResourceProcessor() *ResourceProcessor {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()
	return wpm.resourceProcessor
}

// GetTerraformParser returns the Terraform parser
func (wpm *WorkerPoolManager) GetTerraformParser() *ConcurrentTerraformParser {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()
	return wpm.terraformParser
}

// GetDiffWorker returns the diff worker
func (wpm *WorkerPoolManager) GetDiffWorker() *DiffWorker {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()
	return wpm.diffWorker
}

// GetStorageManager returns the storage manager
func (wpm *WorkerPoolManager) GetStorageManager() *ConcurrentStorageManager {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()
	return wpm.storageManager
}

// GetMetrics returns current metrics
func (wpm *WorkerPoolManager) GetMetrics() WorkerPoolMetrics {
	if wpm.metricsCollector == nil {
		return WorkerPoolMetrics{}
	}
	
	wpm.metricsCollector.mu.RLock()
	defer wpm.metricsCollector.mu.RUnlock()
	return wpm.metricsCollector.metrics
}

// GetHealthStatus returns current health status
func (wpm *WorkerPoolManager) GetHealthStatus() HealthStatus {
	wpm.healthMonitor.mu.RLock()
	defer wpm.healthMonitor.mu.RUnlock()
	return wpm.healthMonitor.status
}

// IsRunning returns whether the manager is running
func (wpm *WorkerPoolManager) IsRunning() bool {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()
	return wpm.running
}

// ProcessResourcesConcurrently processes resources using the resource processor
func (wpm *WorkerPoolManager) ProcessResourcesConcurrently(resources []RawResource) ([]types.Resource, []error) {
	if wpm.resourceProcessor == nil {
		return nil, []error{fmt.Errorf("resource processor is not enabled")}
	}
	
	return wpm.resourceProcessor.ProcessResources(resources)
}

// ParseTerraformStatesConcurrently parses Terraform states using the parser
func (wpm *WorkerPoolManager) ParseTerraformStatesConcurrently(statePaths []string) ([]types.Resource, error) {
	if wpm.terraformParser == nil {
		return nil, fmt.Errorf("Terraform parser is not enabled")
	}
	
	return wpm.terraformParser.ParseStatesConcurrent(statePaths)
}

// ComputeDiffsConcurrently computes diffs using the diff worker
func (wpm *WorkerPoolManager) ComputeDiffsConcurrently(baseline, current *types.Snapshot) (*types.DriftReport, error) {
	if wpm.diffWorker == nil {
		return nil, fmt.Errorf("diff worker is not enabled")
	}
	
	return wpm.diffWorker.ComputeDiffsConcurrent(baseline, current)
}

// SaveSnapshotConcurrently saves a snapshot using the storage manager
func (wpm *WorkerPoolManager) SaveSnapshotConcurrently(snapshot *types.Snapshot) error {
	if wpm.storageManager == nil {
		return fmt.Errorf("storage manager is not enabled")
	}
	
	return wpm.storageManager.SaveSnapshotConcurrent(snapshot)
}

// LoadSnapshotConcurrently loads a snapshot using the storage manager
func (wpm *WorkerPoolManager) LoadSnapshotConcurrently(snapshotID string) (*types.Snapshot, error) {
	if wpm.storageManager == nil {
		return nil, fmt.Errorf("storage manager is not enabled")
	}
	
	return wpm.storageManager.LoadSnapshotConcurrent(snapshotID)
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start(ctx context.Context) {
	if !mc.enabled {
		return
	}
	
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all worker pools
func (mc *MetricsCollector) collectMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.metrics.Timestamp = time.Now()
	mc.metrics.SystemMetrics = mc.collectSystemMetrics()
	mc.lastCollection = time.Now()
}

// collectSystemMetrics collects system-level metrics
func (mc *MetricsCollector) collectSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return SystemMetrics{
		MemoryUsage:    int64(m.Alloc),
		GoroutineCount: runtime.NumGoroutine(),
		GCPauses:       int64(m.NumGC),
		HeapSize:       int64(m.HeapSys),
		StackSize:      int64(m.StackSys),
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start(ctx context.Context) {
	// Register health checks
	hm.registerHealthChecks()
	
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

// registerHealthChecks registers all health checks
func (hm *HealthMonitor) registerHealthChecks() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	hm.healthChecks = []HealthCheck{
		{
			Name:  "memory_usage",
			Check: hm.checkMemoryUsage,
		},
		{
			Name:  "goroutine_count",
			Check: hm.checkGoroutineCount,
		},
	}
}

// performHealthChecks performs all health checks
func (hm *HealthMonitor) performHealthChecks() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	var results []HealthCheckResult
	healthy := true
	
	for i := range hm.healthChecks {
		check := &hm.healthChecks[i]
		err := check.Check()
		check.LastResult = err
		check.LastChecked = time.Now()
		
		result := HealthCheckResult{
			Name:      check.Name,
			Healthy:   err == nil,
			Timestamp: time.Now(),
		}
		
		if err != nil {
			result.Error = err.Error()
			healthy = false
		}
		
		results = append(results, result)
	}
	
	hm.status.Healthy = healthy
	hm.status.Timestamp = time.Now()
	hm.status.Checks = results
	hm.status.Uptime = time.Since(hm.status.StartTime)
}

// checkMemoryUsage checks memory usage
func (hm *HealthMonitor) checkMemoryUsage() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Check if memory usage is above 80% of limit
	memoryLimit := int64(500 * 1024 * 1024) // 500MB default
	if int64(m.Alloc) > memoryLimit*8/10 {
		return fmt.Errorf("memory usage %d exceeds 80%% of limit %d", m.Alloc, memoryLimit)
	}
	
	return nil
}

// checkGoroutineCount checks goroutine count
func (hm *HealthMonitor) checkGoroutineCount() error {
	count := runtime.NumGoroutine()
	
	// Check if goroutine count is above threshold
	if count > 1000 {
		return fmt.Errorf("goroutine count %d exceeds threshold 1000", count)
	}
	
	return nil
}

// DefaultWorkerPoolConfig returns a default configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		ResourceProcessor: ResourceProcessorConfig{
			Enabled:     true,
			WorkerCount: runtime.NumCPU(),
			BufferSize:  100,
			MaxRetries:  3,
			RetryDelay:  100 * time.Millisecond,
			Timeout:     30 * time.Second,
			MemoryLimit: 100 * 1024 * 1024, // 100MB
			RateLimit:   1000,
			RateBurst:   100,
		},
		TerraformParser: TerraformParserConfig{
			Enabled:       true,
			WorkerCount:   runtime.NumCPU(),
			BufferSize:    100,
			MaxFileSize:   500 * 1024 * 1024, // 500MB
			Timeout:       60 * time.Second,
			StreamingMode: true,
		},
		DiffWorker: DiffWorkerConfig{
			Enabled:      true,
			WorkerCount:  runtime.NumCPU(),
			BufferSize:   100,
			BatchSize:    10,
			Timeout:      30 * time.Second,
			CacheEnabled: true,
			CacheTTL:     5 * time.Minute,
		},
		StorageManager: StorageManagerConfig{
			Enabled:     true,
			WorkerCount: runtime.NumCPU(),
			BufferSize:  100,
			Timeout:     60 * time.Second,
			MaxFileSize: 1024 * 1024 * 1024, // 1GB
			Compression: true,
			Encryption:  false,
			Backup:      true,
		},
		MaxConcurrency:  runtime.NumCPU() * 2,
		DefaultTimeout:  30 * time.Second,
		MemoryLimit:     500 * 1024 * 1024, // 500MB
		EnableMetrics:   true,
		MetricsInterval: 30 * time.Second,
		HealthCheckPort: 8080,
	}
}