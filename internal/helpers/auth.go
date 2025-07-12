package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AuthHelper provides authentication help and auto-fix capabilities
type AuthHelper struct{}

// NewAuthHelper creates a new auth helper
func NewAuthHelper() *AuthHelper {
	return &AuthHelper{}
}

// HandleGCPAuthError provides helpful error messages and fixes for GCP auth issues
func (ah *AuthHelper) HandleGCPAuthError(projectID string, originalErr error) error {
	fmt.Println("\nFAILED: GCP Authentication Failed")
	fmt.Println("=====================================")

	// Check what's available
	hasGcloud := ah.isCommandAvailable("gcloud")
	hasADC := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != ""

	if hasGcloud {
		// Check if user is logged in
		cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			account := strings.TrimSpace(string(output))
			fmt.Println("\nSUCCESS: Good news: You have gcloud installed and are logged in!")
			fmt.Printf("   Account: %s\n", account)

			fmt.Println("\nACTION: DO THIS RIGHT NOW (copy and paste):")
			fmt.Println("\n   gcloud auth application-default login")
			fmt.Println("\n   (This will open your browser. Just click 'Allow')")

			fmt.Println("\nINFO: Then run this command:")
			if projectID != "" {
				fmt.Printf("   vaino scan --provider gcp --project %s\n", projectID)
			} else {
				fmt.Println("   vaino scan --provider gcp --project YOUR-PROJECT-ID")
			}

			fmt.Println("\nQUICK: EVEN EASIER - Let WGO do it for you:")
			fmt.Println("   vaino auth gcp")
			fmt.Println("   (This will handle everything automatically)")
		} else {
			fmt.Println("\nSUCCESS: Good news: You have gcloud installed!")
			fmt.Println("FAILED: Bad news: You're not logged in")

			fmt.Println("\nACTION: DO THESE 3 STEPS (copy and paste each line):")
			fmt.Println("\n   STEP 1:")
			fmt.Println("   gcloud auth login")
			fmt.Println("   (This opens your browser - just click your Google account)")

			fmt.Println("\n   STEP 2:")
			fmt.Println("   gcloud auth application-default login")
			fmt.Println("   (This opens browser again - click 'Allow')")

			fmt.Println("\n   STEP 3:")
			if projectID != "" {
				fmt.Printf("   vaino scan --provider gcp --project %s\n", projectID)
			} else {
				fmt.Println("   vaino scan --provider gcp --project YOUR-PROJECT-ID")
			}

			fmt.Println("\nQUICK: OR JUST RUN THIS (easiest):")
			fmt.Println("   vaino auth gcp")
		}
	} else {
		fmt.Println("\nFAILED: You need gcloud CLI installed first")

		fmt.Println("\nACTION: INSTALL IT NOW:")

		// Detect OS and give exact command
		if ah.isCommandAvailable("brew") {
			fmt.Println("\n   You have Homebrew! Just run:")
			fmt.Println("   brew install google-cloud-sdk")
			fmt.Println("\n   Then run:")
			fmt.Println("   vaino auth gcp")
		} else if ah.isCommandAvailable("apt-get") {
			fmt.Println("\n   Run these commands:")
			fmt.Println("   echo \"deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main\" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list")
			fmt.Println("   curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -")
			fmt.Println("   sudo apt-get update && sudo apt-get install google-cloud-sdk")
			fmt.Println("\n   Then run:")
			fmt.Println("   vaino auth gcp")
		} else {
			fmt.Println("\n   Download from:")
			fmt.Println("   https://cloud.google.com/sdk/docs/install")
			fmt.Println("\n   After installing, run:")
			fmt.Println("   vaino auth gcp")
		}

		fmt.Println("\n🔑 ALTERNATIVE - Use a service account (more steps):")
		fmt.Println("   1. Go to: https://console.cloud.google.com/iam-admin/serviceaccounts")
		fmt.Println("   2. Click 'CREATE SERVICE ACCOUNT'")
		fmt.Println("   3. Name it 'vaino-scanner' and click CREATE")
		fmt.Println("   4. Add role: 'Viewer' and click CONTINUE")
		fmt.Println("   5. Click 'CREATE KEY' → JSON → CREATE")
		fmt.Println("   6. Save the downloaded file somewhere safe")
		fmt.Println("   7. Run:")
		fmt.Println("      export GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/key.json")
		if projectID != "" {
			fmt.Printf("      vaino scan --provider gcp --project %s\n", projectID)
		} else {
			fmt.Println("      vaino scan --provider gcp --project YOUR-PROJECT-ID")
		}
	}

	if hasADC {
		fmt.Printf("\nWARNING:  GOOGLE_APPLICATION_CREDENTIALS is set to: %s\n", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		fmt.Println("   But it might be invalid or have insufficient permissions.")
	}

	fmt.Println("\nDOCS: More Info:")
	fmt.Println("  • GCP Auth Guide: https://cloud.google.com/docs/authentication/getting-started")
	fmt.Println("  • Required Permissions: roles/viewer or equivalent")

	return fmt.Errorf("authentication failed: %v", originalErr)
}

