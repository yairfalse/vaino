package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TerraformState represents the structure of a Terraform state file
type TerraformState struct {
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Outputs          map[string]interface{} `json:"outputs"`
	Resources        []TerraformResource    `json:"resources"`
	Modules          []TerraformModule      `json:"modules,omitempty"` // Legacy format
}

// TerraformResource represents a resource in Terraform state
type TerraformResource struct {
	Mode      string              `json:"mode"`
	Type      string              `json:"type"`
	Name      string              `json:"name"`
	Provider  string              `json:"provider"`
	Instances []TerraformInstance `json:"instances"`
	EachMode  string              `json:"each,omitempty"`
	Module    string              `json:"module,omitempty"`
}

// TerraformInstance represents an instance of a resource
type TerraformInstance struct {
	SchemaVersion       int                    `json:"schema_version"`
	Attributes          map[string]interface{} `json:"attributes"`
	Dependencies        []string               `json:"dependencies,omitempty"`
	CreateBeforeDestroy bool                   `json:"create_before_destroy,omitempty"`
	Tainted             bool                   `json:"tainted,omitempty"`
	Deposed             []interface{}          `json:"deposed,omitempty"`
}

// TerraformModule represents a module in legacy state format (< 0.12)
type TerraformModule struct {
	Path         []string                           `json:"path"`
	Outputs      map[string]interface{}             `json:"outputs"`
	Resources    map[string]LegacyTerraformResource `json:"resources"`
	Dependencies []string                           `json:"dependencies"`
}

// LegacyTerraformResource represents resource format in Terraform < 0.12
type LegacyTerraformResource struct {
	Type         string                `json:"type"`
	Primary      LegacyPrimaryResource `json:"primary"`
	Dependencies []string              `json:"depends_on,omitempty"`
	Provider     string                `json:"provider,omitempty"`
}

// LegacyPrimaryResource represents the primary instance in legacy format
type LegacyPrimaryResource struct {
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
	Tainted    bool                   `json:"tainted,omitempty"`
}

// StateParser handles parsing of Terraform state files
type StateParser struct{}

// NewStateParser creates a new state parser
func NewStateParser() *StateParser {
	return &StateParser{}
}

// ParseStateFile reads and parses a Terraform state file
func (p *StateParser) ParseStateFile(filePath string) (*TerraformState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	if len(data) == 0 {
		return &TerraformState{
			Version:   4,
			Resources: []TerraformResource{},
		}, nil
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file JSON: %w", err)
	}

	// Handle legacy state format (version < 4)
	if state.Version < 4 && len(state.Modules) > 0 {
		return p.convertLegacyState(&state)
	}

	return &state, nil
}

// convertLegacyState converts legacy state format to modern format
func (p *StateParser) convertLegacyState(legacyState *TerraformState) (*TerraformState, error) {
	var modernResources []TerraformResource

	for _, module := range legacyState.Modules {
		for resourceKey, resource := range module.Resources {
			// Parse resource key (e.g., "aws_instance.web" -> type: "aws_instance", name: "web")
			resourceType, resourceName := p.parseResourceKey(resourceKey)

			modernResource := TerraformResource{
				Mode:     "managed",
				Type:     resourceType,
				Name:     resourceName,
				Provider: resource.Provider,
				Instances: []TerraformInstance{
					{
						SchemaVersion: 0,
						Attributes:    resource.Primary.Attributes,
						Dependencies:  resource.Dependencies,
						Tainted:       resource.Primary.Tainted,
					},
				},
				Module: p.formatModulePath(module.Path),
			}

			modernResources = append(modernResources, modernResource)
		}
	}

	return &TerraformState{
		Version:          legacyState.Version,
		TerraformVersion: legacyState.TerraformVersion,
		Serial:           legacyState.Serial,
		Lineage:          legacyState.Lineage,
		Outputs:          legacyState.Outputs,
		Resources:        modernResources,
	}, nil
}

