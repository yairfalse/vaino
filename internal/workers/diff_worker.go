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

// ComparisonPair represents a pair of resources to compare
type ComparisonPair struct {
	BaselineResource *types.Resource
	CurrentResource  *types.Resource
	ResourceID       string
	Priority         int // Higher priority pairs are processed first
}

// DiffResult represents the result of comparing two resources
type DiffResult struct {
	ResourceID  string
	Changes     []types.Change
	DriftType   types.DriftType
	Severity    types.DriftSeverity
	RiskScore   float64
	CompareTime time.Duration
	WorkerID    int
	Error       error
}

// DiffJob represents a comparison job
type DiffJob struct {
	Pair       ComparisonPair
	Options    DiffOptions
	ResultChan chan<- DiffResult
}

// DiffOptions configures comparison behavior
type DiffOptions struct {
	DeepComparison bool                // Enable deep comparison
	IgnoreMetadata bool                // Ignore metadata changes
	IgnorePatterns []string            // Patterns to ignore
	CompareTimeout time.Duration       // Timeout for comparison
	ValidationMode bool                // Enable validation
	SeverityFilter types.DriftSeverity // Minimum severity to report
}

// DiffWorker represents a single comparison worker
type DiffWorker struct {
	workerCount int
	jobChan     chan DiffJob
	workers     []*diffWorker

	// Configuration
	bufferSize     int
	compareTimeout time.Duration
	batchSize      int

	// State management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	totalCompared int64
	totalChanges  int64
	totalErrors   int64
	totalTime     time.Duration
	mu            sync.RWMutex

	// Performance optimization
	comparisonCache *ComparisonCache
	fieldComparer   *FieldComparer
}

// diffWorker represents a single diff worker goroutine
type diffWorker struct {
	id       int
	parent   *DiffWorker
	stats    diffWorkerStats
	comparer *ResourceComparer
}

// diffWorkerStats holds statistics for a diff worker
type diffWorkerStats struct {
	compared     int64
	changesFound int64
	errors       int64
	totalTime    time.Duration
	lastActive   time.Time
	mu           sync.RWMutex
}

// ResourceComparer handles the actual comparison logic
type ResourceComparer struct {
	fieldComparer *FieldComparer
	validator     *ResourceValidator
}

// FieldComparer provides optimized field comparison
type FieldComparer struct {
	ignorePatterns []string
	cache          map[string]bool
	mu             sync.RWMutex
}

// ComparisonCache caches comparison results
type ComparisonCache struct {
	cache map[string]DiffResult
	mu    sync.RWMutex
	ttl   time.Duration
}

// ResourceValidator validates resources during comparison
type ResourceValidator struct {
	validationCache map[string]bool
	mu              sync.RWMutex
}

// NewDiffWorker creates a new diff worker pool
func NewDiffWorker(opts ...DiffWorkerOption) *DiffWorker {
	dw := &DiffWorker{
		workerCount:     runtime.NumCPU(),
		bufferSize:      100,
		compareTimeout:  30 * time.Second,
		batchSize:       10,
		comparisonCache: NewComparisonCache(5 * time.Minute),
		fieldComparer:   NewFieldComparer(),
	}

	// Apply options
	for _, opt := range opts {
		opt(dw)
	}

	// Initialize channels
	dw.jobChan = make(chan DiffJob, dw.bufferSize)

	// Create context
	dw.ctx, dw.cancel = context.WithCancel(context.Background())

	// Initialize workers
	dw.workers = make([]*diffWorker, dw.workerCount)
	for i := 0; i < dw.workerCount; i++ {
		dw.workers[i] = &diffWorker{
			id:       i,
			parent:   dw,
			comparer: NewResourceComparer(dw.fieldComparer),
		}
	}

	return dw
}

// DiffWorkerOption configures the diff worker
type DiffWorkerOption func(*DiffWorker)

// WithDiffWorkerCount sets the number of worker goroutines
func WithDiffWorkerCount(count int) DiffWorkerOption {
	return func(dw *DiffWorker) {
		if count > 0 {
			dw.workerCount = count
		}
	}
}

// WithDiffBufferSize sets the buffer size for job channels
func WithDiffBufferSize(size int) DiffWorkerOption {
	return func(dw *DiffWorker) {
		if size > 0 {
			dw.bufferSize = size
		}
	}
}

// WithDiffTimeout sets the comparison timeout
func WithDiffTimeout(timeout time.Duration) DiffWorkerOption {
	return func(dw *DiffWorker) {
		dw.compareTimeout = timeout
	}
}

