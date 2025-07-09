package errors

import (
	"fmt"
	"os"
	"strings"
)

// GCPAuthenticationError creates a GCP authentication error with guidance
func GCPAuthenticationError(originalErr error) *WGOError {
	err := New(ErrorTypeAuthentication, ProviderGCP, "GCP authentication failed")

	// Detect specific authentication issues
	if originalErr != nil {
		errStr := originalErr.Error()
		if strings.Contains(errStr, "could not find default credentials") {
			err.WithCause("Application default credentials not found")
		} else if strings.Contains(errStr, "quota") {
			err.WithCause("API quota exceeded")
			err.WithSolutions(
				"Wait for quota reset",
				"Request quota increase in GCP Console",
			)
			return err
		} else {
			err.WithCause(originalErr.Error())
		}
	}

	// Environment-specific solutions
	if err.Environment == "CI/CD detected" {
		err.WithSolutions(
			`export GOOGLE_APPLICATION_CREDENTIALS="service-account.json"`,
			`echo "$GCP_SA_KEY" | base64 -d > service-account.json`,
			`gcloud auth activate-service-account --key-file=service-account.json`,
		)
	} else {
		err.WithSolutions(
			`gcloud auth application-default login`,
			`export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"`,
		)
	}

	err.WithVerify("gcloud auth list")
	err.WithHelp("wgo configure gcp")

	return err
}

// GCPProjectError creates a GCP project configuration error
func GCPProjectError() *WGOError {
	err := New(ErrorTypeConfiguration, ProviderGCP, "GCP project not configured")

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "your-project-id"
	}

	err.WithSolutions(
		fmt.Sprintf(`export GOOGLE_CLOUD_PROJECT="%s"`, projectID),
		`gcloud config set project your-project-id`,
		`wgo configure gcp`,
	)

	err.WithVerify("gcloud config get-value project")
	err.WithHelp("wgo configure --help")

	return err
}

// AWSCredentialsError creates an AWS credentials error with guidance
func AWSCredentialsError(originalErr error) *WGOError {
	err := New(ErrorTypeAuthentication, ProviderAWS, "AWS credentials not found")
	err.WithCause("No valid credential source detected")

	// Check for specific AWS credential issues
	if originalErr != nil && strings.Contains(originalErr.Error(), "ExpiredToken") {
		err.Message = "AWS credentials expired"
		err.WithCause("Security token has expired")
		err.WithSolutions(
			"Refresh AWS credentials",
			"aws sso login (if using SSO)",
			"Get new temporary credentials",
		)
	} else if err.Environment == "CI/CD detected" {
		err.WithSolutions(
			`Configure AWS IAM role for CI/CD`,
			`export AWS_ACCESS_KEY_ID=your-key AWS_SECRET_ACCESS_KEY=your-secret`,
			`Use AWS Secrets Manager or Parameter Store`,
		)
	} else {
		err.WithSolutions(
			`aws configure`,
			`export AWS_ACCESS_KEY_ID=your-key AWS_SECRET_ACCESS_KEY=your-secret`,
			`aws sso login (if using AWS SSO)`,
		)
	}

	err.WithVerify("aws sts get-caller-identity")
	err.WithHelp("wgo configure aws")

	return err
}

// AWSRegionError creates an AWS region configuration error
func AWSRegionError() *WGOError {
	err := New(ErrorTypeConfiguration, ProviderAWS, "AWS region not specified")

	err.WithSolutions(
		`export AWS_REGION=us-east-1`,
		`aws configure set region us-east-1`,
		`Add --region flag to your command`,
	)

	err.WithVerify("aws configure get region")
	err.WithHelp("wgo configure aws")

	return err
}

// KubernetesConnectionError creates a Kubernetes connection error
func KubernetesConnectionError(context string, originalErr error) *WGOError {
	err := New(ErrorTypeNetwork, ProviderKubernetes, "Kubernetes connection failed")

	if context != "" {
		err.WithCause(fmt.Sprintf("Current context '%s' is not accessible", context))
	} else if originalErr != nil {
		err.WithCause(originalErr.Error())
	}

	err.WithSolutions(
		`kubectl config get-contexts`,
		`kubectl config use-context working-context`,
		`Check if cluster is running: kubectl cluster-info`,
		`Verify VPN connection if using remote cluster`,
	)

	err.WithVerify("kubectl cluster-info")
	err.WithHelp("wgo configure kubernetes")

	return err
}

