package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// ConcurrentScanner manages parallel provider scanning for massive speed improvements
type ConcurrentScanner struct {
	providers   map[string]collectors.EnhancedCollector
	workerPool  chan struct{} // Limit concurrent operations
	maxWorkers  int
	timeout     time.Duration
	clientPool  *ProviderClientPool
	mu          sync.RWMutex
}

// ProviderResult holds the result of a provider scan
type ProviderResult struct {
	Provider string
	Snapshot *types.Snapshot
	Error    error
	Duration time.Duration
}

// CombinedSnapshot represents the merged result of multiple provider scans
type CombinedSnapshot struct {
	*types.Snapshot
	ProviderResults map[string]*ProviderResult
	TotalDuration   time.Duration
	ErrorCount      int
	SuccessCount    int
}

// ScanConfig holds configuration for concurrent scanning
type ScanConfig struct {
	Providers       map[string]collectors.CollectorConfig
	MaxWorkers      int
	Timeout         time.Duration
	FailOnError     bool
	EnablePooling   bool
	SkipMerging     bool
	PreferredOrder  []string // Order preference for provider scanning
}

// NewConcurrentScanner creates a new concurrent scanner with optimized defaults
func NewConcurrentScanner(maxWorkers int, timeout time.Duration) *ConcurrentScanner {
	if maxWorkers <= 0 {
		maxWorkers = 4 // Default to 4 concurrent providers
	}
	
	if timeout <= 0 {
		timeout = 5 * time.Minute // Default timeout
	}

	scanner := &ConcurrentScanner{
		providers:   make(map[string]collectors.EnhancedCollector),
		workerPool:  make(chan struct{}, maxWorkers),
		maxWorkers:  maxWorkers,
		timeout:     timeout,
		clientPool:  NewProviderClientPool(maxWorkers * 10), // 10x connection pool
	}

	// Pre-fill worker pool
	for i := 0; i < maxWorkers; i++ {
		scanner.workerPool <- struct{}{}
	}

	return scanner
}

// RegisterProvider registers a provider collector for concurrent scanning
func (s *ConcurrentScanner) RegisterProvider(name string, collector collectors.EnhancedCollector) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[name] = collector
}

