package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/pkg/config"
	wgoerrors "github.com/yairfalse/wgo/internal/errors"
	"gopkg.in/yaml.v3"
)

var configureCmd = &cobra.Command{
	Use:   "configure [provider]",
	Short: "Interactive configuration wizard for WGO",
	Long: `Configure WGO providers through an interactive wizard.

This command helps you set up:
- Provider authentication
- Default regions and projects
- Storage locations
- API settings

Examples:
  wgo configure              # Interactive setup for all providers
  wgo configure gcp          # Configure only GCP
  wgo configure aws          # Configure only AWS`,
	RunE: runConfigure,
}

func newConfigureCommand() *cobra.Command {
	return configureCmd
}

func runConfigure(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	
	// Load existing config or create new
	cfg := GetConfig()
	if cfg == nil {
		cfg = &config.Config{
			Providers: config.ProvidersConfig{},
			Storage: config.StorageConfig{
				BaseDir: filepath.Join(os.Getenv("HOME"), ".wgo", "storage"),
			},
		}
	}
	
	fmt.Println("WGO Configuration Wizard")
	fmt.Println("=========================")
	fmt.Println()
	
	// Determine which providers to configure
	var providers []string
	if len(args) > 0 {
		providers = []string{args[0]}
	} else {
		// Auto-detect available providers
		detector := config.NewProviderDetector()
		results := detector.DetectAll()
		
		fmt.Println("Detecting available providers...")
		fmt.Println()
		
		for provider, result := range results {
			if result.Available {
				fmt.Printf("  [OK] %s: %s\n", provider, result.Status)
				providers = append(providers, provider)
			} else {
				fmt.Printf("  [FAIL] %s: %s\n", provider, result.Status)
			}
		}
		
		if len(providers) == 0 {
			return wgoerrors.New(wgoerrors.ErrorTypeConfiguration, wgoerrors.ProviderUnknown,
				"No providers detected").
				WithCause("No cloud provider CLIs found").
				WithSolutions(
					"Install gcloud CLI for GCP",
					"Install AWS CLI for AWS",
					"Install kubectl for Kubernetes",
				).
				WithHelp("wgo help providers")
		}
		
		fmt.Printf("\nConfigure all %d detected providers? [Y/n]: ", len(providers))
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		
		if answer == "n" || answer == "no" {
			// Let user select providers
			selected := []string{}
			for _, provider := range providers {
				fmt.Printf("Configure %s? [Y/n]: ", provider)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "n" && answer != "no" {
					selected = append(selected, provider)
				}
			}
			providers = selected
		}
	}
	
	// Configure each provider
	for _, provider := range providers {
		fmt.Printf("\nConfiguring %s...\n", strings.Title(provider))
		
		switch provider {
		case "gcp":
			if err := configureGCP(cfg, reader); err != nil {
				return err
			}
		case "aws":
			if err := configureAWS(cfg, reader); err != nil {
				return err
			}
		case "kubernetes":
			if err := configureKubernetes(cfg, reader); err != nil {
				return err
			}
		case "terraform":
			if err := configureTerraform(cfg, reader); err != nil {
				return err
			}
		default:
			fmt.Printf("Unknown provider: %s\n", provider)
		}
	}
	
	// Save configuration
	configPath := filepath.Join(os.Getenv("HOME"), ".wgo", "config.yaml")
	if err := saveConfig(cfg, configPath); err != nil {
		return wgoerrors.New(wgoerrors.ErrorTypeFileSystem, wgoerrors.ProviderUnknown,
			"Failed to save configuration").
			WithCause(err.Error()).
			WithSolutions(
				"Check directory permissions",
				"Ensure ~/.wgo directory exists",
				"Try creating manually: mkdir -p ~/.wgo",
			).
			WithHelp("wgo check-config")
	}
	
	fmt.Printf("\nConfiguration saved to %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Run 'wgo check-config' to verify configuration")
	fmt.Println("  2. Run 'wgo scan' to create your first snapshot")
	fmt.Println("  3. Run 'wgo help' for more information")
	
	return nil
}

