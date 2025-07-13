package exports

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of workers for processing export requests
type WorkerPool struct {
	mu             sync.RWMutex
	workers        []*Worker
	jobQueue       chan *ExportJob
	resultQueue    chan *ExportResult
	maxWorkers     int
	workerTimeout  time.Duration
	running        bool
	shutdown       chan struct{}
	activeJobs     int64
	totalJobs      int64
	completedJobs  int64
	failedJobs     int64
	startTime      time.Time
	resultCallback func(*ExportResult) // Callback for handling job results
}

// Worker represents a single worker in the pool
type Worker struct {
	id          int
	pool        *WorkerPool
	jobQueue    chan *ExportJob
	resultQueue chan *ExportResult
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	currentJob  *ExportJob
	startTime   time.Time
	jobsHandled int64
}

// ExportJob represents a job to be processed by a worker
type ExportJob struct {
	ID         string
	Request    *ExportRequest
	Plugin     ExportPlugin
	StartTime  time.Time
	Timeout    time.Duration
	RetryCount int
	MaxRetries int
}

// ExportResult represents the result of processing an export job
type ExportResult struct {
	Job      *ExportJob
	Response *ExportResponse
	Error    error
	Duration time.Duration
	WorkerID int
	EndTime  time.Time
}

// WorkerStats contains statistics for a worker
type WorkerStats struct {
	ID              int           `json:"id"`
	Running         bool          `json:"running"`
	JobsHandled     int64         `json:"jobs_handled"`
	CurrentJob      string        `json:"current_job,omitempty"`
	Uptime          time.Duration `json:"uptime"`
	LastJobDuration time.Duration `json:"last_job_duration"`
	AverageJobTime  time.Duration `json:"average_job_time"`
}

// PoolStats contains statistics for the worker pool
type PoolStats struct {
	MaxWorkers    int           `json:"max_workers"`
	ActiveWorkers int           `json:"active_workers"`
	IdleWorkers   int           `json:"idle_workers"`
	QueueSize     int           `json:"queue_size"`
	QueueCapacity int           `json:"queue_capacity"`
	ActiveJobs    int64         `json:"active_jobs"`
	TotalJobs     int64         `json:"total_jobs"`
	CompletedJobs int64         `json:"completed_jobs"`
	FailedJobs    int64         `json:"failed_jobs"`
	Uptime        time.Duration `json:"uptime"`
	ThroughputJPS float64       `json:"throughput_jobs_per_second"`
	SuccessRate   float64       `json:"success_rate"`
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int, workerTimeout time.Duration) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	if workerTimeout <= 0 {
		workerTimeout = 5 * time.Minute
	}

	return &WorkerPool{
		workers:       make([]*Worker, 0, maxWorkers),
		jobQueue:      make(chan *ExportJob, maxWorkers*2),
		resultQueue:   make(chan *ExportResult, maxWorkers*2),
		maxWorkers:    maxWorkers,
		workerTimeout: workerTimeout,
		shutdown:      make(chan struct{}),
		startTime:     time.Now(),
	}
}

// SetResultCallback sets a callback function to handle job results
func (wp *WorkerPool) SetResultCallback(callback func(*ExportResult)) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.resultCallback = callback
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return fmt.Errorf("worker pool already running")
	}

	// Create and start workers
	for i := 0; i < wp.maxWorkers; i++ {
		worker := wp.createWorker(i, ctx)
		wp.workers = append(wp.workers, worker)
		go worker.start()
	}

	wp.running = true
	wp.startTime = time.Now()

	// Start result processor
	go wp.processResults(ctx)

	return nil
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return nil
	}

	// Signal shutdown
	close(wp.shutdown)

	// Stop all workers
	for _, worker := range wp.workers {
		worker.stop()
	}

	// Close channels
	close(wp.jobQueue)
	close(wp.resultQueue)

	wp.running = false
	return nil
}

// SubmitJob submits a job to the worker pool
func (wp *WorkerPool) SubmitJob(job *ExportJob) error {
	if !wp.running {
		return fmt.Errorf("worker pool not running")
	}

	job.StartTime = time.Now()
	atomic.AddInt64(&wp.totalJobs, 1)
	atomic.AddInt64(&wp.activeJobs, 1)

	select {
	case wp.jobQueue <- job:
		return nil
	default:
		atomic.AddInt64(&wp.activeJobs, -1)
		return fmt.Errorf("job queue full")
	}
}