// parseResourceKey splits a resource key into type and name
func (p *StateParser) parseResourceKey(key string) (string, string) {
	// Handle different formats:
	// - "aws_instance.web"
	// - "module.vpc.aws_instance.web"
	// - "aws_instance.web.0" (count)

	parts := []string{}
	current := ""
	inBrackets := false

	for _, char := range key {
		if char == '[' {
			inBrackets = true
			current += string(char)
		} else if char == ']' {
			inBrackets = false
			current += string(char)
		} else if char == '.' && !inBrackets {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) >= 2 {
		// Skip module prefixes
		for i, part := range parts {
			if part != "module" && i+1 < len(parts) {
				// Check if next part is numeric (count index)
				if i+2 < len(parts) {
					if _, err := strconv.Atoi(parts[i+2]); err == nil {
						return part, parts[i+1] // type, name
					}
				}
				return part, parts[i+1] // type, name
			}
		}
	}

	// Fallback
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}

	return "unknown", key
}

// formatModulePath converts module path array to string
func (p *StateParser) formatModulePath(path []string) string {
	if len(path) <= 1 {
		return ""
	}
	// Skip the "root" path element
	if path[0] == "root" {
		path = path[1:]
	}
	if len(path) == 0 {
		return ""
	}

	result := ""
	for i, segment := range path {
		if i > 0 {
			result += "."
		}
		result += segment
	}
	return result
}

// GetResourcesByType returns all resources of a specific type
func (s *TerraformState) GetResourcesByType(resourceType string) []TerraformResource {
	var results []TerraformResource
	for _, resource := range s.Resources {
		if resource.Type == resourceType {
			results = append(results, resource)
		}
	}
	return results
}

// GetResourceByAddress returns a resource by its Terraform address
func (s *TerraformState) GetResourceByAddress(address string) (TerraformResource, bool) {
	for _, resource := range s.Resources {
		if p := (&StateParser{}); p.getResourceAddress(resource) == address {
			return resource, true
		}
	}
	return TerraformResource{}, false
}

// getResourceAddress constructs the Terraform address for a resource
func (p *StateParser) getResourceAddress(resource TerraformResource) string {
	address := resource.Type + "." + resource.Name
	if resource.Module != "" {
		address = "module." + resource.Module + "." + address
	}
	return address
}

// GetCreatedAt attempts to extract creation time from resource attributes
func (r *TerraformResource) GetCreatedAt() time.Time {
	for _, instance := range r.Instances {
		// Common timestamp attributes
		timeAttrs := []string{"created_time", "creation_date", "create_time", "created_at"}

		for _, attr := range timeAttrs {
			if val, exists := instance.Attributes[attr]; exists {
				if timeStr, ok := val.(string); ok {
					// Try different time formats
					formats := []string{
						time.RFC3339,
						"2006-01-02T15:04:05Z",
						"2006-01-02T15:04:05.000Z",
						"2006-01-02 15:04:05",
					}

					for _, format := range formats {
						if t, err := time.Parse(format, timeStr); err == nil {
							return t
						}
					}
				}
			}
		}
	}

	return time.Time{}
}

// GetTags extracts tags from a Terraform resource
func (r *TerraformResource) GetTags() map[string]string {
	tags := make(map[string]string)

	for _, instance := range r.Instances {
		// Check different tag attribute names
		tagAttrs := []string{"tags", "labels", "tag"}

		for _, attr := range tagAttrs {
			if val, exists := instance.Attributes[attr]; exists {
				if tagMap, ok := val.(map[string]interface{}); ok {
					for k, v := range tagMap {
						if strVal, ok := v.(string); ok {
							tags[k] = strVal
						}
					}
				}
			}
		}
	}

	return tags
}

// ParallelStateParser handles multiple state files concurrently
type ParallelStateParser struct {
	parser       *StateParser
	streamParser *StreamingParser
	maxWorkers   int
	timeout      time.Duration
}