// ScanAllProviders scans all registered providers concurrently
func (s *ConcurrentScanner) ScanAllProviders(ctx context.Context, config ScanConfig) (*CombinedSnapshot, error) {
	startTime := time.Now()
	
	// Create context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Results channel with buffer for all providers
	results := make(chan *ProviderResult, len(config.Providers))
	
	// Wait group to track all provider scans
	var wg sync.WaitGroup
	
	// Error tracking
	var errorsMu sync.Mutex
	criticalErrors := make([]error, 0)

	// Launch provider scans concurrently with preferred ordering
	providerOrder := s.getProviderOrder(config.Providers, config.PreferredOrder)
	
	for _, providerName := range providerOrder {
		providerConfig := config.Providers[providerName]
		
		s.mu.RLock()
		provider, exists := s.providers[providerName]
		s.mu.RUnlock()
		
		if !exists {
			// Create error result for missing provider
			results <- &ProviderResult{
				Provider: providerName,
				Error:    fmt.Errorf("provider %s not registered", providerName),
				Duration: 0,
			}
			continue
		}

		wg.Add(1)
		go s.scanProvider(scanCtx, providerName, provider, providerConfig, results, &wg, &errorsMu, &criticalErrors)
	}

	// Close results channel when all scans complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	providerResults := make(map[string]*ProviderResult)
	var snapshots []*types.Snapshot
	
	for result := range results {
		providerResults[result.Provider] = result
		
		if result.Error != nil {
			if config.FailOnError {
				cancel() // Cancel remaining scans
				return nil, fmt.Errorf("provider %s failed: %w", result.Provider, result.Error)
			}
		} else if result.Snapshot != nil {
			snapshots = append(snapshots, result.Snapshot)
		}
	}

	totalDuration := time.Since(startTime)
	
	// Calculate success/error counts
	successCount := 0
	errorCount := 0
	for _, result := range providerResults {
		if result.Error != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// Create combined snapshot
	combinedSnapshot := &CombinedSnapshot{
		ProviderResults: providerResults,
		TotalDuration:   totalDuration,
		ErrorCount:      errorCount,
		SuccessCount:    successCount,
	}

	// Merge snapshots if not skipped
	if !config.SkipMerging && len(snapshots) > 0 {
		merged, err := s.mergeSnapshots(snapshots)
		if err != nil {
			return nil, fmt.Errorf("failed to merge snapshots: %w", err)
		}
		combinedSnapshot.Snapshot = merged
	}

	return combinedSnapshot, nil
}

// scanProvider performs a single provider scan with error handling and metrics
func (s *ConcurrentScanner) scanProvider(
	ctx context.Context,
	providerName string,
	provider collectors.EnhancedCollector,
	config collectors.CollectorConfig,
	results chan<- *ProviderResult,
	wg *sync.WaitGroup,
	errorsMu *sync.Mutex,
	criticalErrors *[]error,
) {
	defer wg.Done()
	
	// Acquire worker from pool
	select {
	case <-s.workerPool:
		// Got worker, continue
	case <-ctx.Done():
		// Context cancelled, return early
		results <- &ProviderResult{
			Provider: providerName,
			Error:    ctx.Err(),
			Duration: 0,
		}
		return
	}
	
	// Release worker back to pool when done
	defer func() {
		s.workerPool <- struct{}{}
	}()

	startTime := time.Now()
	result := &ProviderResult{
		Provider: providerName,
	}

	// Validate provider configuration
	if err := provider.Validate(config); err != nil {
		result.Error = fmt.Errorf("configuration validation failed: %w", err)
		result.Duration = time.Since(startTime)
		results <- result
		return
	}

	// Perform the actual collection
	snapshot, err := provider.Collect(ctx, config)
	result.Duration = time.Since(startTime)
	
	if err != nil {
		result.Error = fmt.Errorf("collection failed: %w", err)
		
		// Track critical errors
		errorsMu.Lock()
		*criticalErrors = append(*criticalErrors, err)
		errorsMu.Unlock()
	} else {
		result.Snapshot = snapshot
	}

	results <- result
}

// mergeSnapshots combines multiple snapshots into a single unified snapshot
func (s *ConcurrentScanner) mergeSnapshots(snapshots []*types.Snapshot) (*types.Snapshot, error) {
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots to merge")
	}

	if len(snapshots) == 1 {
		return snapshots[0], nil
	}

	// Create combined snapshot
	combined := &types.Snapshot{
		ID:        fmt.Sprintf("combined-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "combined",
		Resources: make([]types.Resource, 0),
	}

	// Resource deduplication map
	resourceMap := make(map[string]types.Resource)
	
	// Process snapshots concurrently for better performance
	type snapshotResult struct {
		resources []types.Resource
		err       error
	}
	
	resultChan := make(chan snapshotResult, len(snapshots))
	
	// Process each snapshot in parallel
	for _, snapshot := range snapshots {
		go func(snap *types.Snapshot) {
			result := snapshotResult{
				resources: s.normalizeResources(snap.Resources),
			}
			resultChan <- result
		}(snapshot)
	}
	
	// Collect normalized resources
	allResources := make([]types.Resource, 0)
	for i := 0; i < len(snapshots); i++ {
		result := <-resultChan
		if result.err != nil {
			return nil, fmt.Errorf("failed to normalize resources: %w", result.err)
		}
		allResources = append(allResources, result.resources...)
	}

	// Deduplicate resources by ID and provider
	for _, resource := range allResources {
		key := fmt.Sprintf("%s:%s:%s", resource.Provider, resource.Type, resource.ID)
		
		if existing, exists := resourceMap[key]; exists {
			// Merge resource configurations if they're different
			merged := s.mergeResourceConfigurations(existing, resource)
			resourceMap[key] = merged
		} else {
			resourceMap[key] = resource
		}
	}

	// Convert map back to slice
	for _, resource := range resourceMap {
		combined.Resources = append(combined.Resources, resource)
	}

	return combined, nil
}

// normalizeResources ensures resources have consistent formatting
func (s *ConcurrentScanner) normalizeResources(resources []types.Resource) []types.Resource {
	normalized := make([]types.Resource, len(resources))
	
	for i, resource := range resources {
		normalized[i] = types.Resource{
			ID:            resource.ID,
			Type:          resource.Type,
			Name:          resource.Name,
			Provider:      resource.Provider,
			Namespace:     resource.Namespace,
			Configuration: resource.Configuration,
		}
		
		// Ensure configuration is not nil
		if normalized[i].Configuration == nil {
			normalized[i].Configuration = make(map[string]interface{})
		}
	}
	
	return normalized
}

// mergeResourceConfigurations merges two resource configurations
func (s *ConcurrentScanner) mergeResourceConfigurations(existing, new types.Resource) types.Resource {
	merged := existing
	
	// Merge configurations
	if new.Configuration != nil {
		for key, value := range new.Configuration {
			if existing.Configuration[key] == nil {
				merged.Configuration[key] = value
			}
		}
	}
	
	// Use non-empty fields from new resource
	if new.Name != "" && existing.Name == "" {
		merged.Name = new.Name
	}
	if new.Namespace != "" && existing.Namespace == "" {
		merged.Namespace = new.Namespace
	}
	
	return merged
}

// getProviderOrder returns the optimal order for scanning providers
func (s *ConcurrentScanner) getProviderOrder(providers map[string]collectors.CollectorConfig, preferred []string) []string {
	// Start with preferred order
	order := make([]string, 0, len(providers))
	used := make(map[string]bool)
	
	// Add preferred providers first
	for _, provider := range preferred {
		if _, exists := providers[provider]; exists {
			order = append(order, provider)
			used[provider] = true
		}
	}
	
	// Add remaining providers
	for provider := range providers {
		if !used[provider] {
			order = append(order, provider)
		}
	}
	
	return order
}

// GetStats returns statistics about the concurrent scanner
func (s *ConcurrentScanner) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"registered_providers": len(s.providers),
		"max_workers":         s.maxWorkers,
		"timeout":            s.timeout.String(),
		"connection_pool_size": s.clientPool.maxConns,
	}
}

// Close shuts down the concurrent scanner and cleans up resources
func (s *ConcurrentScanner) Close() error {
	s.clientPool.Close()
	return nil
}