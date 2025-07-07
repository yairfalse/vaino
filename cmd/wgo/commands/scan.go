package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/pkg/types"
)

func newScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan infrastructure for current state",
		Long: `Scan discovers and collects the current state of your infrastructure
from various providers (Terraform, AWS, Kubernetes) and creates a snapshot.

This snapshot can be used as a baseline for future drift detection or 
compared against existing baselines to identify changes.`,
		Example: `  # Scan Terraform state
  wgo scan --provider terraform --path ./terraform

  # Scan AWS resources in multiple regions
  wgo scan --provider aws --region us-east-1,us-west-2

  # Scan Kubernetes cluster
  wgo scan --provider kubernetes --context prod --namespace default,kube-system

  # Scan all providers and save with custom name
  wgo scan --all --output-file my-snapshot.json`,
		RunE: runScan,
	}

	// Flags
	cmd.Flags().StringP("provider", "p", "", "infrastructure provider (terraform, aws, kubernetes)")
	cmd.Flags().StringSliceP("state-file", "s", []string{}, "specific state files to scan (for terraform)")
	cmd.Flags().StringP("output-file", "o", "", "save snapshot to file")
	cmd.Flags().Bool("auto-discover", false, "automatically discover state files")
	cmd.Flags().Bool("all", false, "scan all configured providers")
	cmd.Flags().StringSlice("region", []string{}, "AWS regions to scan (comma-separated)")
	cmd.Flags().String("path", ".", "path to Terraform files")
	cmd.Flags().StringSlice("context", []string{}, "Kubernetes contexts to scan")
	cmd.Flags().StringSlice("namespace", []string{}, "Kubernetes namespaces to scan")
	cmd.Flags().Bool("no-cache", false, "disable caching for this scan")
	cmd.Flags().String("snapshot-name", "", "custom name for the snapshot")
	cmd.Flags().StringSlice("tags", []string{}, "tags to apply to snapshot (key=value)")

	return cmd
}

func runScan(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Infrastructure Scan")
	fmt.Println("=====================")
	
	provider, _ := cmd.Flags().GetString("provider")
	scanAll, _ := cmd.Flags().GetBool("all")
	outputFile, _ := cmd.Flags().GetString("output-file")
	snapshotName, _ := cmd.Flags().GetString("snapshot-name")
	statePaths, _ := cmd.Flags().GetStringSlice("state-file")
	autoDiscover, _ := cmd.Flags().GetBool("auto-discover")
	
	// Initialize the enhanced registry
	enhancedRegistry := collectors.NewEnhancedRegistry()
	terraformCollector := terraform.NewTerraformCollector()
	enhancedRegistry.RegisterEnhanced(terraformCollector)
	
	ctx := cmd.Context()
	
	if !scanAll && provider == "" {
		// List available providers
		enhancedProviders := enhancedRegistry.ListEnhanced()
		
		fmt.Println("Available providers:")
		fmt.Println("\nEnhanced providers (support full collection):")
		for _, name := range enhancedProviders {
			status := "unknown"
			if collector, err := enhancedRegistry.GetEnhanced(name); err == nil {
				status = collector.Status()
			}
			fmt.Printf("  • %s (%s)\n", name, status)
		}
		
		fmt.Println("\nUse --provider <name> to scan a specific provider")
		fmt.Println("Example: wgo scan --provider terraform")
		return nil
	}
	
	if scanAll {
		return fmt.Errorf("--all flag not yet implemented")
	}
	
	// Validate provider
	if !enhancedRegistry.IsEnhanced(provider) {
		fmt.Printf("Error: Provider '%s' is not available or does not support collection\n", provider)
		fmt.Println("Available enhanced providers:")
		for _, name := range enhancedRegistry.ListEnhanced() {
			fmt.Printf("  • %s\n", name)
		}
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	// Get collector
	collector, err := enhancedRegistry.GetEnhanced(provider)
	if err != nil {
		return fmt.Errorf("failed to get collector for provider '%s': %w", provider, err)
	}
	
	// Check collector status
	status := collector.Status()
	if status != "ready" {
		fmt.Printf("Warning: Collector status: %s\n", status)
	}
	
	// Build collector configuration
	var config collectors.CollectorConfig
	
	if autoDiscover {
		fmt.Println("🔍 Auto-discovering configuration...")
		discoveredConfig, err := collector.AutoDiscover()
		if err != nil {
			return fmt.Errorf("auto-discovery failed: %w", err)
		}
		config = discoveredConfig
		fmt.Printf("Found %d state paths\n", len(config.StatePaths))
	} else if len(statePaths) > 0 {
		config.StatePaths = statePaths
	} else {
		// Default configuration for terraform
		if provider == "terraform" {
			config.StatePaths = []string{"./terraform.tfstate", "./"}
		}
	}
	
	// Add common configuration
	config.Config = map[string]interface{}{
		"scan_id": fmt.Sprintf("%s-%d", provider, time.Now().Unix()),
	}
	if snapshotName != "" {
		config.Config["snapshot_name"] = snapshotName
	}
	
	// Validate configuration
	if err := collector.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Perform collection
	fmt.Printf("📊 Collecting resources from %s...\n", provider)
	startTime := time.Now()
	
	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}
	
	collectionTime := time.Since(startTime)
	
	// Display results
	fmt.Printf("\n✅ Collection completed in %v\n", collectionTime)
	fmt.Printf("📋 Snapshot ID: %s\n", snapshot.ID)
	fmt.Printf("📊 Resources found: %d\n", len(snapshot.Resources))
	
	// Group resources by type
	byType := make(map[string]int)
	for _, resource := range snapshot.Resources {
		byType[resource.Type]++
	}
	
	fmt.Println("\n📈 Resource breakdown:")
	for resourceType, count := range byType {
		fmt.Printf("  • %s: %d\n", resourceType, count)
	}
	
	// Save to output file if specified
	if outputFile != "" {
		if err := saveSnapshotToFile(snapshot, outputFile); err != nil {
			fmt.Printf("Warning: Failed to save to output file: %v\n", err)
		} else {
			fmt.Printf("\n📄 Output saved to: %s\n", outputFile)
		}
	}
	
	fmt.Printf("\n💾 Snapshot ready for baseline/drift analysis\n")
	return nil
}

// saveSnapshotToFile saves a snapshot to a JSON file
func saveSnapshotToFile(snapshot *types.Snapshot, filename string) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}
	
	return os.WriteFile(filename, data, 0644)
}