// NewParallelStateParser creates a new parallel parser
func NewParallelStateParser() *ParallelStateParser {
	return &ParallelStateParser{
		parser:       NewStateParser(),
		streamParser: NewStreamingParser(),
		maxWorkers:   4, // Default to 4 workers
		timeout:      30 * time.Second,
	}
}

// ParseResult represents the result of parsing a single state file
type ParseResult struct {
	FilePath string
	State    *TerraformState
	Error    error
	Duration time.Duration
	FileSize int64
}

// ParseMultipleStates parses multiple state files in parallel
func (pp *ParallelStateParser) ParseMultipleStates(ctx context.Context, filePaths []string) ([]*ParseResult, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no state files provided")
	}

	// Create results channel and worker pool
	results := make(chan *ParseResult, len(filePaths))
	jobs := make(chan string, len(filePaths))

	// Determine worker count (don't exceed number of files)
	workerCount := pp.maxWorkers
	if len(filePaths) < workerCount {
		workerCount = len(filePaths)
	}

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go pp.worker(ctx, jobs, results, &wg)
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, filePath := range filePaths {
			select {
			case jobs <- filePath:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var parseResults []*ParseResult
	for result := range results {
		parseResults = append(parseResults, result)
	}

	return parseResults, nil
}

// worker processes individual state files
func (pp *ParallelStateParser) worker(ctx context.Context, jobs <-chan string, results chan<- *ParseResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range jobs {
		select {
		case <-ctx.Done():
			results <- &ParseResult{
				FilePath: filePath,
				Error:    fmt.Errorf("parsing cancelled: %w", ctx.Err()),
			}
			return
		default:
			result := pp.parseWithTimeout(ctx, filePath)
			results <- result
		}
	}
}

// parseWithTimeout parses a single file with timeout protection
func (pp *ParallelStateParser) parseWithTimeout(ctx context.Context, filePath string) *ParseResult {
	start := time.Now()

	// Create timeout context
	parseCtx, cancel := context.WithTimeout(ctx, pp.timeout)
	defer cancel()

	result := &ParseResult{
		FilePath: filePath,
		Duration: time.Since(start),
	}

	// Get file info
	if stat, err := os.Stat(filePath); err != nil {
		result.Error = fmt.Errorf("cannot access state file %s: %w", filePath, err)
		return result
	} else {
		result.FileSize = stat.Size()
	}

	// Parse in goroutine with timeout
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Use streaming parser for large files, regular parser for smaller ones
		if result.FileSize > 50*1024*1024 { // 50MB threshold
			state, err := pp.streamParser.ParseStateFile(filePath)
			result.State = state
			result.Error = err
		} else {
			state, err := pp.parser.ParseStateFile(filePath)
			result.State = state
			result.Error = err
		}

		result.Duration = time.Since(start)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		return result
	case <-parseCtx.Done():
		result.Error = fmt.Errorf("parsing state file %s timed out after %v", filePath, pp.timeout)
		return result
	}
}

// ValidateStateFiles validates multiple state files in parallel
func (pp *ParallelStateParser) ValidateStateFiles(ctx context.Context, filePaths []string) ([]string, []error) {
	var validFiles []string
	var errors []error

	// Create channels for validation
	type validationResult struct {
		filePath string
		error    error
	}

	results := make(chan validationResult, len(filePaths))
	jobs := make(chan string, len(filePaths))

	// Start validation workers
	var wg sync.WaitGroup
	workerCount := pp.maxWorkers
	if len(filePaths) < workerCount {
		workerCount = len(filePaths)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				err := pp.streamParser.ValidateStateFile(filePath)
				results <- validationResult{filePath: filePath, error: err}
			}
		}()
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, filePath := range filePaths {
			select {
			case jobs <- filePath:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.error != nil {
			errors = append(errors, fmt.Errorf("validation failed for %s: %w", result.filePath, result.error))
		} else {
			validFiles = append(validFiles, result.filePath)
		}
	}

	return validFiles, errors
}