// WithDiffBatchSize sets the batch size for processing
func WithDiffBatchSize(size int) DiffWorkerOption {
	return func(dw *DiffWorker) {
		if size > 0 {
			dw.batchSize = size
		}
	}
}

// WithComparisonCache enables comparison caching
func WithComparisonCache(ttl time.Duration) DiffWorkerOption {
	return func(dw *DiffWorker) {
		dw.comparisonCache = NewComparisonCache(ttl)
	}
}

// ComputeDiffsConcurrent computes diffs between two snapshots concurrently
func (dw *DiffWorker) ComputeDiffsConcurrent(baseline, current *types.Snapshot) (*types.DriftReport, error) {
	if baseline == nil || current == nil {
		return nil, fmt.Errorf("baseline and current snapshots are required")
	}

	// Start workers
	if err := dw.start(); err != nil {
		return nil, fmt.Errorf("failed to start diff workers: %w", err)
	}
	defer dw.stop()

	// Create resource maps for efficient lookup
	baselineMap := make(map[string]*types.Resource)
	for i := range baseline.Resources {
		baselineMap[baseline.Resources[i].ID] = &baseline.Resources[i]
	}

	currentMap := make(map[string]*types.Resource)
	for i := range current.Resources {
		currentMap[current.Resources[i].ID] = &current.Resources[i]
	}

	// Create comparison pairs
	pairs := dw.createComparisonPairs(baselineMap, currentMap)

	// Process pairs in batches
	results := make([]DiffResult, 0, len(pairs))
	resultChan := make(chan DiffResult, len(pairs))

	// Submit comparison jobs
	for _, pair := range pairs {
		job := DiffJob{
			Pair: pair,
			Options: DiffOptions{
				DeepComparison: true,
				IgnoreMetadata: false,
				CompareTimeout: dw.compareTimeout,
				ValidationMode: true,
				SeverityFilter: types.DriftSeverityLow,
			},
			ResultChan: resultChan,
		}

		select {
		case dw.jobChan <- job:
		case <-dw.ctx.Done():
			return nil, fmt.Errorf("diff context cancelled")
		}
	}

	// Collect results
	for i := 0; i < len(pairs); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
		case <-time.After(dw.compareTimeout * 2):
			return nil, fmt.Errorf("timeout waiting for diff results")
		}
	}

	// Create drift report
	report := dw.createDriftReport(baseline, current, results)

	return report, nil
}

// start initializes and starts the diff workers
func (dw *DiffWorker) start() error {
	// Start workers
	for i := 0; i < dw.workerCount; i++ {
		dw.wg.Add(1)
		go dw.worker(i)
	}

	return nil
}

// stop gracefully shuts down the diff workers
func (dw *DiffWorker) stop() {
	// Cancel context
	dw.cancel()

	// Close job channel
	close(dw.jobChan)

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		dw.wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Normal shutdown
	case <-time.After(10 * time.Second):
		// Force shutdown
		fmt.Println("Warning: Diff worker shutdown timed out")
	}
}

// worker processes comparison jobs
func (dw *DiffWorker) worker(workerID int) {
	defer dw.wg.Done()

	worker := dw.workers[workerID]

	for {
		select {
		case <-dw.ctx.Done():
			return
		case job, ok := <-dw.jobChan:
			if !ok {
				return
			}

			// Process the job
			result := dw.processComparisonJob(job, workerID)

			// Update worker stats
			worker.updateStats(result)

			// Send result
			select {
			case job.ResultChan <- result:
			case <-dw.ctx.Done():
				return
			}
		}
	}
}

// processComparisonJob processes a single comparison job
func (dw *DiffWorker) processComparisonJob(job DiffJob, workerID int) DiffResult {
	startTime := time.Now()

	result := DiffResult{
		ResourceID:  job.Pair.ResourceID,
		WorkerID:    workerID,
		CompareTime: 0,
	}

	// Check cache first
	if dw.comparisonCache != nil {
		if cached, found := dw.comparisonCache.Get(job.Pair.ResourceID); found {
			cached.WorkerID = workerID
			cached.CompareTime = time.Since(startTime)
			return cached
		}
	}

	// Perform comparison with timeout
	ctx, cancel := context.WithTimeout(dw.ctx, job.Options.CompareTimeout)
	defer cancel()

	changes, driftType, severity, riskScore, err := dw.compareResources(ctx, job.Pair, job.Options)

	result.CompareTime = time.Since(startTime)
	result.Changes = changes
	result.DriftType = driftType
	result.Severity = severity
	result.RiskScore = riskScore
	result.Error = err

	// Cache result
	if dw.comparisonCache != nil && err == nil {
		dw.comparisonCache.Set(job.Pair.ResourceID, result)
	}

	// Update metrics
	atomic.AddInt64(&dw.totalCompared, 1)
	if err != nil {
		atomic.AddInt64(&dw.totalErrors, 1)
	} else {
		atomic.AddInt64(&dw.totalChanges, int64(len(changes)))
	}

	return result
}

