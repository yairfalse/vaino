package analyzer

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
)

// ConcurrentCorrelator implements parallelized correlation analysis
type ConcurrentCorrelator struct {
	timeWindow    time.Duration
	workerCount   int
	patternWorkers []PatternWorker
	mutex         sync.RWMutex
}

// PatternWorker represents a worker that processes specific correlation patterns
type PatternWorker struct {
	ID         int
	PatternType string
	Patterns    []PatternMatcher
	ChangesChan chan []differ.SimpleChange
	ResultsChan chan CorrelationResult
	Done        chan struct{}
	closeOnce   sync.Once
	resultOnce  sync.Once
}

// PatternMatcher defines the interface for correlation pattern matching
type PatternMatcher interface {
	Match(changes []differ.SimpleChange) []ChangeGroup
	GetPatternType() string
	GetConfidence() string
}

// CorrelationResult represents the result from a pattern matching worker
type CorrelationResult struct {
	WorkerID    int
	PatternType string
	Groups      []ChangeGroup
	ProcessTime time.Duration
	Error       error
}

// CorrelatedChanges represents the final correlation analysis result
type CorrelatedChanges struct {
	Groups           []ChangeGroup
	ProcessingTime   time.Duration
	WorkerStats      map[string]WorkerStats
	CorrelationStats CorrelationStats
}

// WorkerStats tracks performance metrics for individual workers
type WorkerStats struct {
	WorkerID        int
	PatternType     string
	ProcessingTime  time.Duration
	GroupsFound     int
	ChangesProcessed int
}

// CorrelationStats provides overall correlation metrics
type CorrelationStats struct {
	TotalChanges     int
	TotalGroups      int
	GroupedChanges   int
	UngroupedChanges int
	HighConfidence   int
	MediumConfidence int
	LowConfidence    int
}

// NewConcurrentCorrelator creates a new concurrent correlation engine
func NewConcurrentCorrelator() *ConcurrentCorrelator {
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > 8 {
		workerCount = 8 // Cap at 8 workers to avoid overhead
	}

	cc := &ConcurrentCorrelator{
		timeWindow:  30 * time.Second,
		workerCount: workerCount,
	}

	// Initialize pattern workers
	cc.initializePatternWorkers()
	
	return cc
}

// initializePatternWorkers sets up the pattern matching workers
func (cc *ConcurrentCorrelator) initializePatternWorkers() {
	cc.patternWorkers = make([]PatternWorker, cc.workerCount)
	
	// Pattern types to distribute across workers
	patternTypes := []string{
		"scaling", "config_update", "service_deployment", 
		"network_changes", "storage_changes", "security_changes",
	}
	
	for i := 0; i < cc.workerCount; i++ {
		patternType := patternTypes[i%len(patternTypes)]
		
		cc.patternWorkers[i] = PatternWorker{
			ID:          i,
			PatternType: patternType,
			Patterns:    cc.createPatternMatchers(patternType),
			ChangesChan: make(chan []differ.SimpleChange, 10),
			ResultsChan: make(chan CorrelationResult, 10),
			Done:        make(chan struct{}),
		}
	}
}

// createPatternMatchers creates pattern matchers for a specific type
func (cc *ConcurrentCorrelator) createPatternMatchers(patternType string) []PatternMatcher {
	switch patternType {
	case "scaling":
		return []PatternMatcher{
			&ScalingPatternMatcher{timeWindow: cc.timeWindow},
		}
	case "config_update":
		return []PatternMatcher{
			&ConfigUpdatePatternMatcher{timeWindow: cc.timeWindow},
		}
	case "service_deployment":
		return []PatternMatcher{
			&ServiceDeploymentPatternMatcher{timeWindow: cc.timeWindow},
		}
	case "network_changes":
		return []PatternMatcher{
			&NetworkPatternMatcher{timeWindow: cc.timeWindow},
		}
	case "storage_changes":
		return []PatternMatcher{
			&StoragePatternMatcher{timeWindow: cc.timeWindow},
		}
	case "security_changes":
		return []PatternMatcher{
			&SecurityPatternMatcher{timeWindow: cc.timeWindow},
		}
	default:
		return []PatternMatcher{}
	}
}

