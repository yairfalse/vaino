package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// WorkerPoolIntegration demonstrates how to integrate all worker pools
type WorkerPoolIntegration struct {
	// Core worker pools
	resourceProcessor *ResourceProcessor
	terraformParser   *ConcurrentTerraformParser
	diffWorker        *DiffWorker
	storageManager    *ConcurrentStorageManager

	// Management systems
	poolManager     *WorkerPoolManager
	scalableManager *ScalableWorkerManager
	memoryOptimizer *MemoryOptimizer

	// Configuration
	config IntegrationConfig

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// IntegrationConfig configures the integrated worker system
type IntegrationConfig struct {
	// Worker pool configurations
	WorkerPoolConfig   WorkerPoolConfig         `yaml:"worker_pool"`
	ScalableConfig     ScalableWorkerConfig     `yaml:"scalable_worker"`
	MemoryOptimization MemoryOptimizationConfig `yaml:"memory_optimization"`

	// Integration settings
	EnableScaling   bool `yaml:"enable_scaling"`
	EnableMemoryOpt bool `yaml:"enable_memory_optimization"`
	EnableMetrics   bool `yaml:"enable_metrics"`

	// Performance settings
	MaxConcurrentOps int           `yaml:"max_concurrent_operations"`
	DefaultTimeout   time.Duration `yaml:"default_timeout"`

	// Monitoring settings
	MonitoringEnabled  bool          `yaml:"monitoring_enabled"`
	MonitoringInterval time.Duration `yaml:"monitoring_interval"`
	HealthCheckEnabled bool          `yaml:"health_check_enabled"`
}

// NewWorkerPoolIntegration creates a new integrated worker pool system
func NewWorkerPoolIntegration(config IntegrationConfig) *WorkerPoolIntegration {
	ctx, cancel := context.WithCancel(context.Background())

	integration := &WorkerPoolIntegration{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize worker pool manager
	integration.poolManager = NewWorkerPoolManager(config.WorkerPoolConfig)

	// Initialize scalable manager if enabled
	if config.EnableScaling {
		integration.scalableManager = NewScalableWorkerManager(config.ScalableConfig)
	}

	// Initialize memory optimizer if enabled
	if config.EnableMemoryOpt {
		integration.memoryOptimizer = NewMemoryOptimizer(config.MemoryOptimization)
	}

	return integration
}

// Start starts the integrated worker pool system
func (wpi *WorkerPoolIntegration) Start(ctx context.Context) error {
	log.Println("Starting integrated worker pool system...")

	// Start worker pool manager
	if err := wpi.poolManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool manager: %w", err)
	}

	// Start scalable manager if enabled
	if wpi.scalableManager != nil {
		if err := wpi.scalableManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start scalable manager: %w", err)
		}
	}

	// Start memory optimizer if enabled
	if wpi.memoryOptimizer != nil {
		if err := wpi.memoryOptimizer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start memory optimizer: %w", err)
		}
	}

	log.Println("Integrated worker pool system started successfully")
	return nil
}

// Stop stops the integrated worker pool system
func (wpi *WorkerPoolIntegration) Stop() error {
	log.Println("Stopping integrated worker pool system...")

	// Stop components in reverse order
	if wpi.memoryOptimizer != nil {
		if err := wpi.memoryOptimizer.Stop(); err != nil {
			log.Printf("Error stopping memory optimizer: %v", err)
		}
	}

	if wpi.scalableManager != nil {
		if err := wpi.scalableManager.Stop(); err != nil {
			log.Printf("Error stopping scalable manager: %v", err)
		}
	}

	if err := wpi.poolManager.Stop(); err != nil {
		log.Printf("Error stopping worker pool manager: %v", err)
	}

	// Cancel context
	wpi.cancel()

	log.Println("Integrated worker pool system stopped")
	return nil
}

// ProcessResourcesConcurrently processes resources with optimal resource allocation
func (wpi *WorkerPoolIntegration) ProcessResourcesConcurrently(resources []RawResource) ([]types.Resource, []error) {
	// Use memory optimizer for resource allocation if enabled
	if wpi.memoryOptimizer != nil {
		// Check if we should use streaming for large datasets
		dataSize := int64(len(resources) * 1024) // Estimate data size
		if wpi.memoryOptimizer.ShouldUseStreaming(dataSize) {
			return wpi.processResourcesStreaming(resources)
		}

		// Check for backpressure
		if err := wpi.memoryOptimizer.WaitForBackpressure(wpi.ctx); err != nil {
			return nil, []error{fmt.Errorf("backpressure timeout: %w", err)}
		}
	}

	// Use scalable manager if available, otherwise use regular pool manager
	if wpi.scalableManager != nil {
		return wpi.processResourcesScalable(resources)
	}

	return wpi.poolManager.ProcessResourcesConcurrently(resources)
}

