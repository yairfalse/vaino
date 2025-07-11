//go:build enterprise
// +build enterprise

package differ

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// EnterpriseDifferEngine is a high-performance diff engine for enterprise workloads
type EnterpriseDifferEngine struct {
	matcher           ResourceMatcher
	comparer          Comparer
	classifier        ChangeClassifier
	correlator        ChangeCorrelator
	riskAssessor      RiskAssessor
	options           EnterpriseDiffOptions
	workerPool        *WorkerPool
	changeStream      chan StreamedChange
	metrics           *DiffMetrics
	cache             *DiffCache
	resourceIndex     *ResourceIndex
	parallelThreshold int
}

// EnterpriseDiffOptions extends DiffOptions with enterprise features
type EnterpriseDiffOptions struct {
	DiffOptions
	// Parallel processing options
	MaxWorkers        int
	ParallelThreshold int // Min resources to trigger parallel processing
	StreamingEnabled  bool
	StreamBufferSize  int

	// Performance options
	EnableCaching       bool
	EnableIndexing      bool
	EnableSmartMatching bool
	EnableCorrelation   bool

	// Enterprise features
	EnableCompliance  bool
	ComplianceRules   []ComplianceRule
	EnableRiskScoring bool
	RiskScoringModel  string
	EnableBaseline    bool
	BaselineID        string

	// Output options
	OutputFormat       string
	EnableDebugMetrics bool
	ProgressCallback   func(progress float64, message string)
}

// DiffMetrics tracks performance and quality metrics
type DiffMetrics struct {
	StartTime          time.Time
	EndTime            time.Time
	TotalResources     int64
	ProcessedResources int64
	MatchedResources   int64
	AddedResources     int64
	RemovedResources   int64
	ModifiedResources  int64
	TotalChanges       int64
	HighRiskChanges    int64
	ComplianceIssues   int64
	ProcessingTimeMs   int64
	MemoryUsageMB      int64
	WorkersUsed        int32
	CacheHits          int64
	CacheMisses        int64
}

// StreamedChange represents a change that can be streamed in real-time
type StreamedChange struct {
	Change       Change
	ResourceDiff ResourceDiff
	Timestamp    time.Time
	SequenceID   int64
}

// NewEnterpriseDifferEngine creates a new enterprise-grade diff engine
func NewEnterpriseDifferEngine(options EnterpriseDiffOptions) *EnterpriseDifferEngine {
	// Set sensible defaults
	if options.MaxWorkers == 0 {
		options.MaxWorkers = runtime.NumCPU() * 2
	}
	if options.ParallelThreshold == 0 {
		options.ParallelThreshold = 100
	}
	if options.StreamBufferSize == 0 {
		options.StreamBufferSize = 1000
	}

	engine := &EnterpriseDifferEngine{
		options:           options,
		parallelThreshold: options.ParallelThreshold,
		metrics:           &DiffMetrics{},
	}

	// Initialize components based on options
	if options.EnableSmartMatching {
		engine.matcher = NewSmartResourceMatcher()
	} else {
		engine.matcher = &DefaultResourceMatcher{}
	}

	// Use smart comparer for better performance
	engine.comparer = NewSmartComparer(options.DiffOptions)

	// Use advanced classifier with ML support if specified
	if options.RiskScoringModel != "" {
		engine.classifier = NewAdvancedClassifier(options.RiskScoringModel)
	} else {
		engine.classifier = &DefaultClassifier{}
	}

	// Initialize correlation engine
	if options.EnableCorrelation {
		engine.correlator = NewChangeCorrelator()
	}

	// Initialize risk assessor
	if options.EnableRiskScoring {
		engine.riskAssessor = NewRiskAssessor()
	}

	// Initialize worker pool
	engine.workerPool = NewWorkerPool(options.MaxWorkers)

	// Initialize cache if enabled
	if options.EnableCaching {
		engine.cache = NewDiffCache(1000) // 1000 entry cache
	}

	// Initialize resource index for fast lookups
	if options.EnableIndexing {
		engine.resourceIndex = NewResourceIndex()
	}

	// Initialize streaming channel if enabled
	if options.StreamingEnabled {
		engine.changeStream = make(chan StreamedChange, options.StreamBufferSize)
	}

	return engine
}

// Compare performs an enterprise-grade comparison of two snapshots
func (e *EnterpriseDifferEngine) Compare(baseline, current *types.Snapshot) (*DriftReport, error) {
	return e.CompareWithContext(context.Background(), baseline, current)
}

