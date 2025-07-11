package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Provider help documentation
var providerHelp = map[string]string{
	"gcp": `
Google Cloud Platform (GCP) Provider Help
=========================================

Authentication Methods:
1. Application Default Credentials (Recommended)
   gcloud auth application-default login

2. Service Account Key
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"

3. gcloud CLI Authentication
   gcloud auth login

Required APIs:
- Cloud Resource Manager API
- Compute Engine API
- Cloud Storage API
- IAM API

Required Permissions:
- resourcemanager.projects.get
- compute.instances.list
- storage.buckets.list
- iam.serviceAccounts.list

Common Issues:
- "Permission denied": Check IAM roles
- "API not enabled": Enable required APIs in Console
- "Quota exceeded": Request quota increase

Examples:
  vaino scan --provider gcp --project my-project
  vaino scan --provider gcp --region us-central1
  vaino scan --provider gcp --credentials ./sa-key.json
`,

	"aws": `
Amazon Web Services (AWS) Provider Help
======================================

Authentication Methods:
1. Environment Variables
   export AWS_ACCESS_KEY_ID=your-key
   export AWS_SECRET_ACCESS_KEY=your-secret
   export AWS_REGION=us-east-1

2. AWS CLI Configuration
   aws configure

3. IAM Roles (EC2/ECS/Lambda)
   Automatic when running on AWS infrastructure

4. AWS SSO
   aws sso login

Required Permissions:
- ec2:Describe*
- s3:List*
- iam:List*
- rds:Describe*

Common Issues:
- "UnauthorizedOperation": Missing IAM permissions
- "ExpiredToken": Refresh credentials
- "InvalidClientTokenId": Check access keys

Examples:
  vaino scan --provider aws --region us-east-1
  vaino scan --provider aws --profile production
`,

	"kubernetes": `
Kubernetes Provider Help
========================

Authentication Methods:
1. Kubeconfig File (Default)
   Default: ~/.kube/config
   Override: export KUBECONFIG=/path/to/config

2. In-Cluster Authentication
   Automatic when running inside a pod

Required Permissions:
- get, list on all resource types
- No write permissions needed

Common Issues:
- "connection refused": Check cluster connectivity
- "Unauthorized": Check RBAC permissions
- "context not found": Verify kubeconfig

Examples:
  vaino scan --provider kubernetes
  vaino scan --provider kubernetes --context prod
  vaino scan --provider kubernetes --namespace default
`,

	"terraform": `
Terraform Provider Help
=======================

Setup:
1. Ensure state files are accessible
2. Support for local and remote state

Supported State Backends:
- Local file (terraform.tfstate)
- S3 backend
- GCS backend
- Azure backend

Common Issues:
- "state locked": Another process using state
- "version mismatch": Terraform version compatibility
- "corrupted state": Restore from backup

Examples:
  vaino scan --provider terraform
  vaino scan --provider terraform --path ./environments/prod
  vaino scan --provider terraform --auto-discover
`,
}

var troubleshootingHelp = `
VAINO Troubleshooting Guide
========================

Quick Diagnostics:
  vaino check-config              # Validate all configuration
  vaino check-config --verbose    # Detailed diagnostics
  vaino status                    # System and provider status

Common Issues:

1. "No providers configured"
   Solution: Run 'vaino configure' or set up manually

2. "Permission denied"
   - Check provider authentication
   - Verify IAM/RBAC permissions
   - Run 'vaino check-config --verbose'

3. "No changes detected" when changes exist
   - Check scan filters (region, namespace)
   - Verify resource permissions
   - Try --verbose flag

4. Network/Connectivity Issues
   - Check internet connectivity
   - Verify firewall rules
   - Check proxy settings: echo $HTTP_PROXY
   - Try VPN if accessing private resources

5. Performance Issues
   - Use filters to limit scope
   - Check available disk space
   - Consider --no-cache flag

Debug Mode:
  vaino scan --debug              # Enable debug logging
  vaino diff --verbose            # Detailed diff output

Getting Help:
  vaino help providers            # Provider-specific help
  vaino help troubleshooting      # This guide
  vaino <command> --help          # Command-specific help

Report Issues:
  https://github.com/yairfalse/vaino/issues
`

// newHelpCommand creates the help command
func newHelpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "help [topic]",
		Short: "Get help on specific topics",
		Long: `Get detailed help on VAINO topics including provider setup,
troubleshooting, and best practices.`,
		Example: `  vaino help providers          # List all provider guides
  vaino help gcp                 # GCP-specific help
  vaino help aws                 # AWS-specific help
  vaino help troubleshooting     # Troubleshooting guide`,
		RunE: runHelp,
	}

	return cmd
}

func runHelp(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show available help topics
		fmt.Println("Available help topics:")
		fmt.Println()
		fmt.Println("  providers         - List all provider setup guides")
		fmt.Println("  gcp              - Google Cloud Platform guide")
		fmt.Println("  aws              - Amazon Web Services guide")
		fmt.Println("  kubernetes       - Kubernetes guide")
		fmt.Println("  terraform        - Terraform guide")
		fmt.Println("  troubleshooting  - Common issues and solutions")
		fmt.Println()
		fmt.Println("Usage: vaino help <topic>")
		return nil
	}

	topic := strings.ToLower(args[0])

	switch topic {
	case "providers":
		fmt.Println()
		fmt.Println("VAINO Provider Setup Guides")
		fmt.Println("========================")
		fmt.Println()
		fmt.Println("Available providers:")
		fmt.Println("  - gcp         (Google Cloud Platform)")
		fmt.Println("  - aws         (Amazon Web Services)")
		fmt.Println("  - kubernetes  (Kubernetes clusters)")
		fmt.Println("  - terraform   (Terraform state)")
		fmt.Println()
		fmt.Print("Get specific help: vaino help <provider>")

	case "gcp", "google", "gcloud":
		fmt.Print(providerHelp["gcp"])

	case "aws", "amazon":
		fmt.Print(providerHelp["aws"])

	case "kubernetes", "k8s", "kubectl":
		fmt.Print(providerHelp["kubernetes"])

	case "terraform", "tf":
		fmt.Print(providerHelp["terraform"])

	case "troubleshooting", "troubleshoot", "debug":
		fmt.Print(troubleshootingHelp)

	default:
		fmt.Printf("Unknown help topic: %s\n\n", topic)
		fmt.Println("Run 'vaino help' to see available topics")
		return fmt.Errorf("unknown help topic")
	}

	return nil
}
