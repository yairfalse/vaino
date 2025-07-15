package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/discovery"
	"github.com/yairfalse/vaino/pkg/types"
)

// TerraformCollector implements the EnhancedCollector interface for Terraform
type TerraformCollector struct {
	parser            *StateParser
	parallelParser    *ParallelStateParser
	streamParser      *StreamingParser
	normalizer        *ResourceNormalizer
	remote            *RemoteStateHandler
	resourceExtractor *OptimizedResourceExtractor
	version           string
}

// NewTerraformCollector creates a new Terraform collector
func NewTerraformCollector() collectors.EnhancedCollector {
	return &TerraformCollector{
		parser:            NewStateParser(),
		parallelParser:    NewParallelStateParser(),
		streamParser:      NewStreamingParser(),
		normalizer:        NewResourceNormalizer(),
		remote:            NewRemoteStateHandler(),
		resourceExtractor: NewOptimizedResourceExtractor(),
		version:           "1.0.0",
	}
}

// Name returns the collector name
func (c *TerraformCollector) Name() string {
	return "terraform"
}

// Status returns the current status of the collector
func (c *TerraformCollector) Status() string {
	// Check if terraform is available in PATH
	if _, err := os.Stat("/usr/local/bin/terraform"); err == nil {
		return "ready"
	}
	if _, err := os.Stat("/usr/bin/terraform"); err == nil {
		return "ready"
	}

	// Check common installation paths
	paths := []string{
		"/opt/homebrew/bin/terraform",
		"/usr/local/bin/terraform",
		"/usr/bin/terraform",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return "ready"
		}
	}

	return "terraform not found in PATH"
}

