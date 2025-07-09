package watchers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// IncrementalScanner implements smart change detection with incremental scanning
type IncrementalScanner struct {
	mu                sync.RWMutex
	provider          string
	collector         collectors.EnhancedCollector
	config            collectors.CollectorConfig
	resourceIndex     map[string]ResourceSnapshot
	hashIndex         map[string]string
	changeDetector    *ChangeDetector
	scanHistory       []ScanResult
	maxHistorySize    int
	scanningStrategy  ScanningStrategy
	enabled           bool
	stats             IncrementalScannerStats
}

// ResourceSnapshot represents a snapshot of a resource at a point in time
type ResourceSnapshot struct {
	Resource      types.Resource `json:"resource"`
	Hash          string         `json:"hash"`
	Timestamp     time.Time      `json:"timestamp"`
	ScanID        string         `json:"scan_id"`
	ChangeCount   int            `json:"change_count"`
	LastModified  time.Time      `json:"last_modified"`
	Dependencies  []string       `json:"dependencies"`
	ChecksumValid bool           `json:"checksum_valid"`
}

// ScanResult represents the result of an incremental scan
type ScanResult struct {
	ScanID        string                 `json:"scan_id"`
	Timestamp     time.Time              `json:"timestamp"`
	Duration      time.Duration          `json:"duration"`
	ResourceCount int                    `json:"resource_count"`
	ChangesFound  int                    `json:"changes_found"`
	NewResources  int                    `json:"new_resources"`
	DeletedResources int                 `json:"deleted_resources"`
	ModifiedResources int                `json:"modified_resources"`
	ErrorCount    int                    `json:"error_count"`
	ScanType      ScanType               `json:"scan_type"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ScanType represents the type of scan performed
type ScanType string

const (
	ScanTypeFull        ScanType = "full"
	ScanTypeIncremental ScanType = "incremental"
	ScanTypeTargeted    ScanType = "targeted"
	ScanTypeDelta       ScanType = "delta"
)

// ScanningStrategy defines the strategy for incremental scanning
type ScanningStrategy struct {
	DefaultScanType    ScanType      `json:"default_scan_type"`
	FullScanInterval   time.Duration `json:"full_scan_interval"`
	IncrementalInterval time.Duration `json:"incremental_interval"`
	TargetedThreshold  int           `json:"targeted_threshold"`
	DeltaChangeRatio   float64       `json:"delta_change_ratio"`
	EnablePredictive   bool          `json:"enable_predictive"`
	EnableOptimization bool          `json:"enable_optimization"`
}

// IncrementalScannerStats holds statistics for the incremental scanner
type IncrementalScannerStats struct {
	TotalScans           int64         `json:"total_scans"`
	FullScans            int64         `json:"full_scans"`
	IncrementalScans     int64         `json:"incremental_scans"`
	TargetedScans        int64         `json:"targeted_scans"`
	DeltaScans           int64         `json:"delta_scans"`
	AverageScanTime      time.Duration `json:"average_scan_time"`
	TotalChangesDetected int64         `json:"total_changes_detected"`
	ChangeDetectionRate  float64       `json:"change_detection_rate"`
	ScanEfficiency       float64       `json:"scan_efficiency"`
	LastFullScan         time.Time     `json:"last_full_scan"`
	LastIncrementalScan  time.Time     `json:"last_incremental_scan"`
	ResourceCacheHitRate float64       `json:"resource_cache_hit_rate"`
	ErrorRate            float64       `json:"error_rate"`
}

// ChangeDetector implements smart change detection algorithms
type ChangeDetector struct {
	mu               sync.RWMutex
	changePatterns   map[string]ChangePattern
	predictionModel  *PredictionModel
	optimizationRules []OptimizationRule
	enabled          bool
}

// ChangePattern represents a pattern of changes
type ChangePattern struct {
	ID          string            `json:"id"`
	Provider    string            `json:"provider"`
	ResourceType string           `json:"resource_type"`
	ChangeType  types.ChangeType  `json:"change_type"`
	Frequency   time.Duration     `json:"frequency"`
	Confidence  float64           `json:"confidence"`
	LastSeen    time.Time         `json:"last_seen"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PredictionModel predicts when resources are likely to change
type PredictionModel struct {
	mu                sync.RWMutex
	resourcePredictions map[string]ResourcePrediction
	enabled           bool
}

// ResourcePrediction represents a prediction for a resource
type ResourcePrediction struct {
	ResourceID       string    `json:"resource_id"`
	NextChangeTime   time.Time `json:"next_change_time"`
	ChangeProbability float64   `json:"change_probability"`
	ChangeType       types.ChangeType `json:"change_type"`
	Confidence       float64   `json:"confidence"`
	LastUpdated      time.Time `json:"last_updated"`
}

// OptimizationRule defines rules for scan optimization
type OptimizationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Conditions  []OptimizationCondition `json:"conditions"`
	Actions     []OptimizationAction   `json:"actions"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
}

// OptimizationCondition defines conditions for optimization
type OptimizationCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// OptimizationAction defines actions for optimization
type OptimizationAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// NewIncrementalScanner creates a new incremental scanner
func NewIncrementalScanner(provider string, collector collectors.EnhancedCollector, config collectors.CollectorConfig) *IncrementalScanner {
	return &IncrementalScanner{
		provider:       provider,
		collector:      collector,
		config:         config,
		resourceIndex:  make(map[string]ResourceSnapshot),
		hashIndex:      make(map[string]string),
		changeDetector: NewChangeDetector(),
		scanHistory:    []ScanResult{},
		maxHistorySize: 100,
		scanningStrategy: ScanningStrategy{
			DefaultScanType:     ScanTypeIncremental,
			FullScanInterval:    30 * time.Minute,
			IncrementalInterval: 5 * time.Minute,
			TargetedThreshold:   10,
			DeltaChangeRatio:    0.1,
			EnablePredictive:    true,
			EnableOptimization:  true,
		},
		enabled: true,
		stats:   IncrementalScannerStats{},
	}
}

// NewChangeDetector creates a new change detector
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{
		changePatterns:    make(map[string]ChangePattern),
		predictionModel:   NewPredictionModel(),
		optimizationRules: []OptimizationRule{},
		enabled:           true,
	}
}

// NewPredictionModel creates a new prediction model
func NewPredictionModel() *PredictionModel {
	return &PredictionModel{
		resourcePredictions: make(map[string]ResourcePrediction),
		enabled:             true,
	}
}

// PerformScan performs an incremental scan
func (is *IncrementalScanner) PerformScan(ctx context.Context) (*ScanResult, error) {
	is.mu.Lock()
	defer is.mu.Unlock()
	
	if !is.enabled {
		return nil, fmt.Errorf("incremental scanner is disabled")
	}
	
	startTime := time.Now()
	scanID := fmt.Sprintf("%s-%d", is.provider, startTime.UnixNano())
	
	// Determine scan type
	scanType := is.determineScanType()
	
	// Perform scan based on type
	var result *ScanResult
	var err error
	
	switch scanType {
	case ScanTypeFull:
		result, err = is.performFullScan(ctx, scanID)
	case ScanTypeIncremental:
		result, err = is.performIncrementalScan(ctx, scanID)
	case ScanTypeTargeted:
		result, err = is.performTargetedScan(ctx, scanID)
	case ScanTypeDelta:
		result, err = is.performDeltaScan(ctx, scanID)
	default:
		result, err = is.performIncrementalScan(ctx, scanID)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to perform %s scan: %w", scanType, err)
	}
	
	// Update statistics
	is.updateStats(result)
	
	// Add to history
	is.addToHistory(*result)
	
	return result, nil
}

// determineScanType determines the type of scan to perform
func (is *IncrementalScanner) determineScanType() ScanType {
	now := time.Now()
	
	// Check if we need a full scan
	if is.stats.LastFullScan.IsZero() || now.Sub(is.stats.LastFullScan) > is.scanningStrategy.FullScanInterval {
		return ScanTypeFull
	}
	
	// Check if we should do targeted scan
	if is.scanningStrategy.EnableOptimization && is.shouldDoTargetedScan() {
		return ScanTypeTargeted
	}
	
	// Check if we should do delta scan
	if is.scanningStrategy.EnableOptimization && is.shouldDoDeltaScan() {
		return ScanTypeDelta
	}
	
	// Default to incremental
	return ScanTypeIncremental
}

// performFullScan performs a full scan
func (is *IncrementalScanner) performFullScan(ctx context.Context, scanID string) (*ScanResult, error) {
	startTime := time.Now()
	
	// Collect all resources
	snapshot, err := is.collector.Collect(ctx, is.config)
	if err != nil {
		return nil, fmt.Errorf("failed to collect resources: %w", err)
	}
	
	// Process resources
	newResources := 0
	modifiedResources := 0
	deletedResources := 0
	
	currentResourceIDs := make(map[string]bool)
	
	for _, resource := range snapshot.Resources {
		currentResourceIDs[resource.ID] = true
		
		if existing, exists := is.resourceIndex[resource.ID]; exists {
			// Check if modified
			if is.resourceChanged(existing.Resource, resource) {
				modifiedResources++
				is.updateResourceSnapshot(resource, scanID)
			}
		} else {
			// New resource
			newResources++
			is.addResourceSnapshot(resource, scanID)
		}
	}
	
	// Check for deleted resources
	for resourceID := range is.resourceIndex {
		if !currentResourceIDs[resourceID] {
			deletedResources++
			delete(is.resourceIndex, resourceID)
			delete(is.hashIndex, resourceID)
		}
	}
	
	result := &ScanResult{
		ScanID:            scanID,
		Timestamp:         startTime,
		Duration:          time.Since(startTime),
		ResourceCount:     len(snapshot.Resources),
		ChangesFound:      newResources + modifiedResources + deletedResources,
		NewResources:      newResources,
		DeletedResources:  deletedResources,
		ModifiedResources: modifiedResources,
		ErrorCount:        0,
		ScanType:          ScanTypeFull,
		Metadata: map[string]interface{}{
			"total_resources_indexed": len(is.resourceIndex),
			"cache_hit_rate":         is.calculateCacheHitRate(),
		},
	}
	
	is.stats.LastFullScan = time.Now()
	
	return result, nil
}

// performIncrementalScan performs an incremental scan
func (is *IncrementalScanner) performIncrementalScan(ctx context.Context, scanID string) (*ScanResult, error) {
	startTime := time.Now()
	
	// Use prediction model to determine resources to check
	var resourcesToCheck []string
	
	if is.changeDetector.predictionModel.enabled {
		resourcesToCheck = is.changeDetector.predictionModel.getPredictedChanges()
	}
	
	// If no predictions, fall back to checking recently changed resources
	if len(resourcesToCheck) == 0 {
		resourcesToCheck = is.getRecentlyChangedResources()
	}
	
	// Perform targeted collection if possible
	var snapshot *types.Snapshot
	var err error
	
	if len(resourcesToCheck) > 0 && len(resourcesToCheck) < is.scanningStrategy.TargetedThreshold {
		snapshot, err = is.performTargetedCollection(ctx, resourcesToCheck)
	} else {
		snapshot, err = is.collector.Collect(ctx, is.config)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to collect resources: %w", err)
	}
	
	// Process only changed resources
	changesFound := 0
	for _, resource := range snapshot.Resources {
		if existing, exists := is.resourceIndex[resource.ID]; exists {
			if is.resourceChanged(existing.Resource, resource) {
				changesFound++
				is.updateResourceSnapshot(resource, scanID)
			}
		}
	}
	
	result := &ScanResult{
		ScanID:            scanID,
		Timestamp:         startTime,
		Duration:          time.Since(startTime),
		ResourceCount:     len(snapshot.Resources),
		ChangesFound:      changesFound,
		NewResources:      0,
		DeletedResources:  0,
		ModifiedResources: changesFound,
		ErrorCount:        0,
		ScanType:          ScanTypeIncremental,
		Metadata: map[string]interface{}{
			"resources_checked":       len(resourcesToCheck),
			"prediction_model_used":   is.changeDetector.predictionModel.enabled,
			"cache_hit_rate":         is.calculateCacheHitRate(),
		},
	}
	
	is.stats.LastIncrementalScan = time.Now()
	
	return result, nil
}

// performTargetedScan performs a targeted scan on specific resources
func (is *IncrementalScanner) performTargetedScan(ctx context.Context, scanID string) (*ScanResult, error) {
	startTime := time.Now()
	
	// Get resources to target
	targetResources := is.getTargetedResources()
	
	// Perform targeted collection
	snapshot, err := is.performTargetedCollection(ctx, targetResources)
	if err != nil {
		return nil, fmt.Errorf("failed to perform targeted collection: %w", err)
	}
	
	// Process results
	changesFound := 0
	for _, resource := range snapshot.Resources {
		if existing, exists := is.resourceIndex[resource.ID]; exists {
			if is.resourceChanged(existing.Resource, resource) {
				changesFound++
				is.updateResourceSnapshot(resource, scanID)
			}
		}
	}
	
	result := &ScanResult{
		ScanID:            scanID,
		Timestamp:         startTime,
		Duration:          time.Since(startTime),
		ResourceCount:     len(snapshot.Resources),
		ChangesFound:      changesFound,
		NewResources:      0,
		DeletedResources:  0,
		ModifiedResources: changesFound,
		ErrorCount:        0,
		ScanType:          ScanTypeTargeted,
		Metadata: map[string]interface{}{
			"targeted_resources": len(targetResources),
			"cache_hit_rate":    is.calculateCacheHitRate(),
		},
	}
	
	return result, nil
}

// performDeltaScan performs a delta scan
func (is *IncrementalScanner) performDeltaScan(ctx context.Context, scanID string) (*ScanResult, error) {
	startTime := time.Now()
	
	// Get resources that have changed since last scan
	deltaResources := is.getDeltaResources()
	
	// Perform collection only on delta resources
	snapshot, err := is.performTargetedCollection(ctx, deltaResources)
	if err != nil {
		return nil, fmt.Errorf("failed to perform delta collection: %w", err)
	}
	
	// Process results
	changesFound := 0
	for _, resource := range snapshot.Resources {
		if existing, exists := is.resourceIndex[resource.ID]; exists {
			if is.resourceChanged(existing.Resource, resource) {
				changesFound++
				is.updateResourceSnapshot(resource, scanID)
			}
		}
	}
	
	result := &ScanResult{
		ScanID:            scanID,
		Timestamp:         startTime,
		Duration:          time.Since(startTime),
		ResourceCount:     len(snapshot.Resources),
		ChangesFound:      changesFound,
		NewResources:      0,
		DeletedResources:  0,
		ModifiedResources: changesFound,
		ErrorCount:        0,
		ScanType:          ScanTypeDelta,
		Metadata: map[string]interface{}{
			"delta_resources": len(deltaResources),
			"cache_hit_rate":  is.calculateCacheHitRate(),
		},
	}
	
	return result, nil
}

// resourceChanged checks if a resource has changed
func (is *IncrementalScanner) resourceChanged(oldResource, newResource types.Resource) bool {
	oldHash := is.calculateResourceHash(oldResource)
	newHash := is.calculateResourceHash(newResource)
	
	return oldHash != newHash
}

// calculateResourceHash calculates a hash for a resource
func (is *IncrementalScanner) calculateResourceHash(resource types.Resource) string {
	// Create a simplified version for hashing
	hashData := struct {
		ID            string                 `json:"id"`
		Type          string                 `json:"type"`
		Name          string                 `json:"name"`
		Configuration map[string]interface{} `json:"configuration"`
		Tags          map[string]string      `json:"tags"`
	}{
		ID:            resource.ID,
		Type:          resource.Type,
		Name:          resource.Name,
		Configuration: resource.Configuration,
		Tags:          resource.Tags,
	}
	
	data, _ := json.Marshal(hashData)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// addResourceSnapshot adds a resource snapshot to the index
func (is *IncrementalScanner) addResourceSnapshot(resource types.Resource, scanID string) {
	snapshot := ResourceSnapshot{
		Resource:      resource,
		Hash:          is.calculateResourceHash(resource),
		Timestamp:     time.Now(),
		ScanID:        scanID,
		ChangeCount:   1,
		LastModified:  time.Now(),
		ChecksumValid: true,
	}
	
	is.resourceIndex[resource.ID] = snapshot
	is.hashIndex[resource.ID] = snapshot.Hash
}

// updateResourceSnapshot updates a resource snapshot
func (is *IncrementalScanner) updateResourceSnapshot(resource types.Resource, scanID string) {
	existing := is.resourceIndex[resource.ID]
	
	snapshot := ResourceSnapshot{
		Resource:      resource,
		Hash:          is.calculateResourceHash(resource),
		Timestamp:     time.Now(),
		ScanID:        scanID,
		ChangeCount:   existing.ChangeCount + 1,
		LastModified:  time.Now(),
		ChecksumValid: true,
	}
	
	is.resourceIndex[resource.ID] = snapshot
	is.hashIndex[resource.ID] = snapshot.Hash
}

// Helper methods for scan optimization
func (is *IncrementalScanner) shouldDoTargetedScan() bool {
	// Check if we have enough change patterns to justify targeted scanning
	return len(is.changeDetector.changePatterns) > 0
}

func (is *IncrementalScanner) shouldDoDeltaScan() bool {
	// Check if change rate is below threshold
	if len(is.scanHistory) < 2 {
		return false
	}
	
	lastScan := is.scanHistory[len(is.scanHistory)-1]
	changeRate := float64(lastScan.ChangesFound) / float64(lastScan.ResourceCount)
	
	return changeRate < is.scanningStrategy.DeltaChangeRatio
}

func (is *IncrementalScanner) getRecentlyChangedResources() []string {
	var resources []string
	cutoff := time.Now().Add(-is.scanningStrategy.IncrementalInterval)
	
	for id, snapshot := range is.resourceIndex {
		if snapshot.LastModified.After(cutoff) {
			resources = append(resources, id)
		}
	}
	
	return resources
}

func (is *IncrementalScanner) getTargetedResources() []string {
	// Get resources from prediction model
	if is.changeDetector.predictionModel.enabled {
		return is.changeDetector.predictionModel.getPredictedChanges()
	}
	
	// Fall back to high-change resources
	return is.getHighChangeResources()
}

func (is *IncrementalScanner) getHighChangeResources() []string {
	var resources []string
	threshold := 3 // Resources with more than 3 changes
	
	for id, snapshot := range is.resourceIndex {
		if snapshot.ChangeCount > threshold {
			resources = append(resources, id)
		}
	}
	
	return resources
}

func (is *IncrementalScanner) getDeltaResources() []string {
	// Get resources that have changed since last scan
	var resources []string
	
	if len(is.scanHistory) == 0 {
		return resources
	}
	
	lastScan := is.scanHistory[len(is.scanHistory)-1]
	cutoff := lastScan.Timestamp
	
	for id, snapshot := range is.resourceIndex {
		if snapshot.LastModified.After(cutoff) {
			resources = append(resources, id)
		}
	}
	
	return resources
}

func (is *IncrementalScanner) performTargetedCollection(ctx context.Context, resourceIDs []string) (*types.Snapshot, error) {
	// This is a simplified implementation
	// In a real implementation, you'd want to modify the collector to support targeted collection
	return is.collector.Collect(ctx, is.config)
}

func (is *IncrementalScanner) calculateCacheHitRate() float64 {
	// Simple calculation based on resource index size
	if len(is.resourceIndex) == 0 {
		return 0.0
	}
	
	// This is a placeholder - in a real implementation, you'd track actual cache hits
	return 0.85
}

func (is *IncrementalScanner) updateStats(result *ScanResult) {
	is.stats.TotalScans++
	is.stats.TotalChangesDetected += int64(result.ChangesFound)
	
	// Update scan type counters
	switch result.ScanType {
	case ScanTypeFull:
		is.stats.FullScans++
	case ScanTypeIncremental:
		is.stats.IncrementalScans++
	case ScanTypeTargeted:
		is.stats.TargetedScans++
	case ScanTypeDelta:
		is.stats.DeltaScans++
	}
	
	// Update average scan time
	if is.stats.AverageScanTime == 0 {
		is.stats.AverageScanTime = result.Duration
	} else {
		is.stats.AverageScanTime = time.Duration((int64(is.stats.AverageScanTime) + int64(result.Duration)) / 2)
	}
	
	// Update change detection rate
	if is.stats.TotalScans > 0 {
		is.stats.ChangeDetectionRate = float64(is.stats.TotalChangesDetected) / float64(is.stats.TotalScans)
	}
	
	// Update scan efficiency (changes per second)
	if result.Duration > 0 {
		currentEfficiency := float64(result.ChangesFound) / result.Duration.Seconds()
		if is.stats.ScanEfficiency == 0 {
			is.stats.ScanEfficiency = currentEfficiency
		} else {
			is.stats.ScanEfficiency = (is.stats.ScanEfficiency + currentEfficiency) / 2
		}
	}
}

func (is *IncrementalScanner) addToHistory(result ScanResult) {
	is.scanHistory = append(is.scanHistory, result)
	
	// Limit history size
	if len(is.scanHistory) > is.maxHistorySize {
		is.scanHistory = is.scanHistory[len(is.scanHistory)-is.maxHistorySize:]
	}
}

// PredictionModel methods
func (pm *PredictionModel) getPredictedChanges() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if !pm.enabled {
		return []string{}
	}
	
	var predicted []string
	now := time.Now()
	
	for resourceID, prediction := range pm.resourcePredictions {
		if prediction.NextChangeTime.Before(now) && prediction.ChangeProbability > 0.5 {
			predicted = append(predicted, resourceID)
		}
	}
	
	return predicted
}

// GetStats returns the incremental scanner statistics
func (is *IncrementalScanner) GetStats() IncrementalScannerStats {
	is.mu.RLock()
	defer is.mu.RUnlock()
	
	is.stats.ResourceCacheHitRate = is.calculateCacheHitRate()
	return is.stats
}

// GetScanHistory returns the scan history
func (is *IncrementalScanner) GetScanHistory() []ScanResult {
	is.mu.RLock()
	defer is.mu.RUnlock()
	
	history := make([]ScanResult, len(is.scanHistory))
	copy(history, is.scanHistory)
	return history
}

// IsEnabled returns whether the incremental scanner is enabled
func (is *IncrementalScanner) IsEnabled() bool {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return is.enabled
}

// SetEnabled enables or disables the incremental scanner
func (is *IncrementalScanner) SetEnabled(enabled bool) {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.enabled = enabled
}