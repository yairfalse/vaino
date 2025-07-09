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

// RawResource represents an unprocessed resource from a provider
type RawResource struct {
	ID       string
	Type     string
	Provider string
	Data     map[string]interface{}
	Metadata map[string]interface{}
}

// ProcessingResult holds the result of resource processing
type ProcessingResult struct {
	Resource    *types.Resource
	Error       error
	ProcessTime time.Duration
	WorkerID    int
}

// ResourceProcessor handles concurrent resource normalization and processing
type ResourceProcessor struct {
	workerCount    int
	inputChan      chan RawResource
	outputChan     chan ProcessingResult
	errorChan      chan error
	workers        []*resourceWorker
	processingFunc func(RawResource) (*types.Resource, error)

	// Metrics
	processedCount int64
	errorCount     int64
	totalTime      time.Duration
	mu             sync.RWMutex

	// Configuration
	maxRetries int
	retryDelay time.Duration
	bufferSize int
	timeout    time.Duration

	// State management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Memory management
	memoryLimit   int64
	currentMemory int64
	backpressure  bool
	rateLimiter   *RateLimiter
}

// resourceWorker represents a single worker goroutine
type resourceWorker struct {
	id        int
	processor *ResourceProcessor
	stats     workerStats
}

// workerStats holds statistics for a worker
type workerStats struct {
	processed  int64
	errors     int64
	totalTime  time.Duration
	lastActive time.Time
	mu         sync.RWMutex
}

// RateLimiter controls the rate of processing
type RateLimiter struct {
	rate   int
	burst  int
	tokens chan struct{}
	ticker *time.Ticker
	stop   chan struct{}
}

// NewResourceProcessor creates a new resource processor with worker pool
func NewResourceProcessor(opts ...ProcessorOption) *ResourceProcessor {
	// Default configuration
	p := &ResourceProcessor{
		workerCount: runtime.NumCPU(),
		bufferSize:  100,
		maxRetries:  3,
		retryDelay:  100 * time.Millisecond,
		timeout:     30 * time.Second,
		memoryLimit: 100 * 1024 * 1024,         // 100MB
		rateLimiter: NewRateLimiter(1000, 100), // 1000 ops/sec, burst 100
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}

	// Initialize channels
	p.inputChan = make(chan RawResource, p.bufferSize)
	p.outputChan = make(chan ProcessingResult, p.bufferSize)
	p.errorChan = make(chan error, p.bufferSize)

	// Create context
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Initialize workers
	p.workers = make([]*resourceWorker, p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		p.workers[i] = &resourceWorker{
			id:        i,
			processor: p,
		}
	}

	return p
}

// ProcessorOption configures the resource processor
type ProcessorOption func(*ResourceProcessor)

// WithWorkerCount sets the number of worker goroutines
func WithWorkerCount(count int) ProcessorOption {
	return func(p *ResourceProcessor) {
		if count > 0 {
			p.workerCount = count
		}
	}
}

// WithBufferSize sets the buffer size for channels
func WithBufferSize(size int) ProcessorOption {
	return func(p *ResourceProcessor) {
		if size > 0 {
			p.bufferSize = size
		}
	}
}

// WithRetryConfig sets retry configuration
func WithRetryConfig(maxRetries int, retryDelay time.Duration) ProcessorOption {
	return func(p *ResourceProcessor) {
		p.maxRetries = maxRetries
		p.retryDelay = retryDelay
	}
}

// WithTimeout sets processing timeout
func WithTimeout(timeout time.Duration) ProcessorOption {
	return func(p *ResourceProcessor) {
		p.timeout = timeout
	}
}

// WithMemoryLimit sets memory limit for backpressure
func WithMemoryLimit(limit int64) ProcessorOption {
	return func(p *ResourceProcessor) {
		p.memoryLimit = limit
	}
}

// WithRateLimit sets rate limiting
func WithRateLimit(rate, burst int) ProcessorOption {
	return func(p *ResourceProcessor) {
		p.rateLimiter = NewRateLimiter(rate, burst)
	}
}