// AnalyzeChangesConcurrent processes changes using parallel pattern matching
func (cc *ConcurrentCorrelator) AnalyzeChangesConcurrent(changes []differ.SimpleChange) *CorrelatedChanges {
	if len(changes) == 0 {
		return &CorrelatedChanges{
			Groups: []ChangeGroup{},
			WorkerStats: make(map[string]WorkerStats),
		}
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize fresh workers for each call
	cc.initializePatternWorkers()

	// Start workers
	cc.startWorkers(ctx)
	defer cc.stopWorkers()

	// Group changes by resource type for efficient processing
	groupedChanges := cc.groupResourcesConcurrent(changes)
	
	// Distribute work to workers
	cc.distributeWork(groupedChanges)
	
	// Collect results
	results := cc.collectResults(ctx)
	
	// Merge and score results
	correlatedChanges := cc.mergeAndScoreResults(results, changes)
	correlatedChanges.ProcessingTime = time.Since(startTime)
	
	return correlatedChanges
}

// startWorkers starts all pattern matching workers
func (cc *ConcurrentCorrelator) startWorkers(ctx context.Context) {
	for i := range cc.patternWorkers {
		go cc.runWorker(ctx, &cc.patternWorkers[i])
	}
}

// runWorker runs a single pattern matching worker
func (cc *ConcurrentCorrelator) runWorker(ctx context.Context, worker *PatternWorker) {
	defer func() {
		// Close ResultsChan safely using sync.Once
		worker.resultOnce.Do(func() {
			close(worker.ResultsChan)
		})
	}()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-worker.Done:
			return
		case changes, ok := <-worker.ChangesChan:
			if !ok {
				return
			}
			
			startTime := time.Now()
			var allGroups []ChangeGroup
			
			// Process with all patterns for this worker
			for _, pattern := range worker.Patterns {
				groups := pattern.Match(changes)
				allGroups = append(allGroups, groups...)
			}
			
			// Send result
			result := CorrelationResult{
				WorkerID:    worker.ID,
				PatternType: worker.PatternType,
				Groups:      allGroups,
				ProcessTime: time.Since(startTime),
			}
			
			select {
			case worker.ResultsChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// groupResourcesConcurrent groups resources by type using parallel processing
func (cc *ConcurrentCorrelator) groupResourcesConcurrent(changes []differ.SimpleChange) map[string][]differ.SimpleChange {
	if len(changes) <= 100 {
		// For small datasets, use sequential processing
		return cc.groupResourcesSequential(changes)
	}
	
	// Use concurrent grouping for larger datasets
	resourceGroups := make(map[string][]differ.SimpleChange)
	mutex := sync.Mutex{}
	
	// Group by resource type concurrently
	chunkSize := len(changes) / cc.workerCount
	if chunkSize < 10 {
		chunkSize = 10
	}
	
	var wg sync.WaitGroup
	for i := 0; i < len(changes); i += chunkSize {
		end := i + chunkSize
		if end > len(changes) {
			end = len(changes)
		}
		
		wg.Add(1)
		go func(chunk []differ.SimpleChange) {
			defer wg.Done()
			
			localGroups := make(map[string][]differ.SimpleChange)
			for _, change := range chunk {
				key := fmt.Sprintf("%s:%s", change.ResourceType, change.Namespace)
				localGroups[key] = append(localGroups[key], change)
			}
			
			// Merge into global groups
			mutex.Lock()
			for key, localChanges := range localGroups {
				resourceGroups[key] = append(resourceGroups[key], localChanges...)
			}
			mutex.Unlock()
		}(changes[i:end])
	}
	
	wg.Wait()
	return resourceGroups
}

// groupResourcesSequential groups resources sequentially (fallback)
func (cc *ConcurrentCorrelator) groupResourcesSequential(changes []differ.SimpleChange) map[string][]differ.SimpleChange {
	resourceGroups := make(map[string][]differ.SimpleChange)
	
	for _, change := range changes {
		key := fmt.Sprintf("%s:%s", change.ResourceType, change.Namespace)
		resourceGroups[key] = append(resourceGroups[key], change)
	}
	
	return resourceGroups
}

// distributeWork distributes grouped changes to workers
func (cc *ConcurrentCorrelator) distributeWork(groupedChanges map[string][]differ.SimpleChange) {
	// Convert grouped changes back to flat list for each worker
	allChanges := make([]differ.SimpleChange, 0)
	for _, changes := range groupedChanges {
		allChanges = append(allChanges, changes...)
	}
	
	// Send to each worker
	for i := range cc.patternWorkers {
		select {
		case cc.patternWorkers[i].ChangesChan <- allChanges:
		default:
			// Worker channel is full, skip
		}
	}
}

// collectResults collects results from all workers
func (cc *ConcurrentCorrelator) collectResults(ctx context.Context) []CorrelationResult {
	var results []CorrelationResult
	expectedResults := cc.workerCount
	
	for i := 0; i < expectedResults; i++ {
		select {
		case <-ctx.Done():
			return results
		case result, ok := <-cc.patternWorkers[i].ResultsChan:
			if ok {
				results = append(results, result)
			}
		case <-time.After(5 * time.Second):
			// Timeout waiting for worker result
			continue
		}
	}
	
	return results
}

// mergeAndScoreResults merges results from all workers and scores confidence
func (cc *ConcurrentCorrelator) mergeAndScoreResults(results []CorrelationResult, originalChanges []differ.SimpleChange) *CorrelatedChanges {
	var allGroups []ChangeGroup
	workerStats := make(map[string]WorkerStats)
	
	// Track which changes have been grouped
	used := make(map[string]bool)
	
	// Process results from each worker
	for _, result := range results {
		workerStats[result.PatternType] = WorkerStats{
			WorkerID:        result.WorkerID,
			PatternType:     result.PatternType,
			ProcessingTime:  result.ProcessTime,
			GroupsFound:     len(result.Groups),
			ChangesProcessed: len(originalChanges),
		}
		
		// Add groups, avoiding duplicates
		for _, group := range result.Groups {
			// Check if any changes in this group are already used
			hasUsedChange := false
			for _, change := range group.Changes {
				if used[change.ResourceID] {
					hasUsedChange = true
					break
				}
			}
			
			if !hasUsedChange {
				// Mark all changes in this group as used
				for _, change := range group.Changes {
					used[change.ResourceID] = true
				}
				allGroups = append(allGroups, group)
			}
		}
	}
	
	// Handle ungrouped changes
	var ungrouped []differ.SimpleChange
	for _, change := range originalChanges {
		if !used[change.ResourceID] {
			ungrouped = append(ungrouped, change)
		}
	}
	
	if len(ungrouped) > 0 {
		allGroups = append(allGroups, ChangeGroup{
			Timestamp:   ungrouped[0].Timestamp,
			Title:       "Other Changes",
			Description: "Individual resource changes",
			Changes:     ungrouped,
			Confidence:  "low",
		})
	}
	
	// Calculate confidence scores in parallel
	cc.calculateConfidenceConcurrent(allGroups)
	
	// Generate correlation statistics
	stats := cc.generateCorrelationStats(allGroups, originalChanges)
	
	return &CorrelatedChanges{
		Groups:           allGroups,
		WorkerStats:      workerStats,
		CorrelationStats: stats,
	}
}

// calculateConfidenceConcurrent calculates confidence scores using parallel processing
func (cc *ConcurrentCorrelator) calculateConfidenceConcurrent(groups []ChangeGroup) {
	if len(groups) <= 10 {
		// For small groups, use sequential processing
		cc.calculateConfidenceSequential(groups)
		return
	}
	
	// Use parallel processing for confidence calculation
	chunkSize := len(groups) / cc.workerCount
	if chunkSize < 1 {
		chunkSize = 1
	}
	
	var wg sync.WaitGroup
	for i := 0; i < len(groups); i += chunkSize {
		end := i + chunkSize
		if end > len(groups) {
			end = len(groups)
		}
		
		wg.Add(1)
		go func(chunk []ChangeGroup, startIdx int) {
			defer wg.Done()
			
			for j, group := range chunk {
				confidence := cc.calculateGroupConfidence(group)
				groups[startIdx+j].Confidence = confidence
			}
		}(groups[i:end], i)
	}
	
	wg.Wait()
}

// calculateConfidenceSequential calculates confidence scores sequentially
func (cc *ConcurrentCorrelator) calculateConfidenceSequential(groups []ChangeGroup) {
	for i := range groups {
		groups[i].Confidence = cc.calculateGroupConfidence(groups[i])
	}
}

// calculateGroupConfidence calculates confidence score for a single group
func (cc *ConcurrentCorrelator) calculateGroupConfidence(group ChangeGroup) string {
	score := 0
	
	// Time window factor
	if len(group.Changes) > 1 {
		maxTime := group.Changes[0].Timestamp
		minTime := group.Changes[0].Timestamp
		
		for _, change := range group.Changes[1:] {
			if change.Timestamp.After(maxTime) {
				maxTime = change.Timestamp
			}
			if change.Timestamp.Before(minTime) {
				minTime = change.Timestamp
			}
		}
		
		timeDiff := maxTime.Sub(minTime)
		if timeDiff <= 30*time.Second {
			score += 3
		} else if timeDiff <= 2*time.Minute {
			score += 2
		} else if timeDiff <= 5*time.Minute {
			score += 1
		}
	}
	
	// Resource relationship factor
	if len(group.Changes) > 1 {
		sameNamespace := true
		namespace := group.Changes[0].Namespace
		for _, change := range group.Changes[1:] {
			if change.Namespace != namespace {
				sameNamespace = false
				break
			}
		}
		
		if sameNamespace {
			score += 2
		}
	}
	
	// Pattern strength factor
	if strings.Contains(group.Title, "Scaling") {
		score += 3
	} else if strings.Contains(group.Title, "Config Update") {
		score += 2
	} else if strings.Contains(group.Title, "Service") {
		score += 2
	}
	
	// Group size factor
	if len(group.Changes) >= 5 {
		score += 2
	} else if len(group.Changes) >= 3 {
		score += 1
	}
	
	// Convert score to confidence level
	if score >= 7 {
		return "high"
	} else if score >= 4 {
		return "medium"
	} else {
		return "low"
	}
}

// generateCorrelationStats generates correlation statistics
func (cc *ConcurrentCorrelator) generateCorrelationStats(groups []ChangeGroup, originalChanges []differ.SimpleChange) CorrelationStats {
	stats := CorrelationStats{
		TotalChanges: len(originalChanges),
		TotalGroups:  len(groups),
	}
	
	for _, group := range groups {
		stats.GroupedChanges += len(group.Changes)
		
		switch group.Confidence {
		case "high":
			stats.HighConfidence++
		case "medium":
			stats.MediumConfidence++
		case "low":
			stats.LowConfidence++
		}
	}
	
	stats.UngroupedChanges = stats.TotalChanges - stats.GroupedChanges
	
	return stats
}

// stopWorkers stops all pattern matching workers
func (cc *ConcurrentCorrelator) stopWorkers() {
	for i := range cc.patternWorkers {
		worker := &cc.patternWorkers[i]
		worker.closeOnce.Do(func() {
			close(worker.Done)
			close(worker.ChangesChan)
		})
	}
}

// GetWorkerCount returns the number of workers
func (cc *ConcurrentCorrelator) GetWorkerCount() int {
	return cc.workerCount
}

// SetWorkerCount sets the number of workers (for testing)
func (cc *ConcurrentCorrelator) SetWorkerCount(count int) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	
	if count < 1 {
		count = 1
	}
	if count > 16 {
		count = 16
	}
	
	cc.workerCount = count
	cc.initializePatternWorkers()
}