// Collect gathers Terraform resources and creates a snapshot
func (c *TerraformCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	startTime := time.Now()

	// Validate configuration
	if err := c.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Separate local and remote state paths
	localPaths, remotePaths := c.separateStatePaths(config.StatePaths)

	var allResources []types.Resource
	var parseStats map[string]interface{}

	// Process local state files in parallel if we have multiple
	if len(localPaths) > 1 {
		resources, stats, err := c.collectFromMultipleLocalStates(ctx, localPaths)
		if err != nil {
			return nil, fmt.Errorf("failed to collect from local state files: %w", err)
		}
		allResources = append(allResources, resources...)
		parseStats = stats
	} else {
		// Process single local state files one by one
		for _, statePath := range localPaths {
			resources, err := c.collectFromStatePath(ctx, statePath)
			if err != nil {
				return nil, fmt.Errorf("failed to collect from %s: %w", statePath, err)
			}
			allResources = append(allResources, resources...)
		}
	}

	// Process remote state files
	for _, remotePath := range remotePaths {
		resources, err := c.collectFromStatePath(ctx, remotePath)
		if err != nil {
			return nil, fmt.Errorf("failed to collect from remote %s: %w", remotePath, err)
		}
		allResources = append(allResources, resources...)
	}

	collectionTime := time.Since(startTime)

	// Create enhanced metadata with performance stats
	metadata := types.SnapshotMetadata{
		CollectorVersion: c.version,
		CollectionTime:   collectionTime,
		ResourceCount:    len(allResources),
		Tags:             config.Tags,
	}

	// Add parsing statistics if available
	if parseStats != nil {
		if metadata.AdditionalData == nil {
			metadata.AdditionalData = make(map[string]interface{})
		}
		metadata.AdditionalData["parsing_stats"] = parseStats
		metadata.AdditionalData["performance_info"] = map[string]interface{}{
			"total_files_processed": len(config.StatePaths),
			"parallel_processing":   len(localPaths) > 1,
			"collection_time_ms":    collectionTime.Milliseconds(),
		}
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("terraform-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "terraform",
		Resources: allResources,
		Metadata:  metadata,
	}

	return snapshot, nil
}

// CollectSeparate creates separate snapshots for each Terraform codebase/state file
func (c *TerraformCollector) CollectSeparate(ctx context.Context, config collectors.CollectorConfig) ([]*types.Snapshot, error) {

	// Validate configuration
	if err := c.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Group state files by logical codebases
	codebases, err := c.groupStateFilesByCodebase(config.StatePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to group state files by codebase: %w", err)
	}

	var snapshots []*types.Snapshot

	// Create a snapshot for each codebase
	for codebaseName, statePaths := range codebases {
		codebaseConfig := collectors.CollectorConfig{
			StatePaths: statePaths,
			Config:     config.Config,
			Tags:       config.Tags,
		}

		// Add codebase information to tags
		if codebaseConfig.Tags == nil {
			codebaseConfig.Tags = make(map[string]string)
		}
		codebaseConfig.Tags["codebase"] = codebaseName

		snapshot, err := c.collectSingleCodebase(ctx, codebaseConfig, codebaseName)
		if err != nil {
			// Log error but continue with other codebases
			fmt.Printf("Warning: Failed to collect codebase %s: %v\n", codebaseName, err)
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots could be created from any codebase")
	}

	return snapshots, nil
}

// groupStateFilesByCodebase groups state files into logical codebases
func (c *TerraformCollector) groupStateFilesByCodebase(statePaths []string) (map[string][]string, error) {
	codebases := make(map[string][]string)

	for _, statePath := range statePaths {
		codebaseName, err := c.determineCodebaseName(statePath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine codebase for %s: %w", statePath, err)
		}

		codebases[codebaseName] = append(codebases[codebaseName], statePath)
	}

	return codebases, nil
}

// determineCodebaseName determines the logical codebase name from a state file path
func (c *TerraformCollector) determineCodebaseName(statePath string) (string, error) {
	// For remote state, use the path as the codebase name
	if strings.HasPrefix(statePath, "s3://") || strings.HasPrefix(statePath, "azurerm://") || strings.HasPrefix(statePath, "gcs://") {
		return filepath.Base(statePath), nil
	}

	// For local files, use the directory structure to determine codebase
	absPath, err := filepath.Abs(statePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// If it's a directory, use the directory name
	if info, err := os.Stat(statePath); err == nil && info.IsDir() {
		return filepath.Base(absPath), nil
	}

	// For state files, use the parent directory name
	dir := filepath.Dir(absPath)

	// Skip common Terraform directories to get meaningful name
	if filepath.Base(dir) == ".terraform" {
		dir = filepath.Dir(dir)
	}
	if filepath.Base(dir) == "terraform.tfstate.d" {
		dir = filepath.Dir(dir)
	}

	codebaseName := filepath.Base(dir)

	// If we're in the root directory, use the state file name
	if codebaseName == "." || codebaseName == "/" {
		fileName := filepath.Base(statePath)
		// Remove .tfstate extension for cleaner name
		if strings.HasSuffix(fileName, ".tfstate") {
			fileName = strings.TrimSuffix(fileName, ".tfstate")
		}
		return fileName, nil
	}

	return codebaseName, nil
}

// collectSingleCodebase collects resources from a single codebase
func (c *TerraformCollector) collectSingleCodebase(ctx context.Context, config collectors.CollectorConfig, codebaseName string) (*types.Snapshot, error) {
	// Separate local and remote state paths
	localPaths, remotePaths := c.separateStatePaths(config.StatePaths)

	var allResources []types.Resource
	var parseStats map[string]interface{}

	// Process local state files
	if len(localPaths) > 1 {
		resources, stats, err := c.collectFromMultipleLocalStates(ctx, localPaths)
		if err != nil {
			return nil, fmt.Errorf("failed to collect from local state files: %w", err)
		}
		allResources = append(allResources, resources...)
		parseStats = stats
	} else {
		for _, statePath := range localPaths {
			resources, err := c.collectFromStatePath(ctx, statePath)
			if err != nil {
				return nil, fmt.Errorf("failed to collect from %s: %w", statePath, err)
			}
			allResources = append(allResources, resources...)
		}
	}

	// Process remote state files
	for _, remotePath := range remotePaths {
		resources, err := c.collectFromStatePath(ctx, remotePath)
		if err != nil {
			return nil, fmt.Errorf("failed to collect from remote %s: %w", remotePath, err)
		}
		allResources = append(allResources, resources...)
	}

	// Create enhanced metadata
	metadata := types.SnapshotMetadata{
		CollectorVersion: c.version,
		CollectionTime:   time.Since(time.Now().Add(-time.Since(time.Now()))),
		ResourceCount:    len(allResources),
		Tags:             config.Tags,
	}

	// Add codebase-specific metadata
	if metadata.AdditionalData == nil {
		metadata.AdditionalData = make(map[string]interface{})
	}
	metadata.AdditionalData["codebase"] = codebaseName
	metadata.AdditionalData["state_files"] = config.StatePaths
	metadata.AdditionalData["is_separate_codebase"] = true

	// Add parsing statistics if available
	if parseStats != nil {
		metadata.AdditionalData["parsing_stats"] = parseStats
	}

	// Create snapshot with codebase-specific ID
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("terraform-%s-%d", codebaseName, time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "terraform",
		Resources: allResources,
		Metadata:  metadata,
	}

	return snapshot, nil
}

// collectFromStatePath processes a single state file path
func (c *TerraformCollector) collectFromStatePath(ctx context.Context, statePath string) ([]types.Resource, error) {
	// Check if it's a remote state reference
	if strings.HasPrefix(statePath, "s3://") || strings.HasPrefix(statePath, "azurerm://") || strings.HasPrefix(statePath, "gcs://") {
		return c.remote.CollectFromRemoteState(ctx, statePath)
	}

	// Handle local state files
	return c.collectFromLocalState(statePath)
}

// collectFromLocalState processes local Terraform state files with optimized parsing
func (c *TerraformCollector) collectFromLocalState(statePath string) ([]types.Resource, error) {
	// Check if it's a directory (look for terraform.tfstate or .terraform/terraform.tfstate)
	if info, err := os.Stat(statePath); err == nil && info.IsDir() {
		return c.collectFromDirectory(statePath)
	}

	// Single state file
	if !strings.HasSuffix(statePath, ".tfstate") {
		return nil, fmt.Errorf("state file must have .tfstate extension: %s", statePath)
	}

	// Check file size to determine parsing strategy
	if stat, err := os.Stat(statePath); err == nil {
		fileSize := stat.Size()
		if fileSize > 50*1024*1024 { // 50MB threshold
			fmt.Printf("Large state file detected (%d MB), using streaming parser...\n", fileSize/(1024*1024))
			tfState, err := c.streamParser.ParseStateFile(statePath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse large state file %s: %w", statePath, err)
			}
			return c.normalizer.NormalizeResources(tfState)
		}
	}

	// Use standard parser for smaller files
	tfState, err := c.parser.ParseStateFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse state file %s: %w", statePath, err)
	}

	return c.normalizer.NormalizeResources(tfState)
}

// collectFromDirectory looks for state files in a directory with optimized processing
func (c *TerraformCollector) collectFromDirectory(dirPath string) ([]types.Resource, error) {
	// Find all state files in directory
	stateFiles, err := c.findStateFilesInDirectory(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find state files in directory %s: %w", dirPath, err)
	}

	if len(stateFiles) == 0 {
		return []types.Resource{}, nil
	}

	// If we have multiple state files, process them in parallel
	if len(stateFiles) > 1 {
		fmt.Printf("Found %d state files in directory, processing in parallel...\n", len(stateFiles))
		ctx := context.Background()
		resources, _, err := c.collectFromMultipleLocalStates(ctx, stateFiles)
		return resources, err
	}

	// Single state file - process normally
	return c.collectFromLocalState(stateFiles[0])
}

// separateStatePaths separates local and remote state paths
func (c *TerraformCollector) separateStatePaths(statePaths []string) (local []string, remote []string) {
	for _, path := range statePaths {
		if strings.HasPrefix(path, "s3://") || strings.HasPrefix(path, "azurerm://") || strings.HasPrefix(path, "gcs://") {
			remote = append(remote, path)
		} else {
			local = append(local, path)
		}
	}
	return
}

// collectFromMultipleLocalStates processes multiple local state files in parallel
func (c *TerraformCollector) collectFromMultipleLocalStates(ctx context.Context, localPaths []string) ([]types.Resource, map[string]interface{}, error) {
	// Expand directory paths to actual state files
	var allStateFiles []string
	for _, path := range localPaths {
		files, err := c.expandStatePath(path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to expand path %s: %w", path, err)
		}
		allStateFiles = append(allStateFiles, files...)
	}

	if len(allStateFiles) == 0 {
		return []types.Resource{}, nil, nil
	}

	// Parse all state files in parallel
	parseResults, err := c.parallelParser.ParseMultipleStates(ctx, allStateFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("parallel parsing failed: %w", err)
	}

	// Extract resources from parsed states
	extractedResources, err := c.resourceExtractor.ExtractResourcesFromResults(parseResults)
	if err != nil {
		return nil, nil, fmt.Errorf("resource extraction failed: %w", err)
	}

	// Normalize all extracted resources
	var allResources []types.Resource
	for _, extracted := range extractedResources {
		// Create a temporary state with just this resource for normalization
		tempState := &TerraformState{
			Version:          extracted.StateVersion,
			TerraformVersion: extracted.TerraformVersion,
			Resources:        []TerraformResource{extracted.OriginalResource},
		}

		resources, err := c.normalizer.NormalizeResources(tempState)
		if err != nil {
			// Log warning but continue with other resources
			fmt.Printf("Warning: Failed to normalize resource %s.%s: %v\n",
				extracted.OriginalResource.Type, extracted.OriginalResource.Name, err)
			continue
		}

		// Add file path information to metadata
		for i := range resources {
			resources[i].Metadata.StateFile = extracted.FilePath
			resources[i].Metadata.StateVersion = fmt.Sprintf("%d", extracted.StateVersion)
			if resources[i].Metadata.AdditionalData == nil {
				resources[i].Metadata.AdditionalData = make(map[string]interface{})
			}
			resources[i].Metadata.AdditionalData["terraform_version"] = extracted.TerraformVersion
		}

		allResources = append(allResources, resources...)
	}

	// Generate parsing statistics
	stats := c.parallelParser.GetParsingStats(parseResults)

	return allResources, stats, nil
}

// expandStatePath expands a path to actual state files
func (c *TerraformCollector) expandStatePath(statePath string) ([]string, error) {
	// Check if it's a directory
	if info, err := os.Stat(statePath); err == nil && info.IsDir() {
		// Find state files in directory
		return c.findStateFilesInDirectory(statePath)
	} else if err == nil {
		// It's a file
		if !strings.HasSuffix(statePath, ".tfstate") {
			return nil, fmt.Errorf("state file must have .tfstate extension: %s", statePath)
		}
		return []string{statePath}, nil
	} else {
		return nil, fmt.Errorf("cannot access path %s: %w", statePath, err)
	}
}

// findStateFilesInDirectory finds all state files in a directory
func (c *TerraformCollector) findStateFilesInDirectory(dirPath string) ([]string, error) {
	var stateFiles []string

	// Common state file locations
	candidateFiles := []string{
		filepath.Join(dirPath, "terraform.tfstate"),
		filepath.Join(dirPath, ".terraform", "terraform.tfstate"),
	}

	// Look for workspace state files
	workspaceDir := filepath.Join(dirPath, "terraform.tfstate.d")
	if info, err := os.Stat(workspaceDir); err == nil && info.IsDir() {
		workspaces, err := os.ReadDir(workspaceDir)
		if err == nil {
			for _, workspace := range workspaces {
				if workspace.IsDir() {
					stateFile := filepath.Join(workspaceDir, workspace.Name(), "terraform.tfstate")
					candidateFiles = append(candidateFiles, stateFile)
				}
			}
		}
	}

	// Check which files actually exist
	for _, candidate := range candidateFiles {
		if _, err := os.Stat(candidate); err == nil {
			stateFiles = append(stateFiles, candidate)
		}
	}

	return stateFiles, nil
}

// Validate checks if the collector configuration is valid
func (c *TerraformCollector) Validate(config collectors.CollectorConfig) error {
	if len(config.StatePaths) == 0 {
		return fmt.Errorf("at least one state path must be specified")
	}

	for _, statePath := range config.StatePaths {
		if statePath == "" {
			return fmt.Errorf("state path cannot be empty")
		}

		// Skip validation for remote state paths
		if strings.HasPrefix(statePath, "s3://") ||
			strings.HasPrefix(statePath, "azurerm://") ||
			strings.HasPrefix(statePath, "gcs://") {
			continue
		}

		// Check local paths exist
		if _, err := os.Stat(statePath); err != nil {
			return fmt.Errorf("state path does not exist: %s", statePath)
		}
	}

	return nil
}

// AutoDiscover automatically discovers Terraform state files in the current directory
func (c *TerraformCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	// Use the discovery service for intelligent state file discovery
	discoveryService := discovery.NewTerraformDiscovery()

	stateFiles, err := discoveryService.DiscoverStateFiles("")
	if err != nil {
		return collectors.CollectorConfig{}, fmt.Errorf("failed to discover state files: %w", err)
	}

	if len(stateFiles) == 0 {
		return collectors.CollectorConfig{}, fmt.Errorf("no Terraform state files found")
	}

	// Get the most relevant state files (limit to 10 to avoid overwhelming)
	preferredFiles := discoveryService.GetPreferredStateFiles(stateFiles, 10)

	var statePaths []string
	for _, file := range preferredFiles {
		statePaths = append(statePaths, file.Path)
	}

	return collectors.CollectorConfig{
		StatePaths: statePaths,
		Config: map[string]interface{}{
			"auto_discovered":   true,
			"total_files_found": len(stateFiles),
			"files_selected":    len(statePaths),
			"discovery_details": preferredFiles,
		},
	}, nil
}

// SupportedRegions returns the regions supported by this collector
// Terraform can manage resources in any region, so this returns common regions
func (c *TerraformCollector) SupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		"ca-central-1", "sa-east-1",
	}
}
