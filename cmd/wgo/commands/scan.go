package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/aws"
	"github.com/yairfalse/wgo/internal/collectors/gcp"
	"github.com/yairfalse/wgo/internal/collectors/kubernetes"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/internal/discovery"
	wgoerrors "github.com/yairfalse/wgo/internal/errors"
	"github.com/yairfalse/wgo/pkg/config"
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

  # Scan GCP resources
  wgo scan --provider gcp

  # Scan GCP with specific project and regions  
  wgo scan --provider gcp --project my-project-123 --region us-central1,us-east1

  # Scan GCP with custom credentials
  wgo scan --provider gcp --credentials ./service-account.json

  # Scan all providers and save with custom name
  wgo scan --all --output-file my-snapshot.json`,
		RunE: runScan,
	}

	// Flags
	cmd.Flags().StringP("provider", "p", "", "infrastructure provider (terraform, aws, kubernetes, gcp)")
	cmd.Flags().StringSliceP("state-file", "s", []string{}, "specific state files to scan (for terraform)")
	cmd.Flags().StringP("output-file", "o", "", "save snapshot to file")
	cmd.Flags().Bool("auto-discover", false, "automatically discover state files")
	cmd.Flags().Bool("all", false, "scan all configured providers")
	cmd.Flags().StringSlice("region", []string{}, "regions to scan (comma-separated)")
	cmd.Flags().String("path", ".", "path to Terraform files")
	cmd.Flags().StringSlice("context", []string{}, "Kubernetes contexts to scan")
	cmd.Flags().StringSlice("namespace", []string{}, "Kubernetes namespaces to scan")
	cmd.Flags().Bool("no-cache", false, "disable caching for this scan")
	cmd.Flags().String("snapshot-name", "", "custom name for the snapshot")
	cmd.Flags().StringSlice("tags", []string{}, "tags to apply to snapshot (key=value)")
	cmd.Flags().Bool("quiet", false, "suppress output (for automated use)")
	
	// AWS specific flags
	cmd.Flags().String("profile", "", "AWS profile to use")
	
	// GCP specific flags
	cmd.Flags().String("project", "", "GCP project ID")
	cmd.Flags().String("credentials", "", "path to GCP service account credentials JSON file")

	return cmd
}

func runScan(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	
	// Helper to print only when not quiet
	log := func(format string, args ...interface{}) {
		if !quiet {
			fmt.Printf(format, args...)
		}
	}
	
	if !quiet {
		fmt.Println("Infrastructure Scan")
		fmt.Println("===================")
	}
	
	provider, _ := cmd.Flags().GetString("provider")
	scanAll, _ := cmd.Flags().GetBool("all")
	outputFile, _ := cmd.Flags().GetString("output-file")
	snapshotName, _ := cmd.Flags().GetString("snapshot-name")
	statePaths, _ := cmd.Flags().GetStringSlice("state-file")
	autoDiscover, _ := cmd.Flags().GetBool("auto-discover")
	
	// Create smart defaults manager
	defaultsManager := config.NewDefaultsManager()
	
	// Initialize the enhanced registry
	enhancedRegistry := collectors.NewEnhancedRegistry()
	terraformCollector := terraform.NewTerraformCollector()
	enhancedRegistry.RegisterEnhanced(terraformCollector)
	kubernetesCollector := kubernetes.NewKubernetesCollector()
	enhancedRegistry.RegisterEnhanced(kubernetesCollector)
	awsCollector := aws.NewAWSCollector()
	enhancedRegistry.RegisterEnhanced(awsCollector)
	gcpCollector := gcp.NewGCPCollector()
	enhancedRegistry.RegisterEnhanced(gcpCollector)
	
	ctx := cmd.Context()
	
	if !scanAll && provider == "" {
		// Generate smart defaults and auto-discover infrastructure
		log("Auto-discovering infrastructure...\n")
		smartConfig, err := defaultsManager.GenerateSmartDefaults()
		if err != nil {
			return fmt.Errorf("failed to generate smart defaults: %w", err)
		}
		
		// Show user-friendly feedback about what was detected
		if !quiet {
			feedback := defaultsManager.GetUserFriendlyFeedback(smartConfig)
			for _, line := range feedback {
				fmt.Println(line)
			}
		}
		
		discovery := discovery.NewTerraformDiscovery()
		stateFiles, err := discovery.DiscoverStateFiles("")
		
		if err == nil && len(stateFiles) > 0 {
			// Found Terraform state files, use terraform provider
			provider = "terraform"
			autoDiscover = true
			fmt.Printf("Found %d Terraform state file(s), using terraform provider\n", len(stateFiles))
		} else {
			// No auto-discovery possible, show helpful guidance
			// Enhanced first-run experience
			fmt.Println("\nWelcome to WGO")
			fmt.Println("==================")
			fmt.Println()
			
			// Check if this is first run
			homeDir, _ := os.UserHomeDir()
			configPath := filepath.Join(homeDir, ".wgo", "config.yaml")
			isFirstRun := false
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				isFirstRun = true
			}
			
			if isFirstRun {
				fmt.Println("First run detected! Let me set up WGO for you...")
				fmt.Println()
				
				// Run auto-detection
				detector := config.NewProviderDetector()
				detectionResults := detector.DetectAll()
				
				availableProviders := []string{}
				fmt.Println("Detecting available providers:")
				for provider, result := range detectionResults {
					if provider == "terraform" && result.StateFiles > 0 {
						fmt.Printf("  [OK] %s: %d state files found\n", provider, result.StateFiles)
						availableProviders = append(availableProviders, provider)
					} else if result.Available && provider != "terraform" {
						fmt.Printf("  [OK] %s: %s\n", provider, result.Status)
						availableProviders = append(availableProviders, provider)
					}
				}
				
				if len(availableProviders) > 0 {
					fmt.Println()
					fmt.Println("Creating default configuration...")
					if err := createDefaultConfig(configPath); err == nil {
						fmt.Printf("Configuration created at: %s\n", configPath)
					}
				}
			}
			
			fmt.Println()
			fmt.Println("I couldn't auto-detect your infrastructure to scan.")
			fmt.Println()
			fmt.Println("QUICK START - Choose your provider:")
			fmt.Println()
			fmt.Println("  For Terraform projects:")
			fmt.Println("    wgo scan --provider terraform")
			fmt.Println()
			fmt.Println("  For Google Cloud:")
			fmt.Println("    wgo scan --provider gcp --project YOUR-PROJECT-ID")
			fmt.Println()
			fmt.Println("  For AWS:")
			fmt.Println("    wgo scan --provider aws --region us-east-1")
			fmt.Println()
			fmt.Println("  For Kubernetes:")
			fmt.Println("    wgo scan --provider kubernetes")
			fmt.Println()
			fmt.Println("TIP: Run 'wgo status' to check your configuration")
			fmt.Println("        Run 'wgo configure' for interactive setup")
			fmt.Println()
			return nil
		}
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
		return wgoerrors.New(wgoerrors.ErrorTypeProvider, wgoerrors.Provider(provider), 
			fmt.Sprintf("Failed to initialize %s collector", provider)).
			WithCause(err.Error()).
			WithSolutions(
				fmt.Sprintf("Ensure %s provider is properly installed", provider),
				"Run 'wgo check-config' to diagnose issues",
			).
			WithHelp(fmt.Sprintf("wgo help %s", provider))
	}
	
	// Check collector status
	status := collector.Status()
	if status != "ready" {
		fmt.Printf("Warning: Collector status: %s\n", status)
	}
	
	// Build collector configuration
	var config collectors.CollectorConfig
	
	if autoDiscover {
		fmt.Println("Auto-discovering configuration...")
		discoveredConfig, err := collector.AutoDiscover()
		if err != nil {
			return fmt.Errorf("auto-discovery failed: %w", err)
		}
		config = discoveredConfig
		log("Found %d state paths\n", len(config.StatePaths))
	} else if len(statePaths) > 0 {
		config.StatePaths = statePaths
	} else {
		// Provider-specific configuration
		switch provider {
		case "terraform":
			config.StatePaths = []string{"./terraform.tfstate", "./"}
		case "kubernetes":
			// Get Kubernetes-specific flags
			contexts, _ := cmd.Flags().GetStringSlice("context")
			namespaces, _ := cmd.Flags().GetStringSlice("namespace")
			
			config.Namespaces = namespaces
			config.Config = map[string]interface{}{}
			if len(contexts) > 0 {
				config.Config["contexts"] = contexts
			}
		case "aws":
			// Get AWS-specific flags
			profile, _ := cmd.Flags().GetString("profile")
			regions, _ := cmd.Flags().GetStringSlice("region")
			
			config.Config = map[string]interface{}{}
			if profile != "" {
				config.Config["profile"] = profile
			}
			if len(regions) > 0 && regions[0] != "" {
				config.Config["region"] = regions[0] // AWS SDK works with single region
			}
		case "gcp":
			// Get GCP-specific flags
			projectID, _ := cmd.Flags().GetString("project")
			credentialsFile, _ := cmd.Flags().GetString("credentials")
			regions, _ := cmd.Flags().GetStringSlice("region")
			
			config.Config = map[string]interface{}{}
			if projectID != "" {
				config.Config["project_id"] = projectID
			}
			if credentialsFile != "" {
				config.Config["credentials_file"] = credentialsFile
			}
			if len(regions) > 0 {
				config.Config["regions"] = regions
			}
		}
	}
	
	// Add common configuration
	if config.Config == nil {
		config.Config = make(map[string]interface{})
	}
	config.Config["scan_id"] = fmt.Sprintf("%s-%d", provider, time.Now().Unix())
	
	// Generate snapshot name if not provided
	if snapshotName == "" {
		snapshotName = defaultsManager.GenerateAutoName("scan")
		log("Auto-generated snapshot name: %s\n", snapshotName)
	}
	config.Config["snapshot_name"] = snapshotName
	
	// Validate configuration
	if err := collector.Validate(config); err != nil {
		// Provider-specific error handling
		switch provider {
		case "gcp":
			if projectID, _ := cmd.Flags().GetString("project"); projectID == "" {
				return wgoerrors.GCPProjectError()
			}
			return wgoerrors.GCPAuthenticationError(err)
		case "aws":
			if region, _ := cmd.Flags().GetString("region"); region == "" && os.Getenv("AWS_REGION") == "" {
				return wgoerrors.AWSRegionError()
			}
			return wgoerrors.AWSCredentialsError(err)
		case "kubernetes":
			return wgoerrors.KubernetesConnectionError("", err)
		case "terraform":
			return wgoerrors.TerraformStateError(filepath.Join(statePaths...))
		default:
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}
	
	// Perform collection
	log("Collecting resources from %s...\n", provider)
	startTime := time.Now()
	
	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		// Check for common error patterns
		errStr := err.Error()
		switch provider {
		case "gcp":
			if strings.Contains(errStr, "403") || strings.Contains(errStr, "permission") {
				return wgoerrors.PermissionError(wgoerrors.ProviderGCP, "GCP APIs")
			}
			if strings.Contains(errStr, "could not find default credentials") {
				return wgoerrors.GCPAuthenticationError(err)
			}
			return wgoerrors.New(wgoerrors.ErrorTypeProvider, wgoerrors.ProviderGCP, "Resource collection failed").
				WithCause(err.Error()).
				WithSolutions("Check GCP permissions", "Verify API is enabled").
				WithHelp("wgo help gcp")
		case "aws":
			if strings.Contains(errStr, "UnauthorizedOperation") || strings.Contains(errStr, "AccessDenied") {
				return wgoerrors.PermissionError(wgoerrors.ProviderAWS, "AWS resources")
			}
			if strings.Contains(errStr, "ExpiredToken") {
				return wgoerrors.AWSCredentialsError(err)
			}
			return wgoerrors.New(wgoerrors.ErrorTypeProvider, wgoerrors.ProviderAWS, "Resource collection failed").
				WithCause(err.Error()).
				WithSolutions("Check AWS permissions", "Verify credentials").
				WithHelp("wgo help aws")
		default:
			return fmt.Errorf("collection failed: %w", err)
		}
	}
	
	collectionTime := time.Since(startTime)
	
	// Display results
	if !quiet {
		fmt.Printf("\nCollection completed in %v\n", collectionTime)
		fmt.Printf("Snapshot ID: %s\n", snapshot.ID)
		fmt.Printf("Resources found: %d\n", len(snapshot.Resources))
		
		// Group resources by type
		byType := make(map[string]int)
		for _, resource := range snapshot.Resources {
			byType[resource.Type]++
		}
		
		fmt.Println("\nResource breakdown:")
		for resourceType, count := range byType {
			fmt.Printf("  - %s: %d\n", resourceType, count)
		}
	}
	
	// Save to history directory for time-based comparisons
	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(homeDir, ".wgo", "history")
	os.MkdirAll(historyDir, 0755)
	
	// Create timestamped filename
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	historyPath := filepath.Join(historyDir, fmt.Sprintf("%s-%s-%s.json", timestamp, provider, snapshot.ID))
	if err := saveSnapshotToFile(snapshot, historyPath); err != nil {
		log("Warning: Failed to save to history: %v\n", err)
	}
	
	// Also save as last-scan for quick access
	wgoDir := filepath.Join(homeDir, ".wgo")
	lastScanPath := filepath.Join(wgoDir, fmt.Sprintf("last-scan-%s.json", provider))
	if err := saveSnapshotToFile(snapshot, lastScanPath); err != nil {
		log("Warning: Failed to save automatic snapshot: %v\n", err)
	}
	
	// Save to output file if specified
	if outputFile != "" {
		if err := saveSnapshotToFile(snapshot, outputFile); err != nil {
			log("Warning: Failed to save to output file: %v\n", err)
		} else {
			log("\nOutput saved to: %s\n", outputFile)
		}
	}
	
	log("\nSnapshot saved - use 'wgo diff' to detect changes\n")
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