// CompareWithContext performs comparison with cancellation support
func (e *EnterpriseDifferEngine) CompareWithContext(ctx context.Context, baseline, current *types.Snapshot) (*DriftReport, error) {
	if baseline == nil || current == nil {
		return nil, fmt.Errorf("both baseline and current snapshots are required")
	}

	// Initialize metrics
	e.metrics.StartTime = time.Now()
	e.metrics.TotalResources = int64(len(baseline.Resources) + len(current.Resources))
	atomic.StoreInt64(&e.metrics.ProcessedResources, 0)

	// Report initial progress
	if e.options.ProgressCallback != nil {
		e.options.ProgressCallback(0, "Starting enterprise diff analysis...")
	}

	// Build indexes if enabled
	if e.options.EnableIndexing {
		e.buildResourceIndexes(baseline, current)
	}

	// Determine if we should use parallel processing
	useParallel := len(baseline.Resources)+len(current.Resources) >= e.parallelThreshold

	var matches []ResourceMatch
	var added, removed []types.Resource
	var matchErr error

	// Phase 1: Resource Matching
	if useParallel {
		matches, added, removed, matchErr = e.parallelMatch(ctx, baseline.Resources, current.Resources)
	} else {
		matches, added, removed = e.matcher.Match(baseline.Resources, current.Resources)
	}

	if matchErr != nil {
		return nil, fmt.Errorf("resource matching failed: %w", matchErr)
	}

	// Update metrics
	atomic.StoreInt64(&e.metrics.MatchedResources, int64(len(matches)))
	atomic.StoreInt64(&e.metrics.AddedResources, int64(len(added)))
	atomic.StoreInt64(&e.metrics.RemovedResources, int64(len(removed)))

	// Phase 2: Change Detection
	var resourceChanges []ResourceDiff
	var allChanges []Change
	var mu sync.Mutex

	if e.options.ProgressCallback != nil {
		e.options.ProgressCallback(0.3, fmt.Sprintf("Matched %d resources, detecting changes...", len(matches)))
	}

	// Process changes (parallel or sequential based on workload)
	if useParallel {
		resourceChanges, allChanges = e.parallelProcessChanges(ctx, matches, added, removed)
	} else {
		resourceChanges, allChanges = e.sequentialProcessChanges(ctx, matches, added, removed)
	}

	// Phase 3: Correlation and Risk Assessment
	if e.options.EnableCorrelation && len(resourceChanges) > 0 {
		if e.options.ProgressCallback != nil {
			e.options.ProgressCallback(0.8, "Correlating changes and assessing risk...")
		}
		resourceChanges = e.correlateChanges(resourceChanges)
	}

	// Phase 4: Build Report
	report := e.buildEnterpriseReport(baseline, current, resourceChanges, allChanges)

	// Update final metrics
	e.metrics.EndTime = time.Now()
	e.metrics.ProcessingTimeMs = e.metrics.EndTime.Sub(e.metrics.StartTime).Milliseconds()
	e.metrics.ModifiedResources = int64(len(matches))
	e.metrics.TotalChanges = int64(len(allChanges))

	// Add metrics to report if debug enabled
	if e.options.EnableDebugMetrics {
		report.Metadata["metrics"] = e.metrics
	}

	// Close streaming channel if enabled
	if e.options.StreamingEnabled && e.changeStream != nil {
		close(e.changeStream)
	}

	if e.options.ProgressCallback != nil {
		e.options.ProgressCallback(1.0, "Diff analysis complete")
	}

	return report, nil
}