// HandleAWSAuthError provides helpful error messages for AWS auth issues
func (ah *AuthHelper) HandleAWSAuthError(originalErr error) error {
	fmt.Println("\nFAILED: AWS Authentication Failed")
	fmt.Println("=====================================")

	// Check what's available
	hasAwsCli := ah.isCommandAvailable("aws")
	hasCredentials := os.Getenv("AWS_ACCESS_KEY_ID") != ""
	hasProfile := os.Getenv("AWS_PROFILE") != ""

	if hasAwsCli {
		// Check for configured profiles
		cmd := exec.Command("aws", "configure", "list-profiles")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			profiles := strings.TrimSpace(string(output))
			profileList := strings.Split(profiles, "\n")

			fmt.Println("\nSUCCESS: Good news: You have AWS CLI installed with profiles!")
			fmt.Printf("   Available profiles: %s\n", strings.Join(profileList, ", "))

			fmt.Println("\nACTION: DO THIS RIGHT NOW (copy and paste):")
			if len(profileList) > 0 && profileList[0] != "" {
				fmt.Printf("\n   export AWS_PROFILE=%s\n", profileList[0])
				fmt.Println("   vaino scan --provider aws")
			} else {
				fmt.Println("\n   export AWS_PROFILE=default")
				fmt.Println("   vaino scan --provider aws")
			}

			fmt.Println("\nINFO: Using a different profile? Run:")
			fmt.Println("   export AWS_PROFILE=your-profile-name")
			fmt.Println("   vaino scan --provider aws")
		} else {
			fmt.Println("\nSUCCESS: Good news: You have AWS CLI installed!")
			fmt.Println("FAILED: Bad news: No AWS credentials configured")

			fmt.Println("\nACTION: DO THIS RIGHT NOW:")
			fmt.Println("\n   vaino auth aws")
			fmt.Println("   (This will walk you through setup)")

			fmt.Println("\nINFO: Or configure manually:")
			fmt.Println("\n   aws configure")
			fmt.Println("\n   You'll need:")
			fmt.Println("   • AWS Access Key ID (starts with AKIA...)")
			fmt.Println("   • AWS Secret Access Key")
			fmt.Println("   • Default region (just press Enter for us-east-1)")
			fmt.Println("   • Output format (just press Enter)")

			fmt.Println("\n   Then run:")
			fmt.Println("   vaino scan --provider aws")
		}
	} else {
		fmt.Println("\nFAILED: You need AWS CLI installed first")

		fmt.Println("\nACTION: INSTALL IT NOW:")

		// Detect OS and give exact command
		if ah.isCommandAvailable("brew") {
			fmt.Println("\n   You have Homebrew! Just run:")
			fmt.Println("   brew install awscli")
			fmt.Println("\n   Then run:")
			fmt.Println("   vaino auth aws")
		} else if ah.isCommandAvailable("apt-get") {
			fmt.Println("\n   Run this command:")
			fmt.Println("   sudo apt-get update && sudo apt-get install awscli")
			fmt.Println("\n   Then run:")
			fmt.Println("   vaino auth aws")
		} else if ah.isCommandAvailable("yum") {
			fmt.Println("\n   Run this command:")
			fmt.Println("   sudo yum install aws-cli")
			fmt.Println("\n   Then run:")
			fmt.Println("   vaino auth aws")
		} else {
			fmt.Println("\n   Download installer from:")
			fmt.Println("   https://aws.amazon.com/cli/")
			fmt.Println("\n   After installing, run:")
			fmt.Println("   vaino auth aws")
		}

		fmt.Println("\n🔑 QUICK ALTERNATIVE - Use environment variables:")
		fmt.Println("\n   1. Get your AWS credentials from:")
		fmt.Println("      https://console.aws.amazon.com/iam/home#/security_credentials")
		fmt.Println("\n   2. Click 'Create access key'")
		fmt.Println("\n   3. Copy the credentials and run:")
		fmt.Println("      export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_HERE")
		fmt.Println("      export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY_HERE")
		fmt.Println("      export AWS_REGION=us-east-1")
		fmt.Println("      vaino scan --provider aws")
	}

	if hasCredentials {
		fmt.Println("\nWARNING:  AWS_ACCESS_KEY_ID is set but authentication still failed")
		fmt.Println("   Check that AWS_SECRET_ACCESS_KEY is also set and valid")
	}

	if hasProfile {
		fmt.Printf("\nWARNING:  AWS_PROFILE is set to: %s\n", os.Getenv("AWS_PROFILE"))
		fmt.Println("   But it might be invalid or not configured properly")
	}

	return fmt.Errorf("authentication failed: %v", originalErr)
}

