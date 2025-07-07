package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// TerraformCollector implements the EnhancedCollector interface for Terraform
type TerraformCollector struct {
	parser     *StateParser
	normalizer *ResourceNormalizer
	remote     *RemoteStateHandler
	version    string
}

// NewTerraformCollector creates a new Terraform collector
func NewTerraformCollector() collectors.EnhancedCollector {
	return &TerraformCollector{
		parser:     NewStateParser(),
		normalizer: NewResourceNormalizer(),
		remote:     NewRemoteStateHandler(),
		version:    "1.0.0",
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
	
	var allResources []types.Resource
	
	// Process each state file path
	for _, statePath := range config.StatePaths {
		resources, err := c.collectFromStatePath(ctx, statePath)
		if err != nil {
			return nil, fmt.Errorf("failed to collect from %s: %w", statePath, err)
		}
		allResources = append(allResources, resources...)
	}
	
	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("terraform-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "terraform",
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: c.version,
			CollectionTime:   time.Since(startTime),
			ResourceCount:    len(allResources),
			Tags:             config.Tags,
		},
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

// collectFromLocalState processes local Terraform state files
func (c *TerraformCollector) collectFromLocalState(statePath string) ([]types.Resource, error) {
	// Check if it's a directory (look for terraform.tfstate or .terraform/terraform.tfstate)
	if info, err := os.Stat(statePath); err == nil && info.IsDir() {
		return c.collectFromDirectory(statePath)
	}
	
	// Single state file
	if !strings.HasSuffix(statePath, ".tfstate") {
		return nil, fmt.Errorf("state file must have .tfstate extension: %s", statePath)
	}
	
	tfState, err := c.parser.ParseStateFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse state file %s: %w", statePath, err)
	}
	
	return c.normalizer.NormalizeResources(tfState)
}

// collectFromDirectory looks for state files in a directory
func (c *TerraformCollector) collectFromDirectory(dirPath string) ([]types.Resource, error) {
	var allResources []types.Resource
	
	// Common state file locations
	stateFiles := []string{
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
					stateFiles = append(stateFiles, stateFile)
				}
			}
		}
	}
	
	// Process each found state file
	for _, stateFile := range stateFiles {
		if _, err := os.Stat(stateFile); err == nil {
			resources, err := c.collectFromLocalState(stateFile)
			if err != nil {
				// Log warning but continue with other state files
				continue
			}
			allResources = append(allResources, resources...)
		}
	}
	
	return allResources, nil
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
	var statePaths []string
	
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return collectors.CollectorConfig{}, err
	}
	
	// Look for common state file patterns
	patterns := []string{
		"terraform.tfstate",
		"**/terraform.tfstate",
		"**/*.tfstate",
		".terraform/terraform.tfstate",
		"terraform.tfstate.d/*/terraform.tfstate",
	}
	
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(cwd, pattern))
		if err != nil {
			continue
		}
		statePaths = append(statePaths, matches...)
	}
	
	// Remove duplicates
	uniquePaths := make(map[string]bool)
	var finalPaths []string
	for _, path := range statePaths {
		if !uniquePaths[path] {
			uniquePaths[path] = true
			finalPaths = append(finalPaths, path)
		}
	}
	
	return collectors.CollectorConfig{
		StatePaths: finalPaths,
		Config: map[string]interface{}{
			"auto_discovered": true,
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