// compareResources performs the actual resource comparison
func (dw *DiffWorker) compareResources(ctx context.Context, pair ComparisonPair, options DiffOptions) ([]types.Change, types.DriftType, types.DriftSeverity, float64, error) {
	baseline := pair.BaselineResource
	current := pair.CurrentResource

	// Determine drift type
	var driftType types.DriftType
	if baseline == nil && current != nil {
		driftType = types.DriftTypeCreated
	} else if baseline != nil && current == nil {
		driftType = types.DriftTypeDeleted
	} else if baseline != nil && current != nil {
		driftType = types.DriftTypeModified
	} else {
		return nil, "", "", 0, fmt.Errorf("invalid resource pair")
	}

	// Handle created/deleted resources
	if driftType == types.DriftTypeCreated || driftType == types.DriftTypeDeleted {
		var resource *types.Resource
		if current != nil {
			resource = current
		} else {
			resource = baseline
		}

		changes := []types.Change{
			{
				Field:       "resource",
				OldValue:    baseline,
				NewValue:    current,
				Severity:    string(types.DriftSeverityHigh),
				Path:        resource.ID,
				Description: fmt.Sprintf("Resource %s", driftType),
			},
		}

		return changes, driftType, types.DriftSeverityHigh, 0.8, nil
	}

	// Compare modified resources
	worker := dw.workers[0] // Use first worker's comparer
	changes, err := worker.comparer.Compare(ctx, baseline, current, options)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("comparison failed: %w", err)
	}

	// Calculate severity and risk score
	severity := dw.calculateSeverity(changes)
	riskScore := dw.calculateRiskScore(changes, baseline, current)

	return changes, driftType, severity, riskScore, nil
}

// createComparisonPairs creates pairs of resources to compare
func (dw *DiffWorker) createComparisonPairs(baselineMap, currentMap map[string]*types.Resource) []ComparisonPair {
	allIDs := make(map[string]bool)

	// Collect all resource IDs
	for id := range baselineMap {
		allIDs[id] = true
	}
	for id := range currentMap {
		allIDs[id] = true
	}

	// Create pairs
	pairs := make([]ComparisonPair, 0, len(allIDs))
	for id := range allIDs {
		baseline := baselineMap[id]
		current := currentMap[id]

		priority := 100
		if baseline != nil && current != nil {
			priority = 50 // Modified resources have lower priority
		}

		pairs = append(pairs, ComparisonPair{
			BaselineResource: baseline,
			CurrentResource:  current,
			ResourceID:       id,
			Priority:         priority,
		})
	}

	// Sort by priority (higher first)
	dw.sortPairsByPriority(pairs)

	return pairs
}

// sortPairsByPriority sorts comparison pairs by priority
func (dw *DiffWorker) sortPairsByPriority(pairs []ComparisonPair) {
	// Simple bubble sort - in production, use sort.Slice
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].Priority < pairs[j].Priority {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
}

// calculateSeverity calculates the severity of changes
func (dw *DiffWorker) calculateSeverity(changes []types.Change) types.DriftSeverity {
	if len(changes) == 0 {
		return types.DriftSeverityLow
	}

	highCount := 0
	for _, change := range changes {
		if change.Severity == string(types.DriftSeverityHigh) || change.Severity == string(types.DriftSeverityCritical) {
			highCount++
		}
	}

	if highCount > len(changes)/2 {
		return types.DriftSeverityHigh
	} else if highCount > 0 {
		return types.DriftSeverityMedium
	}

	return types.DriftSeverityLow
}

// calculateRiskScore calculates the risk score of changes
func (dw *DiffWorker) calculateRiskScore(changes []types.Change, baseline, current *types.Resource) float64 {
	if len(changes) == 0 {
		return 0.0
	}

	score := 0.0
	for _, change := range changes {
		switch change.Severity {
		case string(types.DriftSeverityCritical):
			score += 1.0
		case string(types.DriftSeverityHigh):
			score += 0.7
		case string(types.DriftSeverityMedium):
			score += 0.4
		case string(types.DriftSeverityLow):
			score += 0.1
		}
	}

	// Normalize score
	maxScore := float64(len(changes))
	if maxScore > 0 {
		score = score / maxScore
	}

	return score
}