// GetStats returns worker pool statistics
func (wp *WorkerPool) GetStats() PoolStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	activeWorkers := 0
	idleWorkers := 0

	for _, worker := range wp.workers {
		if worker.running {
			if worker.currentJob != nil {
				activeWorkers++
			} else {
				idleWorkers++
			}
		}
	}

	uptime := time.Since(wp.startTime)
	totalJobs := atomic.LoadInt64(&wp.totalJobs)
	completedJobs := atomic.LoadInt64(&wp.completedJobs)
	failedJobs := atomic.LoadInt64(&wp.failedJobs)

	var throughput float64
	if uptime.Seconds() > 0 {
		throughput = float64(completedJobs) / uptime.Seconds()
	}

	var successRate float64
	if totalJobs > 0 {
		successRate = float64(completedJobs) / float64(totalJobs) * 100
	}

	return PoolStats{
		MaxWorkers:    wp.maxWorkers,
		ActiveWorkers: activeWorkers,
		IdleWorkers:   idleWorkers,
		QueueSize:     len(wp.jobQueue),
		QueueCapacity: cap(wp.jobQueue),
		ActiveJobs:    atomic.LoadInt64(&wp.activeJobs),
		TotalJobs:     totalJobs,
		CompletedJobs: completedJobs,
		FailedJobs:    failedJobs,
		Uptime:        uptime,
		ThroughputJPS: throughput,
		SuccessRate:   successRate,
	}
}

// GetWorkerStats returns statistics for all workers
func (wp *WorkerPool) GetWorkerStats() []WorkerStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	stats := make([]WorkerStats, len(wp.workers))
	for i, worker := range wp.workers {
		stats[i] = worker.getStats()
	}

	return stats
}

// IsRunning returns whether the worker pool is running
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.running
}

// createWorker creates a new worker
func (wp *WorkerPool) createWorker(id int, ctx context.Context) *Worker {
	workerCtx, cancel := context.WithCancel(ctx)

	return &Worker{
		id:          id,
		pool:        wp,
		jobQueue:    wp.jobQueue,
		resultQueue: wp.resultQueue,
		ctx:         workerCtx,
		cancel:      cancel,
		startTime:   time.Now(),
	}
}

// processResults processes job results
func (wp *WorkerPool) processResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.shutdown:
			return
		case result := <-wp.resultQueue:
			wp.handleResult(result)
		}
	}
}

// handleResult handles a single job result
func (wp *WorkerPool) handleResult(result *ExportResult) {
	if result == nil {
		return
	}

	atomic.AddInt64(&wp.activeJobs, -1)

	if result.Error != nil {
		atomic.AddInt64(&wp.failedJobs, 1)

		// Check if job should be retried
		if result.Job != nil && result.Job.RetryCount < result.Job.MaxRetries {
			result.Job.RetryCount++
			// Resubmit job for retry
			go func() {
				time.Sleep(time.Second * time.Duration(result.Job.RetryCount)) // Exponential backoff
				wp.SubmitJob(result.Job)
			}()
			return
		}
	} else {
		atomic.AddInt64(&wp.completedJobs, 1)
	}

	// Call result callback if set
	wp.mu.RLock()
	callback := wp.resultCallback
	wp.mu.RUnlock()

	if callback != nil {
		callback(result)
	}
}

// Worker methods

// start starts the worker
func (w *Worker) start() {
	w.running = true
	w.startTime = time.Now()

	for {
		select {
		case <-w.ctx.Done():
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				return
			}
			w.processJob(job)
		}
	}
}

// stop stops the worker
func (w *Worker) stop() {
	w.running = false
	w.cancel()
}

// processJob processes a single job
func (w *Worker) processJob(job *ExportJob) {
	w.currentJob = job
	startTime := time.Now()

	// Create timeout context for the job
	jobCtx, cancel := context.WithTimeout(w.ctx, job.Timeout)
	defer cancel()

	// Process the export
	response, err := w.executeExport(jobCtx, job)
	duration := time.Since(startTime)

	// Create result
	result := &ExportResult{
		Job:      job,
		Response: response,
		Error:    err,
		Duration: duration,
		WorkerID: w.id,
		EndTime:  time.Now(),
	}

	// Update worker stats
	atomic.AddInt64(&w.jobsHandled, 1)
	w.currentJob = nil

	// Send result
	select {
	case w.resultQueue <- result:
	case <-w.ctx.Done():
		return
	}
}