func configureGCP(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("\nGCP Configuration:")
	fmt.Println("-----------------")
	
	// Check current auth
	authChecker := config.NewAuthChecker()
	gcpAuth := authChecker.CheckGCP()
	
	if gcpAuth.Authenticated {
		fmt.Printf("[OK] Already authenticated (project: %s)\n", gcpAuth.ProjectID)
		cfg.Providers.GCP.Project = gcpAuth.ProjectID
	} else {
		fmt.Println("[WARN] Not authenticated")
		fmt.Println("\nTo authenticate, run:")
		fmt.Println("  gcloud auth application-default login")
		fmt.Println("\nPress Enter to continue...")
		reader.ReadString('\n')
	}
	
	// Get project ID
	fmt.Printf("Default project ID [%s]: ", cfg.Providers.GCP.Project)
	project, _ := reader.ReadString('\n')
	project = strings.TrimSpace(project)
	if project != "" {
		cfg.Providers.GCP.Project = project
	}
	
	// Get regions
	fmt.Print("Default regions (comma-separated) [us-central1]: ")
	regions, _ := reader.ReadString('\n')
	regions = strings.TrimSpace(regions)
	if regions == "" {
		regions = "us-central1"
	}
	cfg.Providers.GCP.Regions = strings.Split(regions, ",")
	
	return nil
}

func configureAWS(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("\nAWS Configuration:")
	fmt.Println("-----------------")
	
	// Check current auth
	authChecker := config.NewAuthChecker()
	awsAuth := authChecker.CheckAWS()
	
	if awsAuth.Authenticated {
		fmt.Printf("[OK] Already authenticated (profile: %s, region: %s)\n", 
			awsAuth.Profile, awsAuth.Region)
		if awsAuth.Region != "" {
			cfg.Providers.AWS.DefaultRegion = awsAuth.Region
		}
	} else {
		fmt.Println("[WARN] Not authenticated")
		fmt.Println("\nTo authenticate, run:")
		fmt.Println("  aws configure")
		fmt.Println("\nPress Enter to continue...")
		reader.ReadString('\n')
	}
	
	// Get default region
	fmt.Printf("Default region [%s]: ", cfg.Providers.AWS.DefaultRegion)
	region, _ := reader.ReadString('\n')
	region = strings.TrimSpace(region)
	if region != "" {
		cfg.Providers.AWS.DefaultRegion = region
	}
	
	// Get profile
	fmt.Print("AWS profile (leave empty for default): ")
	profile, _ := reader.ReadString('\n')
	profile = strings.TrimSpace(profile)
	if profile != "" {
		cfg.Providers.AWS.Profile = profile
	}
	
	return nil
}

func configureKubernetes(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("\nKubernetes Configuration:")
	fmt.Println("------------------------")
	
	// Check current auth
	authChecker := config.NewAuthChecker()
	k8sAuth := authChecker.CheckKubernetes()
	
	if k8sAuth.Authenticated {
		fmt.Printf("[OK] Connected to cluster (context: %s)\n", k8sAuth.Context)
	} else {
		fmt.Println("[WARN] No cluster connection")
		fmt.Println("\nEnsure kubectl is configured with a valid context")
		fmt.Println("\nPress Enter to continue...")
		reader.ReadString('\n')
	}
	
	// Get default namespaces
	fmt.Print("Default namespaces to scan (comma-separated) [default,kube-system]: ")
	namespaces, _ := reader.ReadString('\n')
	namespaces = strings.TrimSpace(namespaces)
	if namespaces == "" {
		namespaces = "default,kube-system"
	}
	cfg.Providers.Kubernetes.Namespaces = strings.Split(namespaces, ",")
	
	return nil
}

func configureTerraform(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("\nTerraform Configuration:")
	fmt.Println("-----------------------")
	
	// Enable auto-discovery
	fmt.Print("Enable auto-discovery of state files? [Y/n]: ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	cfg.Providers.Terraform.AutoDiscover = answer != "n" && answer != "no"
	
	// Get state paths
	fmt.Print("Additional state file paths (comma-separated) [.]: ")
	paths, _ := reader.ReadString('\n')
	paths = strings.TrimSpace(paths)
	if paths == "" {
		paths = "."
	}
	cfg.Providers.Terraform.StatePaths = strings.Split(paths, ",")
	
	return nil
}

func saveConfig(cfg *config.Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Marshal config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	
	// Write file
	return os.WriteFile(path, data, 0644)
}

func createDefaultConfig(configPath string) error {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Terraform: config.TerraformProviderConfig{
				AutoDiscover: true,
				StatePaths: []string{"."},
			},
		},
		Storage: config.StorageConfig{
			BaseDir: filepath.Join(os.Getenv("HOME"), ".wgo", "storage"),
		},
	}
	
	return saveConfig(cfg, configPath)
}