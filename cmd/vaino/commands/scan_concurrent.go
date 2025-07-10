package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/collectors/aws"
	"github.com/yairfalse/vaino/internal/collectors/gcp"
	"github.com/yairfalse/vaino/internal/collectors/kubernetes"
	"github.com/yairfalse/vaino/internal/collectors/terraform"
	"github.com/yairfalse/vaino/internal/scanner"
)

// addConcurrentScanFlags adds flags specific to concurrent scanning
func addConcurrentScanFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("concurrent", false, "enable concurrent provider scanning for massive speed improvements")
	cmd.Flags().Int("max-workers", 4, "maximum number of concurrent workers")
	cmd.Flags().Duration("scan-timeout", 5*time.Minute, "timeout for individual provider scans")
	cmd.Flags().Bool("fail-on-error", false, "fail entire scan if any provider fails")
	cmd.Flags().Bool("skip-merging", false, "skip merging snapshots from multiple providers")
	cmd.Flags().StringSlice("preferred-order", []string{}, "preferred order for provider scanning")
}

// runConcurrentScan executes concurrent multi-provider scanning
func runConcurrentScan(cmd *cobra.Command, args []string) error {
	// Get concurrent scan flags
	concurrent, _ := cmd.Flags().GetBool("concurrent")
	maxWorkers, _ := cmd.Flags().GetInt("max-workers")
	scanTimeout, _ := cmd.Flags().GetDuration("scan-timeout")
	failOnError, _ := cmd.Flags().GetBool("fail-on-error")
	skipMerging, _ := cmd.Flags().GetBool("skip-merging")
	preferredOrder, _ := cmd.Flags().GetStringSlice("preferred-order")
	quiet, _ := cmd.Flags().GetBool("quiet")
	scanAll, _ := cmd.Flags().GetBool("all")

	// Helper to print only when not quiet
	log := func(format string, args ...interface{}) {
		if !quiet {
			fmt.Printf(format, args...)
		}
	}

	if !concurrent && !scanAll {
		return fmt.Errorf("concurrent scanning requires --concurrent flag or --all flag")
	}

	if !quiet {
		fmt.Println("Concurrent Infrastructure Scan")
		fmt.Println("==============================")
	}

	// Create concurrent scanner
	concurrentScanner := scanner.NewConcurrentScanner(maxWorkers, scanTimeout)
	defer concurrentScanner.Close()

	// Register available providers
	providers := map[string]collectors.EnhancedCollector{
		"terraform":  terraform.NewTerraformCollector(),
		"aws":        aws.NewConcurrentAWSCollector(maxWorkers, scanTimeout),
		"gcp":        gcp.NewConcurrentGCPCollector(maxWorkers, scanTimeout),
		"kubernetes": kubernetes.NewConcurrentKubernetesCollector(maxWorkers, scanTimeout),
	}

	// Register all providers with the scanner
	for name, collector := range providers {
		concurrentScanner.RegisterProvider(name, collector)
	}

	// Build provider configurations
	providerConfigs := make(map[string]collectors.CollectorConfig)

	// Check which providers are available and configured
	availableProviders := []string{}

	for providerName, collector := range providers {
		config := buildProviderConfig(cmd, providerName)

		// Validate provider configuration
		if err := collector.Validate(config); err != nil {
			log("Warning: Provider %s not available: %v\n", providerName, err)
			continue
		}

		providerConfigs[providerName] = config
		availableProviders = append(availableProviders, providerName)
	}

	if len(availableProviders) == 0 {
		return fmt.Errorf("no providers are available and configured")
	}

	log("Available providers: %v\n", availableProviders)
	log("Scanning with %d concurrent workers...\n", maxWorkers)

	// Create scan configuration
	scanConfig := scanner.ScanConfig{
		Providers:      providerConfigs,
		MaxWorkers:     maxWorkers,
		Timeout:        scanTimeout,
		FailOnError:    failOnError,
		SkipMerging:    skipMerging,
		PreferredOrder: preferredOrder,
	}

	// Perform concurrent scan
	ctx := context.Background()
	startTime := time.Now()

	result, err := concurrentScanner.ScanAllProviders(ctx, scanConfig)
	if err != nil {
		return fmt.Errorf("concurrent scan failed: %w", err)
	}

	scanDuration := time.Since(startTime)

	// Display results
	if !quiet {
		fmt.Printf("\nConcurrent scan completed in %v\n", scanDuration)
		fmt.Printf("Providers scanned: %d\n", len(result.ProviderResults))
		fmt.Printf("Successful scans: %d\n", result.SuccessCount)
		fmt.Printf("Failed scans: %d\n", result.ErrorCount)

		// Display individual provider results
		fmt.Println("\nProvider Results:")
		for providerName, providerResult := range result.ProviderResults {
			if providerResult.Error != nil {
				fmt.Printf("  ❌ %s: %v (took %v)\n", providerName, providerResult.Error, providerResult.Duration)
			} else {
				fmt.Printf("  ✅ %s: %d resources (took %v)\n",
					providerName, len(providerResult.Snapshot.Resources), providerResult.Duration)
			}
		}

		// Display merged snapshot info
		if result.Snapshot != nil {
			fmt.Printf("\nMerged Snapshot:\n")
			fmt.Printf("  Total resources: %d\n", len(result.Snapshot.Resources))
			fmt.Printf("  Snapshot ID: %s\n", result.Snapshot.ID)

			// Group resources by type
			byType := make(map[string]int)
			for _, resource := range result.Snapshot.Resources {
				byType[resource.Type]++
			}

			fmt.Println("\nResource breakdown:")
			for resourceType, count := range byType {
				fmt.Printf("  - %s: %d\n", resourceType, count)
			}
		}
	}

	// Save results
	if result.Snapshot != nil {
		outputFile, _ := cmd.Flags().GetString("output-file")
		if outputFile != "" {
			if err := saveSnapshotToFile(result.Snapshot, outputFile); err != nil {
				log("Warning: Failed to save to output file: %v\n", err)
			} else {
				log("\nOutput saved to: %s\n", outputFile)
			}
		}
	}

	// Performance summary
	if !quiet {
		fmt.Printf("\n📊 Performance Summary:\n")
		fmt.Printf("  Total scan time: %v\n", scanDuration)
		fmt.Printf("  Average provider time: %v\n", scanDuration/time.Duration(len(availableProviders)))
		fmt.Printf("  Concurrent efficiency: %.1fx faster than sequential\n",
			float64(result.TotalDuration)/float64(scanDuration))
	}

	return nil
}