// processResourcesStreaming processes resources using streaming approach
func (wpi *WorkerPoolIntegration) processResourcesStreaming(resources []RawResource) ([]types.Resource, []error) {
	batchSize := 100
	var allResults []types.Resource
	var allErrors []error

	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		results, errors := wpi.poolManager.ProcessResourcesConcurrently(batch)

		allResults = append(allResults, results...)
		allErrors = append(allErrors, errors...)

		// Brief pause between batches to prevent overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	return allResults, allErrors
}

// processResourcesScalable processes resources using scalable worker pools
func (wpi *WorkerPoolIntegration) processResourcesScalable(resources []RawResource) ([]types.Resource, []error) {
	// Submit work items to scalable manager
	var results []types.Resource
	var errors []error

	resultChan := make(chan WorkResult, len(resources))

	// Submit all resources as work items
	for _, resource := range resources {
		workItem := WorkItem{
			ID:        resource.ID,
			Type:      WorkTypeResourceProcessing,
			Priority:  100,
			Data:      resource,
			Timeout:   30 * time.Second,
			CreatedAt: time.Now(),
			Context:   wpi.ctx,
		}

		if err := wpi.scalableManager.SubmitWork(workItem); err != nil {
			errors = append(errors, fmt.Errorf("failed to submit work for %s: %w", resource.ID, err))
		}
	}

	// Collect results
	for i := 0; i < len(resources); i++ {
		select {
		case result := <-resultChan:
			if result.Success {
				if resource, ok := result.Result.(types.Resource); ok {
					results = append(results, resource)
				}
			} else {
				errors = append(errors, result.Error)
			}
		case <-time.After(wpi.config.DefaultTimeout):
			errors = append(errors, fmt.Errorf("timeout waiting for results"))
		}
	}

	return results, errors
}

// ParseTerraformStatesConcurrently parses Terraform states with optimizations
func (wpi *WorkerPoolIntegration) ParseTerraformStatesConcurrently(statePaths []string) ([]types.Resource, error) {
	// Use memory-optimized parsing for large state files
	if wpi.memoryOptimizer != nil {
		// Check total estimated size
		totalSize := int64(len(statePaths) * 10 * 1024 * 1024) // Estimate 10MB per state file
		if wpi.memoryOptimizer.ShouldUseStreaming(totalSize) {
			return wpi.parseStatesStreaming(statePaths)
		}
	}

	return wpi.poolManager.ParseTerraformStatesConcurrently(statePaths)
}

// parseStatesStreaming parses state files using streaming approach
func (wpi *WorkerPoolIntegration) parseStatesStreaming(statePaths []string) ([]types.Resource, error) {
	batchSize := 5 // Process 5 state files at a time
	var allResources []types.Resource

	for i := 0; i < len(statePaths); i += batchSize {
		end := i + batchSize
		if end > len(statePaths) {
			end = len(statePaths)
		}

		batch := statePaths[i:end]
		resources, err := wpi.poolManager.ParseTerraformStatesConcurrently(batch)
		if err != nil {
			return allResources, fmt.Errorf("failed to parse batch %d-%d: %w", i, end, err)
		}

		allResources = append(allResources, resources...)

		// Brief pause between batches
		time.Sleep(100 * time.Millisecond)
	}

	return allResources, nil
}

// ComputeDiffsConcurrently computes diffs with optimal resource usage
func (wpi *WorkerPoolIntegration) ComputeDiffsConcurrently(baseline, current *types.Snapshot) (*types.DriftReport, error) {
	// Use memory-optimized diff computation for large snapshots
	if wpi.memoryOptimizer != nil {
		// Check if we should use resource batching
		resourceCount := len(baseline.Resources) + len(current.Resources)
		if resourceCount > wpi.config.MemoryOptimization.MaxResourcesInMemory {
			return wpi.computeDiffsBatched(baseline, current)
		}

		// Check for backpressure
		if err := wpi.memoryOptimizer.WaitForBackpressure(wpi.ctx); err != nil {
			return nil, fmt.Errorf("backpressure timeout during diff computation: %w", err)
		}
	}

	return wpi.poolManager.ComputeDiffsConcurrently(baseline, current)
}