// WithProcessingFunction sets the function to process raw resources
func WithProcessingFunction(fn func(RawResource) (*types.Resource, error)) ProcessorOption {
	return func(p *ResourceProcessor) {
		p.processingFunc = fn
	}
}

// Start begins processing with the worker pool
func (p *ResourceProcessor) Start(ctx context.Context) error {
	if p.processingFunc == nil {
		return fmt.Errorf("processing function must be set")
	}

	// Start rate limiter
	if p.rateLimiter != nil {
		p.rateLimiter.Start()
	}

	// Start workers
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Start memory monitor
	p.wg.Add(1)
	go p.memoryMonitor(ctx)

	return nil
}

// Stop gracefully shuts down the processor
func (p *ResourceProcessor) Stop() error {
	// Cancel context
	p.cancel()

	// Close input channel
	close(p.inputChan)

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Normal shutdown
	case <-time.After(10 * time.Second):
		// Force shutdown
	}

	// Stop rate limiter
	if p.rateLimiter != nil {
		p.rateLimiter.Stop()
	}

	// Close output channels
	close(p.outputChan)
	close(p.errorChan)

	return nil
}

// ProcessResource adds a raw resource to the processing queue
func (p *ResourceProcessor) ProcessResource(resource RawResource) error {
	select {
	case p.inputChan <- resource:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("processor is shutting down")
	case <-time.After(p.timeout):
		return fmt.Errorf("timeout adding resource to queue")
	}
}

// ProcessResources processes multiple resources concurrently
func (p *ResourceProcessor) ProcessResources(resources []RawResource) ([]types.Resource, []error) {
	var results []types.Resource
	var errors []error

	// Start processor if not already running
	if err := p.Start(p.ctx); err != nil {
		return nil, []error{err}
	}

	// Channel to collect results
	resultChan := make(chan ProcessingResult, len(resources))

	// Send all resources for processing
	for _, resource := range resources {
		go func(r RawResource) {
			if err := p.ProcessResource(r); err != nil {
				resultChan <- ProcessingResult{Error: err}
			}
		}(resource)
	}

	// Collect results
	processed := 0
	for processed < len(resources) {
		select {
		case result := <-p.outputChan:
			resultChan <- result
			processed++
		case err := <-p.errorChan:
			errors = append(errors, err)
			processed++
		case <-time.After(p.timeout):
			errors = append(errors, fmt.Errorf("timeout waiting for results"))
			return results, errors
		}
	}

	// Process collected results
	for i := 0; i < len(resources); i++ {
		result := <-resultChan
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else if result.Resource != nil {
			results = append(results, *result.Resource)
		}
	}

	return results, errors
}

// worker processes resources from the input channel
func (p *ResourceProcessor) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	worker := p.workers[workerID]

	for {
		select {
		case <-ctx.Done():
			return
		case resource, ok := <-p.inputChan:
			if !ok {
				return
			}

			// Apply rate limiting
			if p.rateLimiter != nil {
				if !p.rateLimiter.Allow() {
					continue
				}
			}

			// Check backpressure
			if p.backpressure {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Process resource with retries
			result := p.processResourceWithRetries(resource, workerID)

			// Update worker stats
			worker.updateStats(result)

			// Send result
			select {
			case p.outputChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// processResourceWithRetries processes a resource with retry logic
func (p *ResourceProcessor) processResourceWithRetries(resource RawResource, workerID int) ProcessingResult {
	startTime := time.Now()

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(p.retryDelay * time.Duration(attempt))
		}

		// Process with timeout
		ctx, cancel := context.WithTimeout(p.ctx, p.timeout)
		result := p.processResourceWithTimeout(ctx, resource, workerID)
		cancel()

		if result.Error == nil {
			result.ProcessTime = time.Since(startTime)
			atomic.AddInt64(&p.processedCount, 1)
			return result
		}

		lastErr = result.Error
	}

	// All retries failed
	atomic.AddInt64(&p.errorCount, 1)
	return ProcessingResult{
		Error:       fmt.Errorf("failed after %d attempts: %w", p.maxRetries+1, lastErr),
		ProcessTime: time.Since(startTime),
		WorkerID:    workerID,
	}
}

// processResourceWithTimeout processes a resource with timeout
func (p *ResourceProcessor) processResourceWithTimeout(ctx context.Context, resource RawResource, workerID int) ProcessingResult {
	resultChan := make(chan ProcessingResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- ProcessingResult{
					Error:    fmt.Errorf("panic in worker %d: %v", workerID, r),
					WorkerID: workerID,
				}
			}
		}()

		processedResource, err := p.processingFunc(resource)
		resultChan <- ProcessingResult{
			Resource: processedResource,
			Error:    err,
			WorkerID: workerID,
		}
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return ProcessingResult{
			Error:    fmt.Errorf("processing timeout for worker %d", workerID),
			WorkerID: workerID,
		}
	}
}

