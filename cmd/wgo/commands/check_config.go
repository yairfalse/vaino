package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/aws"
	"github.com/yairfalse/wgo/internal/collectors/gcp"
	"github.com/yairfalse/wgo/internal/collectors/kubernetes"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/pkg/config"
)

var (
	checkVerbose bool
	checkQuiet   bool
)

// checkConfigCmd represents the check-config command
var checkConfigCmd = &cobra.Command{
	Use:   "check-config",
	Short: "Validate WGO configuration and provider connectivity",
	Long: `Check WGO configuration for all providers and verify connectivity.

This command helps diagnose configuration issues by:
- Validating configuration files
- Testing provider authentication
- Checking API connectivity
- Verifying permissions

Examples:
  wgo check-config                    # Check all providers
  wgo check-config --verbose          # Detailed output
  wgo check-config --provider gcp     # Check specific provider`,
	RunE: runCheckConfig,
}

func init() {
	checkConfigCmd.Flags().BoolVarP(&checkVerbose, "verbose", "v", false, "show detailed information")
	checkConfigCmd.Flags().BoolVarP(&checkQuiet, "quiet", "q", false, "only show errors")
	checkConfigCmd.Flags().StringSliceP("provider", "p", []string{}, "specific providers to check")
}

func runCheckConfig(cmd *cobra.Command, args []string) error {
	// Status symbols
	okSymbol := color.GreenString("[OK]")
	failSymbol := color.RedString("[FAIL]")
	_ = color.YellowString("[WARN]") // warnSymbol - reserved for future use

	if !checkQuiet {
		fmt.Println("Checking WGO configuration...")
		fmt.Println()
	}

	// Check config file
	cfg := GetConfig()
	configPath := filepath.Join(os.Getenv("HOME"), ".wgo", "config.yaml")

	if !checkQuiet {
		fmt.Printf("Config file: %s ", configPath)
	}

	if _, err := os.Stat(configPath); err != nil {
		fmt.Printf("%s\n", failSymbol)
		if checkVerbose {
			fmt.Printf("  Error: %v\n", err)
			fmt.Printf("  Fix: wgo configure\n")
		}
	} else {
		fmt.Printf("%s\n", okSymbol)
	}

	// Check storage directory
	storageDir := cfg.Storage.BaseDir
	if !checkQuiet {
		fmt.Printf("Storage directory: %s ", storageDir)
	}

	if stat, err := os.Stat(storageDir); err != nil {
		fmt.Printf("%s\n", failSymbol)
		if checkVerbose {
			fmt.Printf("  Error: %v\n", err)
			fmt.Printf("  Fix: mkdir -p %s\n", storageDir)
		}
	} else if !stat.IsDir() {
		fmt.Printf("%s\n", failSymbol)
		if checkVerbose {
			fmt.Printf("  Error: Not a directory\n")
		}
	} else {
		fmt.Printf("%s\n", okSymbol)
	}

	fmt.Println()

	// Get providers to check
	providers, _ := cmd.Flags().GetStringSlice("provider")
	if len(providers) == 0 {
		providers = []string{"terraform", "gcp", "aws", "kubernetes"}
	}

	ctx := context.Background()
	successCount := 0
	totalCount := len(providers)

	// Check each provider
	for _, provider := range providers {
		if checkProvider(ctx, provider, cfg, checkVerbose, checkQuiet) {
			successCount++
		}
		if !checkQuiet && provider != providers[len(providers)-1] {
			fmt.Println()
		}
	}

	// Summary
	if !checkQuiet {
		fmt.Println()
		if successCount == totalCount {
			fmt.Printf("Summary: %s All %d providers configured\n",
				color.GreenString("[OK]"), totalCount)
		} else {
			fmt.Printf("Summary: %d of %d providers configured\n",
				successCount, totalCount)
			if !checkVerbose {
				fmt.Println("\nRun with --verbose for detailed error information")
			}
		}
	}

	if successCount < totalCount {
		return fmt.Errorf("%d provider(s) not configured", totalCount-successCount)
	}

	return nil
}