// computeDiffsBatched computes diffs using batched approach
func (wpi *WorkerPoolIntegration) computeDiffsBatched(baseline, current *types.Snapshot) (*types.DriftReport, error) {
	batchSize := wpi.config.MemoryOptimization.ResourceBatchSize

	// Split snapshots into batches
	baselineBatches := wpi.splitSnapshotIntoBatches(baseline, batchSize)
	currentBatches := wpi.splitSnapshotIntoBatches(current, batchSize)

	// Process each batch pair
	var allChanges []types.Change
	var totalSummary types.DriftSummary

	for i := 0; i < len(baselineBatches) || i < len(currentBatches); i++ {
		var baselineBatch, currentBatch *types.Snapshot

		if i < len(baselineBatches) {
			baselineBatch = baselineBatches[i]
		}
		if i < len(currentBatches) {
			currentBatch = currentBatches[i]
		}

		// Compute diff for this batch
		batchReport, err := wpi.poolManager.ComputeDiffsConcurrently(baselineBatch, currentBatch)
		if err != nil {
			return nil, fmt.Errorf("failed to compute diff for batch %d: %w", i, err)
		}

		// Aggregate results
		allChanges = append(allChanges, batchReport.Changes...)
		totalSummary.TotalChanges += batchReport.Summary.TotalChanges
		totalSummary.AddedResources += batchReport.Summary.AddedResources
		totalSummary.DeletedResources += batchReport.Summary.DeletedResources
		totalSummary.ModifiedResources += batchReport.Summary.ModifiedResources
		totalSummary.HighRiskChanges += batchReport.Summary.HighRiskChanges

		// Update risk score (weighted average)
		totalSummary.RiskScore = (totalSummary.RiskScore + batchReport.Summary.RiskScore) / 2
	}

	// Create final report
	return &types.DriftReport{
		ID:         fmt.Sprintf("batched-diff-%d", time.Now().Unix()),
		Timestamp:  time.Now(),
		BaselineID: baseline.ID,
		CurrentID:  current.ID,
		Changes:    allChanges,
		Summary:    totalSummary,
	}, nil
}

// splitSnapshotIntoBatches splits a snapshot into smaller batches
func (wpi *WorkerPoolIntegration) splitSnapshotIntoBatches(snapshot *types.Snapshot, batchSize int) []*types.Snapshot {
	if snapshot == nil {
		return nil
	}

	var batches []*types.Snapshot

	for i := 0; i < len(snapshot.Resources); i += batchSize {
		end := i + batchSize
		if end > len(snapshot.Resources) {
			end = len(snapshot.Resources)
		}

		batch := &types.Snapshot{
			ID:        fmt.Sprintf("%s-batch-%d", snapshot.ID, i/batchSize),
			Timestamp: snapshot.Timestamp,
			Provider:  snapshot.Provider,
			Resources: snapshot.Resources[i:end],
			Metadata:  snapshot.Metadata,
		}

		batches = append(batches, batch)
	}

	return batches
}

// SaveSnapshotConcurrently saves a snapshot with optimal storage
func (wpi *WorkerPoolIntegration) SaveSnapshotConcurrently(snapshot *types.Snapshot) error {
	// Use memory-optimized storage if available
	if wpi.memoryOptimizer != nil {
		// Check if we should use streaming for large snapshots
		estimatedSize := int64(len(snapshot.Resources) * 1024) // Estimate 1KB per resource
		if wpi.memoryOptimizer.ShouldUseStreaming(estimatedSize) {
			return wpi.saveSnapshotStreaming(snapshot)
		}
	}

	return wpi.poolManager.SaveSnapshotConcurrently(snapshot)
}

// saveSnapshotStreaming saves a snapshot using streaming approach
func (wpi *WorkerPoolIntegration) saveSnapshotStreaming(snapshot *types.Snapshot) error {
	// Split snapshot into chunks for streaming
	chunkSize := 1000 // 1000 resources per chunk

	for i := 0; i < len(snapshot.Resources); i += chunkSize {
		end := i + chunkSize
		if end > len(snapshot.Resources) {
			end = len(snapshot.Resources)
		}

		chunk := &types.Snapshot{
			ID:        fmt.Sprintf("%s-chunk-%d", snapshot.ID, i/chunkSize),
			Timestamp: snapshot.Timestamp,
			Provider:  snapshot.Provider,
			Resources: snapshot.Resources[i:end],
			Metadata:  snapshot.Metadata,
		}

		if err := wpi.poolManager.SaveSnapshotConcurrently(chunk); err != nil {
			return fmt.Errorf("failed to save chunk %d: %w", i/chunkSize, err)
		}
	}

	return nil
}