// executeExport executes the actual export operation
func (w *Worker) executeExport(ctx context.Context, job *ExportJob) (*ExportResponse, error) {
	// Execute the plugin export
	response, err := job.Plugin.Export(ctx, job.Request)
	if response == nil {
		response = &ExportResponse{
			ID:          job.Request.ID,
			PluginName:  job.Plugin.Name(),
			ProcessedAt: time.Now(),
			Duration:    time.Since(job.StartTime),
			Metadata:    make(map[string]interface{}),
			Metrics: ExportMetrics{
				ProcessingTime: time.Since(job.StartTime),
				RetryCount:     job.RetryCount,
			},
		}
	}

	if err != nil {
		response.Status = StatusFailed
		response.Error = err.Error()
	} else {
		response.Status = StatusCompleted
	}

	return response, err
}

// getStats returns worker statistics
func (w *Worker) getStats() WorkerStats {
	var currentJobID string
	if w.currentJob != nil {
		currentJobID = w.currentJob.ID
	}

	var averageJobTime time.Duration
	jobsHandled := atomic.LoadInt64(&w.jobsHandled)
	if jobsHandled > 0 {
		averageJobTime = time.Since(w.startTime) / time.Duration(jobsHandled)
	}

	return WorkerStats{
		ID:             w.id,
		Running:        w.running,
		JobsHandled:    jobsHandled,
		CurrentJob:     currentJobID,
		Uptime:         time.Since(w.startTime),
		AverageJobTime: averageJobTime,
	}
}

// MetricsCollector collects and aggregates metrics from plugins and the system
type MetricsCollector struct {
	mu              sync.RWMutex
	plugins         map[string]ExportPlugin
	pluginMetrics   map[string]*PluginMetrics
	systemMetrics   *SystemMetrics
	config          MetricsConfig
	shutdown        chan struct{}
	running         bool
	startTime       time.Time
	collectionCount int64
}

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	CollectionInterval    time.Duration `json:"collection_interval"`
	RetentionPeriod       time.Duration `json:"retention_period"`
	BufferSize            int           `json:"buffer_size"`
	EnableDetailedMetrics bool          `json:"enable_detailed_metrics"`
	ExportMetrics         bool          `json:"export_metrics"`
	MetricsEndpoint       string        `json:"metrics_endpoint"`
}