// KubernetesConfigError creates a Kubernetes configuration error
func KubernetesConfigError() *WGOError {
	err := New(ErrorTypeConfiguration, ProviderKubernetes, "Kubernetes configuration not found")
	err.WithCause("No kubeconfig file found")

	err.WithSolutions(
		`Ensure kubectl is configured: kubectl config view`,
		`Set KUBECONFIG environment variable`,
		`Copy config to ~/.kube/config`,
		`For new cluster: gcloud container clusters get-credentials cluster-name`,
	)

	err.WithVerify("kubectl config current-context")
	err.WithHelp("wgo configure kubernetes")

	return err
}

// TerraformStateError creates a Terraform state error
func TerraformStateError(path string) *WGOError {
	err := New(ErrorTypeFileSystem, ProviderTerraform, "No terraform state files found")

	if path != "" {
		err.WithCause(fmt.Sprintf("No .tfstate files in %s", path))
	} else {
		err.WithCause("No .tfstate files in current directory or configured paths")
	}

	err.WithSolutions(
		`Run from terraform project directory`,
		`Configure state paths in ~/.wgo/config.yaml`,
		`Specify path with --path flag`,
		`Check if using remote state backend`,
	)

	err.WithVerify("terraform show")
	err.WithHelp("wgo configure terraform")

	return err
}

// TerraformVersionError creates a Terraform version compatibility error
func TerraformVersionError(required, found string) *WGOError {
	err := New(ErrorTypeValidation, ProviderTerraform, "Terraform version mismatch")
	err.WithCause(fmt.Sprintf("Required: %s, Found: %s", required, found))

	err.WithSolutions(
		`Install required Terraform version`,
		`Use tfenv to manage multiple versions`,
		`Update state file version (use with caution)`,
	)

	err.WithVerify("terraform version")
	err.WithHelp("wgo help terraform")

	return err
}

// PermissionError creates a generic permission error
func PermissionError(provider Provider, resource string) *WGOError {
	err := New(ErrorTypePermission, provider, fmt.Sprintf("Permission denied accessing %s", resource))

	switch provider {
	case ProviderGCP:
		err.WithSolutions(
			`Check IAM permissions in GCP Console`,
			`Ensure service account has required roles`,
			`gcloud projects add-iam-policy-binding PROJECT_ID --member=user:EMAIL --role=roles/viewer`,
		)
		err.WithVerify("gcloud projects get-iam-policy PROJECT_ID")

	case ProviderAWS:
		err.WithSolutions(
			`Check IAM policies attached to user/role`,
			`Use AWS Policy Simulator to test permissions`,
			`aws iam get-user --user-name USERNAME`,
		)
		err.WithVerify("aws iam get-user")

	case ProviderKubernetes:
		err.WithSolutions(
			`Check RBAC permissions`,
			`kubectl auth can-i --list`,
			`Contact cluster administrator for access`,
		)
		err.WithVerify("kubectl auth can-i --list")
	}

	err.WithHelp(fmt.Sprintf("wgo configure %s", strings.ToLower(string(provider))))

	return err
}

// NetworkError creates a network connectivity error
func NetworkError(provider Provider, endpoint string) *WGOError {
	err := New(ErrorTypeNetwork, provider, "Network connection failed")
	err.WithCause(fmt.Sprintf("Cannot reach %s", endpoint))

	err.WithSolutions(
		`Check internet connectivity`,
		`Verify firewall rules`,
		`Check proxy settings: echo $HTTP_PROXY $HTTPS_PROXY`,
		`Try using VPN if accessing private resources`,
	)

	if provider == ProviderGCP {
		err.WithVerify("gcloud compute regions list")
	} else if provider == ProviderAWS {
		err.WithVerify("aws ec2 describe-regions")
	}

	err.WithHelp("wgo help troubleshooting")

	return err
}
