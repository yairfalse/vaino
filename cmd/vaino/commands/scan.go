package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/collectors/aws"
	"github.com/yairfalse/vaino/internal/collectors/gcp"
	"github.com/yairfalse/vaino/internal/collectors/kubernetes"
	"github.com/yairfalse/vaino/internal/collectors/terraform"
	"github.com/yairfalse/vaino/internal/discovery"
	vainoerrors "github.com/yairfalse/vaino/internal/errors"
	"github.com/yairfalse/vaino/internal/output"
	"github.com/yairfalse/vaino/pkg/config"
	"github.com/yairfalse/vaino/pkg/types"
)

func newScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan infrastructure for current state",
		SilenceUsage: true,
		Long: `Scan discovers and collects the current state of your infrastructure
from various providers (Terraform, AWS, Kubernetes) and creates a snapshot.

This snapshot can be used as a reference point for future drift detection or 
compared against other snapshots to identify changes.`,
		Example: `  # Scan Terraform state
  vaino scan --provider terraform --path ./terraform

  # Scan AWS resources in multiple regions
  vaino scan --provider aws --region us-east-1,us-west-2

  # Scan Kubernetes cluster
  vaino scan --provider kubernetes --context prod --namespace default,kube-system

  # Scan GCP resources
  vaino scan --provider gcp

  # Scan GCP with specific project and regions  
  vaino scan --provider gcp --project my-project-123 --region us-central1,us-east1

  # Scan GCP with custom credentials
  vaino scan --provider gcp --credentials ./service-account.json

  # Scan all providers and save with custom name
  vaino scan --all --output-file my-snapshot.json
  
  # Create a baseline for production infrastructure
  vaino scan --provider terraform --baseline --baseline-name production
  
  # Create baseline with reason
  vaino scan --provider aws --baseline --baseline-name v2.1 --baseline-reason "Release 2.1 deployment"`,
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

	// Baseline flags (transparent to users)
	cmd.Flags().Bool("baseline", false, "mark this scan as a baseline for future comparisons")
	cmd.Flags().String("baseline-name", "", "name for the baseline (e.g., 'production', 'v1.0')")
	cmd.Flags().String("baseline-reason", "", "reason for creating this baseline")

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

	provider, _ := cmd.Flags().GetString("provider")
	scanAll, _ := cmd.Flags().GetBool("all")
	outputFile, _ := cmd.Flags().GetString("output-file")
	snapshotName, _ := cmd.Flags().GetString("snapshot-name")
	statePaths, _ := cmd.Flags().GetStringSlice("state-file")
	autoDiscover, _ := cmd.Flags().GetBool("auto-discover")

	// Baseline flags
	isBaseline, _ := cmd.Flags().GetBool("baseline")
	baselineName, _ := cmd.Flags().GetString("baseline-name")
	baselineReason, _ := cmd.Flags().GetString("baseline-reason")

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
		_, err := defaultsManager.GenerateSmartDefaults()
		if err != nil {
			return fmt.Errorf("failed to generate smart defaults: %w", err)
		}

		discovery := discovery.NewTerraformDiscovery()
		stateFiles, err := discovery.DiscoverStateFiles("")

		if err == nil && len(stateFiles) > 0 {
			// Found Terraform state files, use terraform provider
			provider = "terraform"
			autoDiscover = true
			log("Found %d Terraform state file(s)\n", len(stateFiles))
		} else {
			// No auto-discovery possible, show helpful guidance
			fmt.Println("\nNo infrastructure auto-detected.")
			fmt.Println("\nChoose your provider:")
			fmt.Println("  vaino scan --provider terraform")
			fmt.Println("  vaino scan --provider aws --region us-east-1")
			fmt.Println("  vaino scan --provider gcp --project YOUR-PROJECT-ID")
			fmt.Println("  vaino scan --provider kubernetes")
			fmt.Println("\nTip: Run 'vaino configure' for setup")
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
		return vainoerrors.New(vainoerrors.ErrorTypeProvider, vainoerrors.Provider(provider),
			fmt.Sprintf("Failed to initialize %s collector", provider)).
			WithCause(err.Error()).
			WithSolutions(
				fmt.Sprintf("Ensure %s provider is properly installed", provider),
				"Run 'vaino check-config' to diagnose issues",
			).
			WithHelp(fmt.Sprintf("vaino help %s", provider))
	}

	// Check collector status
	status := collector.Status()
	if status != "ready" {
		fmt.Printf("Warning: Collector status: %s\n", status)
	}

	// Build collector configuration
	var config collectors.CollectorConfig

	if autoDiscover {
		discoveredConfig, err := collector.AutoDiscover()
		if err != nil {
			return fmt.Errorf("auto-discovery failed: %w", err)
		}
		config = discoveredConfig
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
	}
	config.Config["snapshot_name"] = snapshotName

	// Validate configuration
	if err := collector.Validate(config); err != nil {
		// Provider-specific error handling
		switch provider {
		case "gcp":
			if projectID, _ := cmd.Flags().GetString("project"); projectID == "" {
				return vainoerrors.GCPProjectError()
			}
			return vainoerrors.GCPAuthenticationError(err)
		case "aws":
			if region, _ := cmd.Flags().GetString("region"); region == "" && os.Getenv("AWS_REGION") == "" {
				return vainoerrors.AWSRegionError()
			}
			return vainoerrors.AWSCredentialsError(err)
		case "kubernetes":
			return vainoerrors.KubernetesConnectionError("", err)
		case "terraform":
			return vainoerrors.TerraformStateError(filepath.Join(statePaths...))
		default:
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Perform collection
	if !quiet {
		fmt.Println("Scanning infrastructure...")
	}

	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		// Check for common error patterns
		errStr := err.Error()
		switch provider {
		case "gcp":
			if strings.Contains(errStr, "403") || strings.Contains(errStr, "permission") {
				return vainoerrors.PermissionError(vainoerrors.ProviderGCP, "GCP APIs")
			}
			if strings.Contains(errStr, "could not find default credentials") {
				return vainoerrors.GCPAuthenticationError(err)
			}
			return vainoerrors.New(vainoerrors.ErrorTypeProvider, vainoerrors.ProviderGCP, "Resource collection failed").
				WithCause(err.Error()).
				WithSolutions("Check GCP permissions", "Verify API is enabled").
				WithHelp("vaino help gcp")
		case "aws":
			if strings.Contains(errStr, "UnauthorizedOperation") || strings.Contains(errStr, "AccessDenied") {
				return vainoerrors.PermissionError(vainoerrors.ProviderAWS, "AWS resources")
			}
			if strings.Contains(errStr, "ExpiredToken") {
				return vainoerrors.AWSCredentialsError(err)
			}
			return vainoerrors.New(vainoerrors.ErrorTypeProvider, vainoerrors.ProviderAWS, "Resource collection failed").
				WithCause(err.Error()).
				WithSolutions("Check AWS permissions", "Verify credentials").
				WithHelp("vaino help aws")
		default:
			return fmt.Errorf("collection failed: %w", err)
		}
	}

	// Mark as baseline if requested
	if isBaseline {
		snapshot.MarkAsBaseline(baselineName, baselineReason)
		if !quiet {
			if baselineName != "" {
				fmt.Printf("Marked as baseline: %s\n", baselineName)
			} else {
				fmt.Printf("Marked as baseline\n")
			}
		}
	}

	// Display results using enhanced formatter
	formatter := output.NewScanFormatter(snapshot, quiet)
	fmt.Print(formatter.FormatOutput())

	// Save to history directory for time-based comparisons
	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(homeDir, ".vaino", "history")
	os.MkdirAll(historyDir, 0755)

	// Create timestamped filename
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	historyPath := filepath.Join(historyDir, fmt.Sprintf("%s-%s-%s.json", timestamp, provider, snapshot.ID))
	if err := saveSnapshotToFile(snapshot, historyPath); err != nil {
		log("Warning: Failed to save to history: %v\n", err)
	}

	// Also save as last-scan for quick access
	vainoDir := filepath.Join(homeDir, ".vaino")
	lastScanPath := filepath.Join(vainoDir, fmt.Sprintf("last-scan-%s.json", provider))
	if err := saveSnapshotToFile(snapshot, lastScanPath); err != nil {
		log("Warning: Failed to save automatic snapshot: %v\n", err)
	}

	// Save to output file if specified
	if outputFile != "" {
		if err := saveSnapshotToFile(snapshot, outputFile); err != nil {
			log("Warning: Failed to save to output file: %v\n", err)
		} else {
			log("\nSnapshot saved to: %s\n", outputFile)
		}
	}

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