// GetParsingStats returns statistics about parsing performance
func (pp *ParallelStateParser) GetParsingStats(results []*ParseResult) map[string]interface{} {
	stats := make(map[string]interface{})

	var totalFiles, successFiles, failedFiles int
	var totalSize, totalDuration int64
	var avgDuration time.Duration

	for _, result := range results {
		totalFiles++
		totalSize += result.FileSize
		totalDuration += int64(result.Duration)

		if result.Error != nil {
			failedFiles++
		} else {
			successFiles++
		}
	}

	if totalFiles > 0 {
		avgDuration = time.Duration(totalDuration / int64(totalFiles))
	}

	stats["total_files"] = totalFiles
	stats["successful_files"] = successFiles
	stats["failed_files"] = failedFiles
	stats["total_size_mb"] = float64(totalSize) / (1024 * 1024)
	stats["average_parse_time"] = avgDuration.String()
	stats["success_rate"] = float64(successFiles) / float64(totalFiles) * 100

	return stats
}

// OptimizedResourceExtractor efficiently extracts resources from parsed states
type OptimizedResourceExtractor struct {
	resourceTypeCache map[string]bool
	mu                sync.RWMutex
}

// NewOptimizedResourceExtractor creates a new optimized extractor
func NewOptimizedResourceExtractor() *OptimizedResourceExtractor {
	return &OptimizedResourceExtractor{
		resourceTypeCache: make(map[string]bool),
	}
}

// ExtractResourcesFromResults extracts resources from multiple parse results
func (ore *OptimizedResourceExtractor) ExtractResourcesFromResults(results []*ParseResult) ([]ExtractedResource, error) {
	var allResources []ExtractedResource
	var mu sync.Mutex

	// Process results in parallel
	var wg sync.WaitGroup

	for _, result := range results {
		if result.Error != nil || result.State == nil {
			continue
		}

		wg.Add(1)
		go func(r *ParseResult) {
			defer wg.Done()

			resources := ore.extractFromState(r.State, r.FilePath)

			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(result)
	}

	wg.Wait()

	return allResources, nil
}

// extractFromState extracts resources from a single state
func (ore *OptimizedResourceExtractor) extractFromState(state *TerraformState, filePath string) []ExtractedResource {
	var resources []ExtractedResource

	for _, tfResource := range state.Resources {
		// Skip data sources and other non-managed resources
		if tfResource.Mode != "managed" {
			continue
		}

		// Check if resource type is supported (with caching)
		if !ore.isResourceTypeSupported(tfResource.Type) {
			continue
		}

		for instanceIndex, instance := range tfResource.Instances {
			resource := ExtractedResource{
				OriginalResource: tfResource,
				InstanceIndex:    instanceIndex,
				Instance:         instance,
				FilePath:         filePath,
				StateVersion:     state.Version,
				TerraformVersion: state.TerraformVersion,
			}

			resources = append(resources, resource)
		}
	}

	return resources
}

// isResourceTypeSupported checks if a resource type is supported (with caching)
func (ore *OptimizedResourceExtractor) isResourceTypeSupported(resourceType string) bool {
	ore.mu.RLock()
	supported, exists := ore.resourceTypeCache[resourceType]
	ore.mu.RUnlock()

	if exists {
		return supported
	}

	// Determine support based on prefix
	supportedPrefixes := []string{
		"aws_", "azurerm_", "google_", "kubernetes_",
		"helm_", "docker_", "local_", "random_",
		"tls_", "http_", "external_", "archive_",
	}

	supported = false
	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(resourceType, prefix) {
			supported = true
			break
		}
	}

	// Cache the result
	ore.mu.Lock()
	ore.resourceTypeCache[resourceType] = supported
	ore.mu.Unlock()

	return supported
}

// ExtractedResource represents a resource extracted from state
type ExtractedResource struct {
	OriginalResource TerraformResource
	InstanceIndex    int
	Instance         TerraformInstance
	FilePath         string
	StateVersion     int
	TerraformVersion string
}