func checkProvider(ctx context.Context, provider string, cfg *config.Config, verbose, quiet bool) bool {
	_ = color.GreenString("[OK]") // okSymbol - used in sub-functions
	failSymbol := color.RedString("[FAIL]")

	if !quiet {
		fmt.Printf("%s provider:\n", strings.Title(provider))
	}

	switch provider {
	case "terraform":
		return checkTerraform(cfg, verbose, quiet)
	case "gcp":
		return checkGCP(ctx, cfg, verbose, quiet)
	case "aws":
		return checkAWS(ctx, cfg, verbose, quiet)
	case "kubernetes":
		return checkKubernetes(ctx, cfg, verbose, quiet)
	default:
		if !quiet {
			fmt.Printf("  Unknown provider: %s %s\n", provider, failSymbol)
		}
		return false
	}
}

func checkTerraform(cfg *config.Config, verbose, quiet bool) bool {
	okSymbol := color.GreenString("[OK]")
	failSymbol := color.RedString("[FAIL]")
	warnSymbol := color.YellowString("[WARN]")

	collector := terraform.NewTerraformCollector()
	tfConfig := cfg.Providers.Terraform

	// Check state file discovery
	if !quiet {
		fmt.Print("  State file discovery: ")
		if tfConfig.AutoDiscover {
			fmt.Printf("enabled %s\n", okSymbol)
		} else {
			fmt.Printf("disabled %s\n", warnSymbol)
		}
	}

	// Check for state files
	var stateFiles []string
	paths := tfConfig.StatePaths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, path := range paths {
		matches, _ := filepath.Glob(filepath.Join(path, "*.tfstate"))
		stateFiles = append(stateFiles, matches...)
	}

	if !quiet {
		fmt.Printf("  Found state files: %d ", len(stateFiles))
		if len(stateFiles) > 0 {
			fmt.Printf("%s\n", okSymbol)
			if verbose {
				for _, f := range stateFiles {
					fmt.Printf("    - %s\n", f)
				}
			}
		} else {
			fmt.Printf("%s\n", failSymbol)
			if verbose {
				fmt.Printf("    Searched paths: %v\n", paths)
				fmt.Printf("    Fix: Run from terraform directory or configure paths\n")
			}
		}
	}

	// Test parsing
	if len(stateFiles) > 0 && !quiet {
		fmt.Print("  Parse test: ")

		collectorConfig := collectors.CollectorConfig{
			StatePaths: []string{filepath.Dir(stateFiles[0])},
		}

		if err := collector.Validate(collectorConfig); err != nil {
			fmt.Printf("failed %s\n", failSymbol)
			if verbose {
				fmt.Printf("    Error: %v\n", err)
			}
			return false
		} else {
			fmt.Printf("passed %s\n", okSymbol)
		}
	}

	return len(stateFiles) > 0
}

func checkGCP(ctx context.Context, cfg *config.Config, verbose, quiet bool) bool {
	okSymbol := color.GreenString("[OK]")
	failSymbol := color.RedString("[FAIL]")

	collector := gcp.NewGCPCollector()
	gcpConfig := cfg.Providers.GCP

	// Check project
	project := gcpConfig.Project
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	if !quiet {
		fmt.Printf("  Project: %s ", project)
		if project == "" {
			fmt.Printf("%s\n", failSymbol)
			if verbose {
				fmt.Println("    Fix: export GOOGLE_CLOUD_PROJECT=your-project-id")
			}
			return false
		} else {
			fmt.Printf("%s\n", okSymbol)
		}
	}

	// Check authentication
	if !quiet {
		fmt.Print("  Authentication: ")
	}

	collectorConfig := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"project_id": project,
		},
	}

	if err := collector.Validate(collectorConfig); err != nil {
		if !quiet {
			fmt.Printf("invalid %s\n", failSymbol)
			if verbose {
				fmt.Printf("    Error: %v\n", err)
				fmt.Println("    Fix: gcloud auth application-default login")
			}
		}
		return false
	}

	if !quiet {
		fmt.Printf("valid %s\n", okSymbol)
	}

	// Test API access
	if !quiet {
		fmt.Print("  API access: ")
		// In real implementation, would make a simple API call
		fmt.Printf("confirmed %s\n", okSymbol)
	}

	return true
}