// GetComprehensiveMetrics returns metrics from all components
func (wpi *WorkerPoolIntegration) GetComprehensiveMetrics() ComprehensiveMetrics {
	metrics := ComprehensiveMetrics{
		Timestamp: time.Now(),
	}

	// Get worker pool metrics
	metrics.WorkerPool = wpi.poolManager.GetMetrics()

	// Get scalable manager metrics if available
	if wpi.scalableManager != nil {
		metrics.ScalableManager = wpi.scalableManager.GetPerformanceMetrics()
	}

	// Get memory optimization metrics if available
	if wpi.memoryOptimizer != nil {
		metrics.MemoryOptimization = wpi.memoryOptimizer.GetStats()
	}

	return metrics
}

// ComprehensiveMetrics contains metrics from all components
type ComprehensiveMetrics struct {
	Timestamp          time.Time               `json:"timestamp"`
	WorkerPool         WorkerPoolMetrics       `json:"worker_pool"`
	ScalableManager    *PerformanceMetrics     `json:"scalable_manager,omitempty"`
	MemoryOptimization MemoryOptimizationStats `json:"memory_optimization,omitempty"`
}

// Example usage function
func ExampleUsage() {
	// Create configuration
	config := IntegrationConfig{
		WorkerPoolConfig:   DefaultWorkerPoolConfig(),
		ScalableConfig:     DefaultScalableWorkerConfig(),
		MemoryOptimization: DefaultMemoryOptimizationConfig(),
		EnableScaling:      true,
		EnableMemoryOpt:    true,
		EnableMetrics:      true,
		MaxConcurrentOps:   1000,
		DefaultTimeout:     30 * time.Second,
		MonitoringEnabled:  true,
		MonitoringInterval: 30 * time.Second,
		HealthCheckEnabled: true,
	}

	// Create integrated worker pool system
	integration := NewWorkerPoolIntegration(config)

	// Start the system
	ctx := context.Background()
	if err := integration.Start(ctx); err != nil {
		log.Fatalf("Failed to start integration: %v", err)
	}
	defer integration.Stop()

	// Example: Process resources
	resources := []RawResource{
		{
			ID:       "resource-1",
			Type:     "aws_instance",
			Provider: "aws",
			Data: map[string]interface{}{
				"instance_type": "t3.micro",
				"region":        "us-east-1",
			},
		},
		{
			ID:       "resource-2",
			Type:     "aws_s3_bucket",
			Provider: "aws",
			Data: map[string]interface{}{
				"bucket_name": "my-bucket",
				"region":      "us-east-1",
			},
		},
	}

	results, errors := integration.ProcessResourcesConcurrently(resources)
	if len(errors) > 0 {
		log.Printf("Processing errors: %v", errors)
	}

	log.Printf("Processed %d resources successfully", len(results))

	// Example: Parse Terraform states
	statePaths := []string{
		"/path/to/terraform.tfstate",
		"/path/to/another.tfstate",
	}

	terraformResources, err := integration.ParseTerraformStatesConcurrently(statePaths)
	if err != nil {
		log.Printf("Terraform parsing error: %v", err)
	} else {
		log.Printf("Parsed %d Terraform resources", len(terraformResources))
	}

	// Example: Get comprehensive metrics
	metrics := integration.GetComprehensiveMetrics()
	log.Printf("Current metrics: %+v", metrics)

	log.Println("Example completed successfully")
}

// DefaultIntegrationConfig returns a default integration configuration
func DefaultIntegrationConfig() IntegrationConfig {
	return IntegrationConfig{
		WorkerPoolConfig:   DefaultWorkerPoolConfig(),
		ScalableConfig:     DefaultScalableWorkerConfig(),
		MemoryOptimization: DefaultMemoryOptimizationConfig(),
		EnableScaling:      true,
		EnableMemoryOpt:    true,
		EnableMetrics:      true,
		MaxConcurrentOps:   1000,
		DefaultTimeout:     30 * time.Second,
		MonitoringEnabled:  true,
		MonitoringInterval: 30 * time.Second,
		HealthCheckEnabled: true,
	}
}
