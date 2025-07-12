package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/helpers"
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication for cloud providers",
		Long: `The auth command helps you set up authentication for various cloud providers.
It provides interactive setup and validation of credentials.`,
		Example: `  # Set up GCP authentication
  vaino auth gcp

  # Set up AWS authentication  
  vaino auth aws

  # Test current authentication
  vaino auth test

  # Show authentication status
  vaino auth status`,
	}

	// Add subcommands
	cmd.AddCommand(newAuthGCPCommand())
	cmd.AddCommand(newAuthAWSCommand())
	cmd.AddCommand(newAuthTestCommand())
	cmd.AddCommand(newAuthStatusCommand())

	return cmd
}

func newAuthGCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Set up Google Cloud Platform authentication",
		Long: `Interactive setup for GCP authentication. This command will:
- Check if gcloud is installed
- Set up Application Default Credentials
- Optionally set a default project`,
		Example: `  # Basic GCP auth setup
  vaino auth gcp

  # Set up auth with specific project
  vaino auth gcp --project my-project-123`,
		RunE: runAuthGCP,
	}

	cmd.Flags().String("project", "", "GCP project ID to use as default")
	cmd.Flags().BoolP("quiet", "q", false, "suppress decorative output")

	return cmd
}

func newAuthAWSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Set up AWS authentication",
		Long: `Interactive setup for AWS authentication. This command will:
- Check if AWS CLI is installed
- Run aws configure to set up credentials
- Validate the configuration`,
		Example: `  # Set up AWS auth
  vaino auth aws`,
		RunE: runAuthAWS,
	}

	cmd.Flags().BoolP("quiet", "q", false, "suppress decorative output")

	return cmd
}

func newAuthTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test authentication for all providers",
		Long:  `Tests authentication for all configured cloud providers and shows the status.`,
		Example: `  # Test all authentication
  vaino auth test

  # Test specific provider
  vaino auth test --provider gcp`,
		RunE: runAuthTest,
	}

	cmd.Flags().String("provider", "", "specific provider to test")
	cmd.Flags().BoolP("quiet", "q", false, "suppress decorative output")

	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  `Shows the current authentication status for all providers.`,
		RunE:  runAuthStatus,
	}

	cmd.Flags().BoolP("quiet", "q", false, "suppress decorative output and tips")

	return cmd
}

func runAuthGCP(cmd *cobra.Command, args []string) error {
	projectID, _ := cmd.Flags().GetString("project")

	fmt.Println("🔐 Setting up GCP Authentication")
	fmt.Println("================================")

	authHelper := helpers.NewAuthHelper()
	return authHelper.SetupGCPAuth(projectID)
}

func runAuthAWS(cmd *cobra.Command, args []string) error {
	fmt.Println("🔐 Setting up AWS Authentication")
	fmt.Println("================================")

	authHelper := helpers.NewAuthHelper()
	return authHelper.SetupAWSAuth()
}

func runAuthTest(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("Testing Authentication")
		fmt.Println("========================")
	}

	// TODO: Implement actual authentication testing
	// For now, provide helpful information

	if provider == "" || provider == "gcp" {
		fmt.Println("\nGCP Authentication:")
		testGCPAuth()
	}

	if provider == "" || provider == "aws" {
		fmt.Println("\nAWS Authentication:")
		testAWSAuth()
	}

	if provider == "" || provider == "terraform" {
		fmt.Println("\nTerraform:")
		testTerraformAuth()
	}

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🔐 Authentication Status")
		fmt.Println("=======================")
	}

	// Check GCP
	fmt.Println("\nGoogle Cloud Platform:")
	showGCPAuthStatus()

	// Check AWS
	fmt.Println("\nAWS:")
	showAWSAuthStatus()

	// Check Terraform
	fmt.Println("\n📋 Terraform:")
	showTerraformStatus()

	if !quiet {
		fmt.Println("\nTips:")
		fmt.Println("  • Run 'vaino auth <provider>' to set up authentication")
		fmt.Println("  • Run 'vaino auth test' to verify your credentials work")
	}

	return nil
}

// Helper functions

func testGCPAuth() {
	// Simple checks for now
	if gcloudAccount := getGcloudAccount(); gcloudAccount != "" {
		fmt.Printf("  Logged in as: %s\n", gcloudAccount)
	} else {
		fmt.Println("  Not authenticated")
		fmt.Println("     Run: vaino auth gcp")
	}
}

func testAWSAuth() {
	// Check for AWS credentials
	if awsProfile := getAWSProfile(); awsProfile != "" {
		fmt.Printf("  Using profile: %s\n", awsProfile)
	} else {
		fmt.Println("  No AWS credentials found")
		fmt.Println("     Run: vaino auth aws")
	}
}

func testTerraformAuth() {
	// Just check if terraform is installed
	authHelper := helpers.NewAuthHelper()
	if err := authHelper.CheckTerraformAuth(); err != nil {
		fmt.Println("  Terraform not properly configured")
	} else {
		fmt.Println("  Terraform is available")
	}
}

func showGCPAuthStatus() {
	// Implementation would check various auth methods
	fmt.Println("  • gcloud CLI: " + checkCommandStatus("gcloud"))
	fmt.Println("  • Application Default Credentials: " + checkADCStatus())
	fmt.Println("  • Service Account Key: " + checkGCPKeyStatus())
}

func showAWSAuthStatus() {
	fmt.Println("  • AWS CLI: " + checkCommandStatus("aws"))
	fmt.Println("  • Environment Variables: " + checkAWSEnvStatus())
	fmt.Println("  • Credentials File: " + checkAWSCredsFileStatus())
}

func showTerraformStatus() {
	fmt.Println("  • Terraform CLI: " + checkCommandStatus("terraform"))
	fmt.Println("  • State Files: " + checkTerraformStateStatus())
}

// Utility functions (simplified for now)

func getGcloudAccount() string {
	// Would actually run gcloud auth list
	return ""
}

func getAWSProfile() string {
	// Would check AWS_PROFILE env var
	return ""
}

func checkCommandStatus(cmd string) string {
	// Would check if command exists
	return "❓ Unknown"
}

func checkADCStatus() string {
	return "❓ Unknown"
}

func checkGCPKeyStatus() string {
	return "❓ Unknown"
}

func checkAWSEnvStatus() string {
	return "❓ Unknown"
}

func checkAWSCredsFileStatus() string {
	return "❓ Unknown"
}

func checkTerraformStateStatus() string {
	return "❓ Unknown"
}