// buildProviderConfig builds configuration for a specific provider
func buildProviderConfig(cmd *cobra.Command, providerName string) collectors.CollectorConfig {
	config := collectors.CollectorConfig{
		Config: make(map[string]interface{}),
	}

	switch providerName {
	case "terraform":
		statePaths, _ := cmd.Flags().GetStringSlice("state-file")
		path, _ := cmd.Flags().GetString("path")
		autoDiscover, _ := cmd.Flags().GetBool("auto-discover")

		if len(statePaths) > 0 {
			config.StatePaths = statePaths
		} else if path != "" {
			config.StatePaths = []string{path}
		}

		config.Config["auto_discover"] = autoDiscover

	case "aws":
		profile, _ := cmd.Flags().GetString("profile")
		regions, _ := cmd.Flags().GetStringSlice("region")

		if profile != "" {
			config.Config["profile"] = profile
		}
		if len(regions) > 0 && regions[0] != "" {
			config.Config["region"] = regions[0]
		}

	case "gcp":
		projectID, _ := cmd.Flags().GetString("project")
		credentialsFile, _ := cmd.Flags().GetString("credentials")
		regions, _ := cmd.Flags().GetStringSlice("region")

		if projectID != "" {
			config.Config["project_id"] = projectID
		}
		if credentialsFile != "" {
			config.Config["credentials_file"] = credentialsFile
		}
		if len(regions) > 0 {
			config.Config["regions"] = regions
		}

	case "kubernetes":
		contexts, _ := cmd.Flags().GetStringSlice("context")
		namespaces, _ := cmd.Flags().GetStringSlice("namespace")

		config.Namespaces = namespaces
		if len(contexts) > 0 {
			config.Config["contexts"] = contexts
		}
	}

	return config
}

// enhancedScanCommand creates an enhanced scan command with concurrent support
func enhancedScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan infrastructure for current state with concurrent support",
		Long: `Scan discovers and collects the current state of your infrastructure
from various providers (Terraform, AWS, GCP, Kubernetes) and creates a snapshot.

The scan command supports both sequential and concurrent execution modes:
- Sequential: Scans one provider at a time (default)
- Concurrent: Scans all providers simultaneously for massive speed improvements

Concurrent mode can provide 3-10x performance improvements by parallelizing:
- Multi-provider scanning
- API call parallelization within each provider
- Connection pooling and reuse
- Optimized resource merging and deduplication`,
		Example: `  # Sequential scan (default)
  vaino scan --provider terraform --path ./terraform

  # Concurrent scan of all providers
  vaino scan --all --concurrent --max-workers 8

  # Concurrent scan with specific providers
  vaino scan --concurrent --max-workers 4 \
    --provider aws --region us-east-1 \
    --provider gcp --project my-project \
    --provider kubernetes --namespace default

  # Concurrent scan with performance tuning
  vaino scan --all --concurrent \
    --max-workers 8 \
    --scan-timeout 10m \
    --preferred-order kubernetes,aws,gcp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			concurrent, _ := cmd.Flags().GetBool("concurrent")
			scanAll, _ := cmd.Flags().GetBool("all")

			if concurrent || scanAll {
				return runConcurrentScan(cmd, args)
			}

			// Fall back to original scan logic
			return runScan(cmd, args)
		},
	}

	// Add original scan flags
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

	// Add concurrent scan flags
	addConcurrentScanFlags(cmd)

	return cmd
}