// createDriftReport creates a drift report from comparison results
func (dw *DiffWorker) createDriftReport(baseline, current *types.Snapshot, results []DiffResult) *types.DriftReport {
	report := &types.DriftReport{
		ID:         fmt.Sprintf("drift-%d", time.Now().Unix()),
		Timestamp:  time.Now(),
		BaselineID: baseline.ID,
		CurrentID:  current.ID,
		Changes:    make([]types.Change, 0),
		Summary: types.DriftSummary{
			TotalChanges:      0,
			AddedResources:    0,
			DeletedResources:  0,
			ModifiedResources: 0,
			RiskScore:         0.0,
			HighRiskChanges:   0,
		},
	}

	// Aggregate results
	totalRiskScore := 0.0
	for _, result := range results {
		if result.Error != nil {
			continue
		}

		report.Changes = append(report.Changes, result.Changes...)
		report.Summary.TotalChanges += len(result.Changes)

		switch result.DriftType {
		case types.DriftTypeCreated:
			report.Summary.AddedResources++
		case types.DriftTypeDeleted:
			report.Summary.DeletedResources++
		case types.DriftTypeModified:
			if len(result.Changes) > 0 {
				report.Summary.ModifiedResources++
			}
		}

		totalRiskScore += result.RiskScore

		if result.Severity == types.DriftSeverityHigh || result.Severity == types.DriftSeverityCritical {
			report.Summary.HighRiskChanges++
		}
	}

	// Calculate overall risk score
	if len(results) > 0 {
		report.Summary.RiskScore = totalRiskScore / float64(len(results))
	}

	return report
}

// updateStats updates worker statistics
func (w *diffWorker) updateStats(result DiffResult) {
	w.stats.mu.Lock()
	defer w.stats.mu.Unlock()

	w.stats.lastActive = time.Now()
	w.stats.totalTime += result.CompareTime
	w.stats.compared++

	if result.Error != nil {
		w.stats.errors++
	} else {
		w.stats.changesFound += int64(len(result.Changes))
	}
}

// GetStats returns diff worker statistics
func (dw *DiffWorker) GetStats() DiffWorkerStats {
	dw.mu.RLock()
	defer dw.mu.RUnlock()

	stats := DiffWorkerStats{
		TotalCompared: atomic.LoadInt64(&dw.totalCompared),
		TotalChanges:  atomic.LoadInt64(&dw.totalChanges),
		TotalErrors:   atomic.LoadInt64(&dw.totalErrors),
		WorkerCount:   dw.workerCount,
		WorkerStats:   make([]DiffWorkerStatDetail, len(dw.workers)),
	}

	for i, worker := range dw.workers {
		worker.stats.mu.RLock()
		stats.WorkerStats[i] = DiffWorkerStatDetail{
			WorkerID:     i,
			Compared:     worker.stats.compared,
			ChangesFound: worker.stats.changesFound,
			Errors:       worker.stats.errors,
			TotalTime:    worker.stats.totalTime,
			LastActive:   worker.stats.lastActive,
		}
		worker.stats.mu.RUnlock()
	}

	return stats
}

// DiffWorkerStats holds diff worker statistics
type DiffWorkerStats struct {
	TotalCompared int64
	TotalChanges  int64
	TotalErrors   int64
	WorkerCount   int
	WorkerStats   []DiffWorkerStatDetail
}

// DiffWorkerStatDetail holds individual worker statistics
type DiffWorkerStatDetail struct {
	WorkerID     int
	Compared     int64
	ChangesFound int64
	Errors       int64
	TotalTime    time.Duration
	LastActive   time.Time
}

// NewResourceComparer creates a new resource comparer
func NewResourceComparer(fieldComparer *FieldComparer) *ResourceComparer {
	return &ResourceComparer{
		fieldComparer: fieldComparer,
		validator: &ResourceValidator{
			validationCache: make(map[string]bool),
		},
	}
}

// Compare compares two resources and returns changes
func (rc *ResourceComparer) Compare(ctx context.Context, baseline, current *types.Resource, options DiffOptions) ([]types.Change, error) {
	var changes []types.Change

	// Compare basic fields
	if baseline.Name != current.Name {
		changes = append(changes, types.Change{
			Field:       "name",
			OldValue:    baseline.Name,
			NewValue:    current.Name,
			Severity:    string(types.DriftSeverityMedium),
			Path:        "name",
			Description: "Resource name changed",
		})
	}

	if baseline.Type != current.Type {
		changes = append(changes, types.Change{
			Field:       "type",
			OldValue:    baseline.Type,
			NewValue:    current.Type,
			Severity:    string(types.DriftSeverityHigh),
			Path:        "type",
			Description: "Resource type changed",
		})
	}

	// Compare configuration
	configChanges := rc.fieldComparer.CompareConfiguration(baseline.Configuration, current.Configuration)
	changes = append(changes, configChanges...)

	// Compare tags if not ignoring metadata
	if !options.IgnoreMetadata {
		tagChanges := rc.fieldComparer.CompareTags(baseline.Tags, current.Tags)
		changes = append(changes, tagChanges...)
	}

	return changes, nil
}