// memoryMonitor monitors memory usage and applies backpressure
func (p *ResourceProcessor) memoryMonitor(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			currentMemory := int64(m.Alloc)
			atomic.StoreInt64(&p.currentMemory, currentMemory)

			// Apply backpressure if memory limit exceeded
			if currentMemory > p.memoryLimit {
				p.backpressure = true
				runtime.GC() // Force garbage collection
			} else {
				p.backpressure = false
			}
		}
	}
}

// updateStats updates worker statistics
func (w *resourceWorker) updateStats(result ProcessingResult) {
	w.stats.mu.Lock()
	defer w.stats.mu.Unlock()

	w.stats.lastActive = time.Now()
	w.stats.totalTime += result.ProcessTime

	if result.Error != nil {
		w.stats.errors++
	} else {
		w.stats.processed++
	}
}

// GetStats returns processing statistics
func (p *ResourceProcessor) GetStats() ProcessingStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := ProcessingStats{
		TotalProcessed:     atomic.LoadInt64(&p.processedCount),
		TotalErrors:        atomic.LoadInt64(&p.errorCount),
		WorkerCount:        p.workerCount,
		CurrentMemory:      atomic.LoadInt64(&p.currentMemory),
		MemoryLimit:        p.memoryLimit,
		BackpressureActive: p.backpressure,
		WorkerStats:        make([]WorkerStats, len(p.workers)),
	}

	for i, worker := range p.workers {
		worker.stats.mu.RLock()
		stats.WorkerStats[i] = WorkerStats{
			WorkerID:   i,
			Processed:  worker.stats.processed,
			Errors:     worker.stats.errors,
			TotalTime:  worker.stats.totalTime,
			LastActive: worker.stats.lastActive,
		}
		worker.stats.mu.RUnlock()
	}

	return stats
}

// ProcessingStats holds overall processing statistics
type ProcessingStats struct {
	TotalProcessed     int64
	TotalErrors        int64
	WorkerCount        int
	CurrentMemory      int64
	MemoryLimit        int64
	BackpressureActive bool
	WorkerStats        []WorkerStats
}

// WorkerStats holds individual worker statistics
type WorkerStats struct {
	WorkerID   int
	Processed  int64
	Errors     int64
	TotalTime  time.Duration
	LastActive time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		rate:   rate,
		burst:  burst,
		tokens: make(chan struct{}, burst),
		stop:   make(chan struct{}),
	}
}

// Start begins the rate limiter
func (rl *RateLimiter) Start() {
	// Fill initial tokens
	for i := 0; i < rl.burst; i++ {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}

	// Start token refill
	if rl.rate > 0 {
		interval := time.Second / time.Duration(rl.rate)
		rl.ticker = time.NewTicker(interval)
		go func() {
			for {
				select {
				case <-rl.ticker.C:
					select {
					case rl.tokens <- struct{}{}:
					default:
					}
				case <-rl.stop:
					return
				}
			}
		}()
	}
}

// Allow checks if an operation is allowed
func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stop)
	if rl.ticker != nil {
		rl.ticker.Stop()
	}
}