// SystemMetrics contains system-wide metrics
type SystemMetrics struct {
	TotalExports      int64         `json:"total_exports"`
	SuccessfulExports int64         `json:"successful_exports"`
	FailedExports     int64         `json:"failed_exports"`
	AverageLatency    time.Duration `json:"average_latency"`
	TotalBytes        int64         `json:"total_bytes"`
	ErrorRate         float64       `json:"error_rate"`
	ThroughputMBps    float64       `json:"throughput_mbps"`
	Uptime            time.Duration `json:"uptime"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(interval, retention time.Duration) *MetricsCollector {
	config := MetricsConfig{
		CollectionInterval:    interval,
		RetentionPeriod:       retention,
		BufferSize:            1000,
		EnableDetailedMetrics: true,
		ExportMetrics:         false,
	}

	return &MetricsCollector{
		plugins:       make(map[string]ExportPlugin),
		pluginMetrics: make(map[string]*PluginMetrics),
		systemMetrics: &SystemMetrics{},
		config:        config,
		shutdown:      make(chan struct{}),
		startTime:     time.Now(),
	}
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return fmt.Errorf("metrics collector already running")
	}

	mc.running = true
	mc.startTime = time.Now()

	// Start collection loop
	go mc.collectionLoop(ctx)

	return nil
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return nil
	}

	close(mc.shutdown)
	mc.running = false

	return nil
}

// RegisterPlugin registers a plugin for metrics collection
func (mc *MetricsCollector) RegisterPlugin(name string, plugin ExportPlugin) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.plugins[name] = plugin

	// Get initial metrics from plugin
	initialMetrics := plugin.GetMetrics()
	mc.pluginMetrics[name] = &initialMetrics
}

// UnregisterPlugin removes a plugin from metrics collection
func (mc *MetricsCollector) UnregisterPlugin(name string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.plugins, name)
	delete(mc.pluginMetrics, name)
}

// GetPluginMetrics returns metrics for a specific plugin
func (mc *MetricsCollector) GetPluginMetrics(pluginName string) PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, exists := mc.pluginMetrics[pluginName]; exists {
		return *metrics
	}

	return PluginMetrics{}
}

// GetSystemMetrics returns system-wide metrics
func (mc *MetricsCollector) GetSystemMetrics() map[string]PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]PluginMetrics)
	for name, metrics := range mc.pluginMetrics {
		result[name] = *metrics
	}

	return result
}

// RecordSuccess records a successful export
func (mc *MetricsCollector) RecordSuccess(pluginName string, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if metrics, exists := mc.pluginMetrics[pluginName]; exists {
		metrics.TotalRequests++
		metrics.SuccessfulExports++
		metrics.LastExport = time.Now()

		// Update latency metrics (simplified)
		if metrics.AverageLatency == 0 {
			metrics.AverageLatency = duration
		} else {
			metrics.AverageLatency = (metrics.AverageLatency + duration) / 2
		}

		metrics.MetricsUpdatedAt = time.Now()
	}
}

// RecordError records a failed export
func (mc *MetricsCollector) RecordError(pluginName string, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if metrics, exists := mc.pluginMetrics[pluginName]; exists {
		metrics.TotalRequests++
		metrics.FailedExports++
		metrics.LastError = time.Now()

		// Calculate error rate
		if metrics.TotalRequests > 0 {
			metrics.ErrorRate = float64(metrics.FailedExports) / float64(metrics.TotalRequests) * 100
		}

		metrics.MetricsUpdatedAt = time.Now()
	}
}

// collectionLoop runs periodic metrics collection
func (mc *MetricsCollector) collectionLoop(ctx context.Context) {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.shutdown:
			return
		case <-ticker.C:
			mc.collectMetrics(ctx)
		}
	}
}

// collectMetrics collects metrics from all registered plugins
func (mc *MetricsCollector) collectMetrics(ctx context.Context) {
	mc.mu.RLock()
	plugins := make(map[string]ExportPlugin)
	for name, plugin := range mc.plugins {
		plugins[name] = plugin
	}
	mc.mu.RUnlock()

	// Collect from plugins
	for name, plugin := range plugins {
		go func(pluginName string, p ExportPlugin) {
			metrics := p.GetMetrics()

			mc.mu.Lock()
			mc.pluginMetrics[pluginName] = &metrics
			mc.mu.Unlock()
		}(name, plugin)
	}

	// Update system metrics
	mc.updateSystemMetrics()

	atomic.AddInt64(&mc.collectionCount, 1)
}

// updateSystemMetrics updates system-wide metrics
func (mc *MetricsCollector) updateSystemMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var totalExports, successfulExports, failedExports int64
	var totalLatency time.Duration
	var activePlugins int

	for _, metrics := range mc.pluginMetrics {
		totalExports += metrics.TotalRequests
		successfulExports += metrics.SuccessfulExports
		failedExports += metrics.FailedExports
		totalLatency += metrics.AverageLatency
		activePlugins++
	}

	var averageLatency time.Duration
	if activePlugins > 0 {
		averageLatency = totalLatency / time.Duration(activePlugins)
	}

	var errorRate float64
	if totalExports > 0 {
		errorRate = float64(failedExports) / float64(totalExports) * 100
	}

	mc.systemMetrics.TotalExports = totalExports
	mc.systemMetrics.SuccessfulExports = successfulExports
	mc.systemMetrics.FailedExports = failedExports
	mc.systemMetrics.AverageLatency = averageLatency
	mc.systemMetrics.ErrorRate = errorRate
	mc.systemMetrics.Uptime = time.Since(mc.startTime)
	mc.systemMetrics.LastUpdated = time.Now()
}

// Cleanup removes old metrics data
func (mc *MetricsCollector) Cleanup() {
	// In a production system, this would clean up historical data
	// For now, just update the collection timestamp
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.config.RetentionPeriod)

	for _, metrics := range mc.pluginMetrics {
		if metrics.MetricsUpdatedAt.Before(cutoff) {
			// Reset metrics for inactive plugins
			metrics.TotalRequests = 0
			metrics.SuccessfulExports = 0
			metrics.FailedExports = 0
		}
	}
}