// NewFieldComparer creates a new field comparer
func NewFieldComparer() *FieldComparer {
	return &FieldComparer{
		cache: make(map[string]bool),
	}
}

// CompareConfiguration compares resource configurations
func (fc *FieldComparer) CompareConfiguration(baseline, current map[string]interface{}) []types.Change {
	var changes []types.Change

	// Find all keys
	allKeys := make(map[string]bool)
	for k := range baseline {
		allKeys[k] = true
	}
	for k := range current {
		allKeys[k] = true
	}

	// Compare each key
	for key := range allKeys {
		oldVal, oldExists := baseline[key]
		newVal, newExists := current[key]

		if !oldExists && newExists {
			changes = append(changes, types.Change{
				Field:       key,
				OldValue:    nil,
				NewValue:    newVal,
				Severity:    string(types.DriftSeverityLow),
				Path:        fmt.Sprintf("configuration.%s", key),
				Description: fmt.Sprintf("Configuration key %s added", key),
			})
		} else if oldExists && !newExists {
			changes = append(changes, types.Change{
				Field:       key,
				OldValue:    oldVal,
				NewValue:    nil,
				Severity:    string(types.DriftSeverityMedium),
				Path:        fmt.Sprintf("configuration.%s", key),
				Description: fmt.Sprintf("Configuration key %s removed", key),
			})
		} else if oldExists && newExists && !fc.valuesEqual(oldVal, newVal) {
			changes = append(changes, types.Change{
				Field:       key,
				OldValue:    oldVal,
				NewValue:    newVal,
				Severity:    string(types.DriftSeverityMedium),
				Path:        fmt.Sprintf("configuration.%s", key),
				Description: fmt.Sprintf("Configuration key %s changed", key),
			})
		}
	}

	return changes
}

// CompareTags compares resource tags
func (fc *FieldComparer) CompareTags(baseline, current map[string]string) []types.Change {
	var changes []types.Change

	// Find all keys
	allKeys := make(map[string]bool)
	for k := range baseline {
		allKeys[k] = true
	}
	for k := range current {
		allKeys[k] = true
	}

	// Compare each key
	for key := range allKeys {
		oldVal, oldExists := baseline[key]
		newVal, newExists := current[key]

		if !oldExists && newExists {
			changes = append(changes, types.Change{
				Field:       key,
				OldValue:    "",
				NewValue:    newVal,
				Severity:    string(types.DriftSeverityLow),
				Path:        fmt.Sprintf("tags.%s", key),
				Description: fmt.Sprintf("Tag %s added", key),
			})
		} else if oldExists && !newExists {
			changes = append(changes, types.Change{
				Field:       key,
				OldValue:    oldVal,
				NewValue:    "",
				Severity:    string(types.DriftSeverityLow),
				Path:        fmt.Sprintf("tags.%s", key),
				Description: fmt.Sprintf("Tag %s removed", key),
			})
		} else if oldExists && newExists && oldVal != newVal {
			changes = append(changes, types.Change{
				Field:       key,
				OldValue:    oldVal,
				NewValue:    newVal,
				Severity:    string(types.DriftSeverityLow),
				Path:        fmt.Sprintf("tags.%s", key),
				Description: fmt.Sprintf("Tag %s changed", key),
			})
		}
	}

	return changes
}

// valuesEqual compares two interface{} values
func (fc *FieldComparer) valuesEqual(a, b interface{}) bool {
	// Simple equality check - in production, use deep comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// NewComparisonCache creates a new comparison cache
func NewComparisonCache(ttl time.Duration) *ComparisonCache {
	return &ComparisonCache{
		cache: make(map[string]DiffResult),
		ttl:   ttl,
	}
}

// Get retrieves a cached result
func (cc *ComparisonCache) Get(key string) (DiffResult, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result, exists := cc.cache[key]
	return result, exists
}

// Set stores a result in the cache
func (cc *ComparisonCache) Set(key string, result DiffResult) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.cache[key] = result
}
