package terraform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

const (
	ProviderName = "terraform"
)

// Collector implements the Collector interface for Terraform state files
type Collector struct {
	name string
}

// NewCollector creates a new Terraform collector
func NewCollector() *Collector {
	return &Collector{
		name: ProviderName,
	}
}

// Name returns the name of the collector
func (c *Collector) Name() string {
	return c.name
}

// Collect gathers resources from Terraform state files and returns a snapshot
func (c *Collector) Collect(ctx context.Context, config collectors.Config) (*types.Snapshot, error) {
	// Validate the configuration
	if err := c.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Find all state files
	stateFiles, err := FindStateFiles(config.Paths)
	if err != nil {
		return nil, fmt.Errorf("failed to find state files: %w", err)
	}

	if len(stateFiles) == 0 {
		return nil, errors.New("no Terraform state files found in the specified paths")
	}

	// Collect resources from all state files
	var allResources []types.Resource
	for _, stateFile := range stateFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resources, err := ParseStateFile(stateFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse state file %s: %w", stateFile, err)
		}

		allResources = append(allResources, resources...)
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        generateSnapshotID(),
		Timestamp: time.Now(),
		Provider:  config.Provider,
		Region:    config.Region,
		Resources: allResources,
		Metadata:  c.buildMetadata(config, stateFiles),
	}

	return snapshot, nil
}

// Validate checks if the provided configuration is valid for this collector
func (c *Collector) Validate(config collectors.Config) error {
	if strings.TrimSpace(config.Provider) == "" {
		return errors.New("provider is required")
	}

	if config.Provider != ProviderName {
		return fmt.Errorf("invalid provider '%s', expected 'terraform'", config.Provider)
	}

	if strings.TrimSpace(config.Region) == "" {
		return errors.New("region is required")
	}

	if len(config.Paths) == 0 {
		return errors.New("at least one path is required")
	}

	// Validate paths are not empty
	for i, path := range config.Paths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("path at index %d cannot be empty", i)
		}
	}

	return nil
}

// buildMetadata creates metadata for the snapshot
func (c *Collector) buildMetadata(config collectors.Config, stateFiles []string) map[string]string {
	metadata := make(map[string]string)

	metadata["collector"] = c.name
	metadata["state_files_count"] = fmt.Sprintf("%d", len(stateFiles))
	metadata["collection_time"] = time.Now().Format(time.RFC3339)

	// Add first few state files to metadata for reference
	if len(stateFiles) > 0 {
		metadata["primary_state_file"] = stateFiles[0]
	}

	// Add workspace information if available
	if workspace := c.extractWorkspace(config); workspace != "" {
		metadata["workspace"] = workspace
	}

	// Add any additional options from config
	for key, value := range config.Options {
		if valueStr, ok := value.(string); ok {
			metadata[fmt.Sprintf("option_%s", key)] = valueStr
		}
	}

	return metadata
}

// extractWorkspace extracts workspace information from config options
func (c *Collector) extractWorkspace(config collectors.Config) string {
	if config.Options == nil {
		return ""
	}

	// Check for workspace in options
	if workspace, ok := config.Options["workspace"]; ok {
		if workspaceStr, ok := workspace.(string); ok {
			return workspaceStr
		}
	}

	return ""
}

// generateSnapshotID generates a unique snapshot ID
func generateSnapshotID() string {
	return fmt.Sprintf("terraform-snapshot-%d", time.Now().Unix())
}

// CollectFromPaths is a convenience method to collect resources from specific paths
func (c *Collector) CollectFromPaths(ctx context.Context, paths []string, region string) (*types.Snapshot, error) {
	config := collectors.Config{
		Provider: ProviderName,
		Region:   region,
		Paths:    paths,
	}

	return c.Collect(ctx, config)
}

// CollectFromWorkspace collects resources from a Terraform workspace
func (c *Collector) CollectFromWorkspace(ctx context.Context, workspacePath, workspace, region string) (*types.Snapshot, error) {
	config := collectors.Config{
		Provider: ProviderName,
		Region:   region,
		Paths:    []string{workspacePath},
		Options: map[string]interface{}{
			"workspace": workspace,
		},
	}

	return c.Collect(ctx, config)
}

// SupportedResourceTypes returns a list of Terraform resource types that this collector recognizes
func (c *Collector) SupportedResourceTypes() []string {
	return []string{
		// AWS resources
		"aws_instance",
		"aws_security_group",
		"aws_vpc",
		"aws_subnet",
		"aws_internet_gateway",
		"aws_route_table",
		"aws_s3_bucket",
		"aws_rds_instance",
		"aws_load_balancer",
		"aws_autoscaling_group",
		"aws_iam_role",
		"aws_iam_policy",
		"aws_lambda_function",
		"aws_cloudwatch_log_group",
		"aws_ecs_cluster",
		"aws_ecs_service",
		"aws_eks_cluster",
		"aws_eks_node_group",

		// Azure resources
		"azurerm_resource_group",
		"azurerm_virtual_network",
		"azurerm_subnet",
		"azurerm_virtual_machine",
		"azurerm_storage_account",
		"azurerm_sql_server",
		"azurerm_kubernetes_cluster",

		// Google Cloud resources
		"google_compute_instance",
		"google_compute_network",
		"google_compute_subnetwork",
		"google_storage_bucket",
		"google_sql_database_instance",
		"google_container_cluster",

		// Generic resources
		"null_resource",
		"random_id",
		"random_password",
		"local_file",
		"template_file",
	}
}

// IsResourceTypeSupported checks if a resource type is supported by this collector
func (c *Collector) IsResourceTypeSupported(resourceType string) bool {
	supportedTypes := c.SupportedResourceTypes()
	for _, supportedType := range supportedTypes {
		if supportedType == resourceType {
			return true
		}
	}
	return false
}