// parallelMatch performs resource matching in parallel
func (e *EnterpriseDifferEngine) parallelMatch(ctx context.Context, baseline, current []types.Resource) ([]ResourceMatch, []types.Resource, []types.Resource, error) {
	// For large datasets, partition the work
	numWorkers := e.options.MaxWorkers
	if numWorkers > len(baseline)/10 {
		numWorkers = len(baseline)/10 + 1
	}

	// Use smart matching algorithm that partitions by resource type
	// This improves cache locality and matching accuracy
	baselineByType := make(map[string][]types.Resource)
	currentByType := make(map[string][]types.Resource)

	for _, r := range baseline {
		baselineByType[r.Type] = append(baselineByType[r.Type], r)
	}
	for _, r := range current {
		currentByType[r.Type] = append(currentByType[r.Type], r)
	}

	var matches []ResourceMatch
	var added, removed []types.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process each resource type in parallel
	for resourceType := range baselineByType {
		wg.Add(1)
		go func(rType string) {
			defer wg.Done()

			baseResources := baselineByType[rType]
			currResources := currentByType[rType]

			// Use type-specific matcher if available
			typeMatches, typeAdded, typeRemoved := e.matcher.Match(baseResources, currResources)

			mu.Lock()
			matches = append(matches, typeMatches...)
			added = append(added, typeAdded...)
			removed = append(removed, typeRemoved...)
			mu.Unlock()

			// Update progress
			processed := atomic.AddInt64(&e.metrics.ProcessedResources, int64(len(baseResources)+len(currResources)))
			progress := float64(processed) / float64(e.metrics.TotalResources) * 0.3
			if e.options.ProgressCallback != nil {
				e.options.ProgressCallback(progress, fmt.Sprintf("Matching %s resources...", rType))
			}
		}(resourceType)
	}

	// Check for added resource types
	for resourceType, resources := range currentByType {
		if _, exists := baselineByType[resourceType]; !exists {
			mu.Lock()
			added = append(added, resources...)
			mu.Unlock()
		}
	}

	wg.Wait()
	return matches, added, removed, nil
}

// parallelProcessChanges processes resource changes in parallel
func (e *EnterpriseDifferEngine) parallelProcessChanges(ctx context.Context, matches []ResourceMatch, added, removed []types.Resource) ([]ResourceDiff, []Change) {
	var resourceChanges []ResourceDiff
	var allChanges []Change
	var mu sync.Mutex

	// Create job channel
	jobs := make(chan interface{}, len(matches)+len(added)+len(removed))
	results := make(chan processingResult, len(matches)+len(added)+len(removed))

	// Start workers
	var wg sync.WaitGroup
	workerCount := e.options.MaxWorkers
	atomic.StoreInt32(&e.metrics.WorkersUsed, int32(workerCount))

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go e.changeProcessingWorker(ctx, jobs, results, &wg)
	}

	// Queue jobs
	for _, match := range matches {
		jobs <- match
	}
	for _, resource := range added {
		jobs <- addedResource{resource: resource}
	}
	for _, resource := range removed {
		jobs <- removedResource{resource: resource}
	}
	close(jobs)

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	var sequenceID int64
	for result := range results {
		if result.err != nil {
			continue // Log error but continue processing
		}

		mu.Lock()
		if result.resourceDiff != nil {
			resourceChanges = append(resourceChanges, *result.resourceDiff)
		}
		if len(result.changes) > 0 {
			allChanges = append(allChanges, result.changes...)

			// Stream changes if enabled
			if e.options.StreamingEnabled && e.changeStream != nil {
				for _, change := range result.changes {
					select {
					case e.changeStream <- StreamedChange{
						Change:       change,
						ResourceDiff: *result.resourceDiff,
						Timestamp:    time.Now(),
						SequenceID:   atomic.AddInt64(&sequenceID, 1),
					}:
					default:
						// Channel full, skip streaming
					}
				}
			}
		}
		mu.Unlock()

		// Update progress
		processed := atomic.AddInt64(&e.metrics.ProcessedResources, 1)
		progress := 0.3 + (float64(processed)/float64(len(matches)+len(added)+len(removed)))*0.5
		if e.options.ProgressCallback != nil {
			e.options.ProgressCallback(progress, "Processing changes...")
		}
	}

	return resourceChanges, allChanges
}

// Helper types for job processing
type addedResource struct{ resource types.Resource }
type removedResource struct{ resource types.Resource }
type processingResult struct {
	resourceDiff *ResourceDiff
	changes      []Change
	err          error
}

// changeProcessingWorker processes change detection jobs
func (e *EnterpriseDifferEngine) changeProcessingWorker(ctx context.Context, jobs <-chan interface{}, results chan<- processingResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			result := e.processChangeJob(job)
			results <- result
		}
	}
}

// processChangeJob processes a single change detection job
func (e *EnterpriseDifferEngine) processChangeJob(job interface{}) processingResult {
	switch j := job.(type) {
	case ResourceMatch:
		return e.processMatchedResource(j)
	case addedResource:
		return e.processAddedResource(j.resource)
	case removedResource:
		return e.processRemovedResource(j.resource)
	default:
		return processingResult{err: fmt.Errorf("unknown job type: %T", job)}
	}
}

// Continue with remaining implementation...
