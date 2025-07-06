package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/config"
)

// newSetupCommand creates the setup command for initial configuration
func (a *App) newSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Initialize WGO configuration for your infrastructure",
		Long: `Setup helps you configure WGO for your specific infrastructure.
It will detect your Terraform state files, AWS configuration, and Git repository
to create an optimized configuration.`,
		Example: `  # Interactive setup
  wgo setup

  # Generate default config only
  wgo setup --config-only
  
  # Setup for specific providers
  wgo setup --providers terraform,aws`,
		RunE: a.runSetupCommand,
	}

	// Flags
	cmd.Flags().Bool("config-only", false, "only generate config file, don't auto-detect")
	cmd.Flags().StringSlice("providers", []string{}, "specific providers to configure (terraform, aws, kubernetes)")
	cmd.Flags().Bool("force", false, "overwrite existing configuration")
	cmd.Flags().String("config-path", "", "custom config file path")

	return cmd
}

func (a *App) runSetupCommand(cmd *cobra.Command, args []string) error {
	configOnly, _ := cmd.Flags().GetBool("config-only")
	providers, _ := cmd.Flags().GetStringSlice("providers")
	force, _ := cmd.Flags().GetBool("force")
	configPath, _ := cmd.Flags().GetString("config-path")

	a.logger.Info("Starting WGO setup...")

	// Check if config already exists
	if configPath == "" {
		configPath = filepath.Join(os.Getenv("HOME"), ".wgo", "config.yaml")
	}

	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Printf("⚠️  Configuration file already exists at %s\n", configPath)
		fmt.Println("Use --force to overwrite, or edit the existing file.")
		return nil
	}

	fmt.Println("🚀 Setting up WGO for your infrastructure...")
	fmt.Println()

	if configOnly {
		return a.generateDefaultConfig(configPath)
	}

	// Auto-detect infrastructure
	return a.detectAndConfigure(configPath, providers)
}

func (a *App) generateDefaultConfig(configPath string) error {
	if err := config.InitConfigFile(); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Default configuration created!")
	fmt.Printf("📄 Config file: %s\n", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the config file to match your infrastructure")
	fmt.Println("  2. Run 'wgo status' to test your configuration")
	fmt.Println("  3. Run 'wgo scan' to create your first snapshot")

	return nil
}

func (a *App) detectAndConfigure(configPath string, providers []string) error {
	fmt.Println("🔍 Auto-detecting your infrastructure...")

	// Detect Git repository
	gitDetected := a.detectGit()
	
	// Detect Terraform
	terraformPaths := a.detectTerraform()
	
	// Detect AWS configuration
	awsConfigured := a.detectAWS()
	
	// Detect Kubernetes
	k8sContexts := a.detectKubernetes()

	// Show detection results
	fmt.Println()
	fmt.Println("📋 Detection Results:")
	fmt.Printf("  Git repository: %s\n", boolToStatus(gitDetected))
	fmt.Printf("  Terraform states: %d found\n", len(terraformPaths))
	fmt.Printf("  AWS configuration: %s\n", boolToStatus(awsConfigured))
	fmt.Printf("  Kubernetes contexts: %d found\n", len(k8sContexts))
	fmt.Println()

	// Generate optimized configuration
	return a.generateOptimizedConfig(configPath, terraformPaths, awsConfigured, k8sContexts)
}

func (a *App) detectGit() bool {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return false
	}
	fmt.Println("  ✅ Git repository detected")
	return true
}

func (a *App) detectTerraform() []string {
	var paths []string
	
	// Common Terraform state file locations
	candidates := []string{
		"terraform.tfstate",
		"terraform/terraform.tfstate", 
		".terraform/terraform.tfstate",
		"infra/terraform.tfstate",
		"infrastructure/terraform.tfstate",
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			paths = append(paths, candidate)
			fmt.Printf("  ✅ Found Terraform state: %s\n", candidate)
		}
	}
	
	// Look for .tf files
	if matches, err := filepath.Glob("*.tf"); err == nil && len(matches) > 0 {
		fmt.Printf("  ✅ Found %d Terraform files in current directory\n", len(matches))
		if !contains(paths, ".") {
			paths = append(paths, ".")
		}
	}
	
	return paths
}

func (a *App) detectAWS() bool {
	// Check for AWS credentials
	homeDir, _ := os.UserHomeDir()
	awsDir := filepath.Join(homeDir, ".aws")
	
	if _, err := os.Stat(awsDir); err == nil {
		fmt.Println("  ✅ AWS configuration detected")
		return true
	}
	
	// Check for AWS environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		fmt.Println("  ✅ AWS environment variables detected")
		return true
	}
	
	return false
}

func (a *App) detectKubernetes() []string {
	var contexts []string
	
	// Check for kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(homeDir, ".kube", "config")
	}
	
	if _, err := os.Stat(kubeconfig); err == nil {
		fmt.Println("  ✅ Kubernetes configuration detected")
		// In a real implementation, you'd parse the kubeconfig to get contexts
		contexts = append(contexts, "default")
	}
	
	return contexts
}

func (a *App) generateOptimizedConfig(configPath string, terraformPaths []string, awsEnabled bool, k8sContexts []string) error {
	configContent := fmt.Sprintf(`# WGO Configuration
# Auto-generated based on detected infrastructure

# Basic settings
verbose: false
debug: false

# Provider configurations
providers:
  terraform:
    enabled: %t
    state_paths:%s
    auto_discover: true
    
  aws:
    enabled: %t
    regions:
      - "us-east-1"
      - "us-west-2"
    
  kubernetes:
    enabled: %t
    contexts:%s

# Git integration
git:
  enabled: true
  track_commits: true
  baseline_on_tag: true
  auto_baseline: false

# Output preferences
output:
  default_format: "table"
  color: true
  timestamps: true
`,
		len(terraformPaths) > 0,
		formatStringSlice(terraformPaths, 6),
		awsEnabled,
		len(k8sContexts) > 0,
		formatStringSlice(k8sContexts, 6),
	)

	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ Optimized configuration created at %s\n", configPath)
	fmt.Println()
	fmt.Println("🎉 WGO is now configured for your infrastructure!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'wgo status' to see your infrastructure overview")
	fmt.Println("  2. Run 'wgo scan' to create your first snapshot")
	if len(terraformPaths) > 0 {
		fmt.Println("  3. Run 'wgo scan --provider terraform' to scan Terraform state")
	}
	if awsEnabled {
		fmt.Println("  4. Set up AWS credentials to enable AWS scanning")
	}

	return nil
}

// Helper functions
func boolToStatus(b bool) string {
	if b {
		return "✅ detected"
	}
	return "❌ not found"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatStringSlice(slice []string, indent int) string {
	if len(slice) == 0 {
		return ""
	}
	
	result := "\n"
	spaces := ""
	for i := 0; i < indent; i++ {
		spaces += " "
	}
	
	for _, item := range slice {
		result += fmt.Sprintf("%s- \"%s\"\n", spaces, item)
	}
	
	return result
}