// isCommandAvailable checks if a command exists in PATH
func (ah *AuthHelper) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// SetupGCPAuth attempts to automatically set up GCP authentication
func (ah *AuthHelper) SetupGCPAuth(projectID string) error {
	if !ah.isCommandAvailable("gcloud") {
		return fmt.Errorf("gcloud CLI is required but not installed")
	}

	fmt.Println("FIX: Setting up GCP authentication...")

	// Run gcloud auth application-default login
	cmd := exec.Command("gcloud", "auth", "application-default", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set up authentication: %w", err)
	}

	// Set the default project if provided
	if projectID != "" {
		fmt.Printf("\nNOTE: Setting default project to: %s\n", projectID)
		cmd = exec.Command("gcloud", "config", "set", "project", projectID)
		if err := cmd.Run(); err != nil {
			fmt.Printf("WARNING:  Warning: Could not set default project: %v\n", err)
		}
	}

	fmt.Println("\nSUCCESS: GCP authentication configured successfully!")
	fmt.Println("You can now run:")
	fmt.Printf("  vaino scan --provider gcp --project %s\n", projectID)

	return nil
}

// SetupAWSAuth attempts to help set up AWS authentication
func (ah *AuthHelper) SetupAWSAuth() error {
	if !ah.isCommandAvailable("aws") {
		return fmt.Errorf("aws CLI is required but not installed")
	}

	fmt.Println("FIX: Setting up AWS authentication...")
	fmt.Println("\nThis will run 'aws configure' to set up your credentials.")
	fmt.Println("You'll need:")
	fmt.Println("  • AWS Access Key ID")
	fmt.Println("  • AWS Secret Access Key")
	fmt.Println("  • Default region (e.g., us-east-1)")
	fmt.Println("")

	// Run aws configure
	cmd := exec.Command("aws", "configure")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure AWS: %w", err)
	}

	fmt.Println("\nSUCCESS: AWS authentication configured successfully!")
	fmt.Println("You can now run:")
	fmt.Println("  vaino scan --provider aws")

	return nil
}

// CheckTerraformAuth checks if Terraform is properly configured
func (ah *AuthHelper) CheckTerraformAuth() error {
	if !ah.isCommandAvailable("terraform") {
		fmt.Println("\nWARNING:  Terraform CLI not found")
		fmt.Println("\nFIX: Quick Fix:")
		fmt.Println("Install Terraform:")
		fmt.Println("  • macOS: brew install terraform")
		fmt.Println("  • Linux/Windows: https://www.terraform.io/downloads")
		return fmt.Errorf("terraform not installed")
	}

	// Check if we're in a Terraform directory
	if _, err := os.Stat("terraform.tfstate"); err != nil && os.IsNotExist(err) {
		if _, err := os.Stat(".terraform"); err != nil && os.IsNotExist(err) {
			fmt.Println("\nWARNING:  No Terraform state found in current directory")
			fmt.Println("\nFIX: Quick Fix:")
			fmt.Println("Navigate to your Terraform project directory, or specify the path:")
			fmt.Println("  vaino scan --provider terraform --path /path/to/terraform/project")
			return fmt.Errorf("no terraform state found")
		}
	}

	return nil
}