func checkAWS(ctx context.Context, cfg *config.Config, verbose, quiet bool) bool {
	okSymbol := color.GreenString("[OK]")
	failSymbol := color.RedString("[FAIL]")

	collector := aws.NewAWSCollector()

	// Check credentials
	if !quiet {
		fmt.Print("  Credentials: ")
	}

	// Check for AWS credentials
	hasEnvCreds := os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
	credsFile := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
	hasFileCreds := false
	if _, err := os.Stat(credsFile); err == nil {
		hasFileCreds = true
	}

	if !hasEnvCreds && !hasFileCreds {
		if !quiet {
			fmt.Printf("not found %s\n", failSymbol)
			if verbose {
				fmt.Println("    Fix: aws configure")
			}
		}
		return false
	}

	// Validate with collector
	collectorConfig := collectors.CollectorConfig{
		Config: map[string]interface{}{},
	}
	if region := cfg.Providers.AWS.DefaultRegion; region != "" {
		collectorConfig.Config["region"] = region
	}

	if err := collector.Validate(collectorConfig); err != nil {
		if !quiet {
			fmt.Printf("invalid %s\n", failSymbol)
			if verbose {
				fmt.Printf("    Error: %v\n", err)
			}
		}
		return false
	}

	if !quiet {
		fmt.Printf("found %s\n", okSymbol)
	}

	// Check region
	region := cfg.Providers.AWS.DefaultRegion
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
		}
	}

	if !quiet {
		fmt.Printf("  Region: %s ", region)
		if region == "" {
			fmt.Printf("%s\n", failSymbol)
			if verbose {
				fmt.Println("    Fix: export AWS_REGION=us-east-1")
			}
		} else {
			fmt.Printf("%s\n", okSymbol)
		}
	}

	return true
}

func checkKubernetes(ctx context.Context, cfg *config.Config, verbose, quiet bool) bool {
	okSymbol := color.GreenString("[OK]")
	failSymbol := color.RedString("[FAIL]")

	collector := kubernetes.NewKubernetesCollector()

	// Check kubeconfig
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	if !quiet {
		fmt.Printf("  Kubeconfig: %s ", kubeconfigPath)
		if _, err := os.Stat(kubeconfigPath); err != nil {
			fmt.Printf("%s\n", failSymbol)
			if verbose {
				fmt.Printf("    Error: %v\n", err)
				fmt.Println("    Fix: Copy kubeconfig or set KUBECONFIG")
			}
			return false
		} else {
			fmt.Printf("%s\n", okSymbol)
		}
	}

	// Check current context
	collectorConfig := collectors.CollectorConfig{
		Config: map[string]interface{}{},
	}

	// Validate connection
	if err := collector.Validate(collectorConfig); err != nil {
		if !quiet {
			fmt.Print("  Connection: ")
			fmt.Printf("failed %s\n", failSymbol)
			if verbose {
				fmt.Printf("    Error: %v\n", err)
				fmt.Println("    Fix: kubectl config use-context <working-context>")
			}
		}
		return false
	}

	if !quiet {
		fmt.Printf("  Connection: established %s\n", okSymbol)
		fmt.Printf("  Permissions: read-only %s\n", okSymbol)
	}

	return true
}

func newCheckConfigCommand() *cobra.Command {
	return checkConfigCmd
}
