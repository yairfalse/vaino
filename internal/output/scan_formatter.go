package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yairfalse/vaino/pkg/types"
)

// ResourceContext provides human-readable context for different resource types
type ResourceContext struct {
	Description string
	Importance  string
	CostImpact  string
}

// ResourceDescriptions maps resource types to their context
var ResourceDescriptions = map[string]ResourceContext{
	// AWS Resources
	"aws_instance": {
		Description: "EC2 compute instances (virtual servers)",
		Importance:  "Core compute infrastructure",
		CostImpact:  "High - charged per hour/second",
	},
	"aws_s3_bucket": {
		Description: "Object storage containers",
		Importance:  "Data storage and static content",
		CostImpact:  "Medium - charged for storage and requests",
	},
	"aws_vpc": {
		Description: "Virtual private cloud networks",
		Importance:  "Network isolation and security boundaries",
		CostImpact:  "Low - VPCs are free, but NAT gateways cost",
	},
	"aws_security_group": {
		Description: "Virtual firewalls for EC2 instances",
		Importance:  "Critical for security",
		CostImpact:  "None - security groups are free",
	},
	"aws_rds_instance": {
		Description: "Managed relational databases",
		Importance:  "Data persistence layer",
		CostImpact:  "High - charged per hour plus storage",
	},
	"aws_lambda_function": {
		Description: "Serverless compute functions",
		Importance:  "Event-driven processing",
		CostImpact:  "Low - pay per invocation and duration",
	},
	"aws_elb": {
		Description: "Load balancers for traffic distribution",
		Importance:  "High availability and scaling",
		CostImpact:  "Medium - charged per hour plus data",
	},
	"aws_ecs_cluster": {
		Description: "Container orchestration clusters",
		Importance:  "Container management",
		CostImpact:  "Low - clusters free, pay for underlying compute",
	},
	"aws_eks_cluster": {
		Description: "Managed Kubernetes clusters",
		Importance:  "Kubernetes orchestration",
		CostImpact:  "Medium - $0.10/hour for control plane",
	},
	"aws_dynamodb_table": {
		Description: "NoSQL database tables",
		Importance:  "High-performance data storage",
		CostImpact:  "Variable - based on capacity and usage",
	},

	// GCP Resources
	"google_compute_instance": {
		Description: "Compute Engine VMs",
		Importance:  "Core compute infrastructure",
		CostImpact:  "High - charged per second",
	},
	"google_storage_bucket": {
		Description: "Cloud Storage containers",
		Importance:  "Object storage",
		CostImpact:  "Medium - storage and egress charges",
	},
	"google_container_cluster": {
		Description: "GKE Kubernetes clusters",
		Importance:  "Container orchestration",
		CostImpact:  "Medium - management fee plus nodes",
	},

	// Kubernetes Resources
	"Deployment": {
		Description: "Application deployments with replica management",
		Importance:  "Application lifecycle",
		CostImpact:  "Indirect - uses cluster resources",
	},
	"Service": {
		Description: "Network endpoints for applications",
		Importance:  "Service discovery and load balancing",
		CostImpact:  "May trigger cloud load balancer costs",
	},
	"Pod": {
		Description: "Running container instances",
		Importance:  "Actual workloads",
		CostImpact:  "Consumes cluster compute/memory",
	},
	"ConfigMap": {
		Description: "Configuration data storage",
		Importance:  "Application configuration",
		CostImpact:  "Minimal - stored in etcd",
	},
	"Secret": {
		Description: "Sensitive data storage",
		Importance:  "Security-critical",
		CostImpact:  "Minimal - stored encrypted in etcd",
	},
}

// ScanFormatter formats scan output with rich context
type ScanFormatter struct {
	snapshot *types.Snapshot
	quiet    bool
}

// NewScanFormatter creates a new scan formatter
func NewScanFormatter(snapshot *types.Snapshot, quiet bool) *ScanFormatter {
	return &ScanFormatter{
		snapshot: snapshot,
		quiet:    quiet,
	}
}

// FormatOutput generates formatted scan output
func (f *ScanFormatter) FormatOutput() string {
	if f.quiet {
		return ""
	}

	var output strings.Builder

	// If no resources found
	if len(f.snapshot.Resources) == 0 {
		output.WriteString("No resources found in scan.\n")
		output.WriteString("\nPossible reasons:\n")
		output.WriteString("  • No infrastructure exists in the scanned regions\n")
		output.WriteString("  • Insufficient permissions to access resources\n")
		output.WriteString("  • Incorrect provider configuration\n")
		output.WriteString("\nTry running 'vaino check-config' to diagnose issues\n")
		return output.String()
	}

	// Group resources by type
	resourcesByType := f.groupResourcesByType()

	// Group by region
	resourcesByRegion := f.groupResourcesByRegion()

	// Group by state file (for Terraform)
	stateFiles := f.getStateFiles()

	// Build summary
	output.WriteString(fmt.Sprintf("Scanned %s infrastructure at %s\n",
		f.snapshot.Provider,
		f.snapshot.Timestamp.Format("2006-01-02 15:04:05")))
	output.WriteString("\n")

	// Show scan context
	if len(stateFiles) > 0 {
		output.WriteString(fmt.Sprintf("📁 Processed %d Terraform state file(s)\n", len(stateFiles)))
		for _, sf := range stateFiles {
			output.WriteString(fmt.Sprintf("   • %s\n", sf))
		}
		output.WriteString("\n")
	}

	// Resource summary with context
	output.WriteString(fmt.Sprintf("📊 Found %d resources:\n\n", len(f.snapshot.Resources)))

	// Sort resource types by count (descending)
	types := make([]string, 0, len(resourcesByType))
	for t := range resourcesByType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return len(resourcesByType[types[i]]) > len(resourcesByType[types[j]])
	})

	// Display resources with context
	for _, resourceType := range types {
		resources := resourcesByType[resourceType]
		count := len(resources)

		// Get resource context
		context, hasContext := ResourceDescriptions[resourceType]

		// Resource type header
		output.WriteString(fmt.Sprintf("  %s (%d)\n", resourceType, count))

		if hasContext {
			output.WriteString(fmt.Sprintf("  └─ %s\n", context.Description))
			if context.CostImpact != "" && context.CostImpact != "None" {
				output.WriteString(fmt.Sprintf("     💰 Cost impact: %s\n", context.CostImpact))
			}
		}

		// Show a few example resources (up to 3)
		examples := resources
		if len(examples) > 3 {
			examples = examples[:3]
		}

		for _, r := range examples {
			name := r.Name
			if name == "" {
				name = r.ID
			}
			if len(name) > 40 {
				name = name[:37] + "..."
			}

			location := ""
			if r.Region != "" {
				location = fmt.Sprintf(" [%s]", r.Region)
			} else if r.Namespace != "" {
				location = fmt.Sprintf(" [ns: %s]", r.Namespace)
			}

			output.WriteString(fmt.Sprintf("     • %s%s\n", name, location))
		}

		if len(resources) > 3 {
			output.WriteString(fmt.Sprintf("     ... and %d more\n", len(resources)-3))
		}

		output.WriteString("\n")
	}

	// Regional distribution
	if len(resourcesByRegion) > 1 {
		output.WriteString("🌍 Geographic distribution:\n")
		regions := make([]string, 0, len(resourcesByRegion))
		for r := range resourcesByRegion {
			if r != "" {
				regions = append(regions, r)
			}
		}
		sort.Strings(regions)

		for _, region := range regions {
			count := len(resourcesByRegion[region])
			output.WriteString(fmt.Sprintf("  • %s: %d resources\n", region, count))
		}
		output.WriteString("\n")
	}

	// Next steps
	output.WriteString("📌 Next steps:\n")
	output.WriteString("  • Run 'vaino diff' to detect any drift\n")
	output.WriteString("  • Run 'vaino watch' to monitor changes in real-time\n")
	output.WriteString("  • Run 'vaino scan --baseline' to set this as your baseline\n")

	return output.String()
}

func (f *ScanFormatter) groupResourcesByType() map[string][]types.Resource {
	groups := make(map[string][]types.Resource)
	for _, r := range f.snapshot.Resources {
		groups[r.Type] = append(groups[r.Type], r)
	}
	return groups
}

func (f *ScanFormatter) groupResourcesByRegion() map[string][]types.Resource {
	groups := make(map[string][]types.Resource)
	for _, r := range f.snapshot.Resources {
		region := r.Region
		if region == "" {
			region = "global"
		}
		groups[region] = append(groups[region], r)
	}
	return groups
}

func (f *ScanFormatter) getStateFiles() []string {
	files := make(map[string]bool)
	for _, r := range f.snapshot.Resources {
		if r.Metadata.StateFile != "" {
			files[r.Metadata.StateFile] = true
		}
	}

	result := make([]string, 0, len(files))
	for f := range files {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

func (f *ScanFormatter) generateCostInsights(resourcesByType map[string][]types.Resource) string {
	var insights []string

	// Check for high-cost resources
	highCostTypes := []string{"aws_instance", "aws_rds_instance", "google_compute_instance", "aws_eks_cluster"}
	highCostCount := 0
	for _, t := range highCostTypes {
		if resources, ok := resourcesByType[t]; ok {
			highCostCount += len(resources)
		}
	}

	if highCostCount > 0 {
		insights = append(insights, fmt.Sprintf("  • Found %d high-cost compute/database resources", highCostCount))
	}

	// Check for potential waste
	if lambdas, ok := resourcesByType["aws_lambda_function"]; ok && len(lambdas) > 10 {
		insights = append(insights, fmt.Sprintf("  • %d Lambda functions - consider consolidating if underutilized", len(lambdas)))
	}

	// Storage insights
	storageTypes := []string{"aws_s3_bucket", "google_storage_bucket", "aws_dynamodb_table"}
	storageCount := 0
	for _, t := range storageTypes {
		if resources, ok := resourcesByType[t]; ok {
			storageCount += len(resources)
		}
	}
	if storageCount > 0 {
		insights = append(insights, fmt.Sprintf("  • %d storage resources - monitor for unused data", storageCount))
	}

	return strings.Join(insights, "\n")
}

func (f *ScanFormatter) generateSecurityInsights(resourcesByType map[string][]types.Resource) string {
	var insights []string

	// Security groups
	if sgs, ok := resourcesByType["aws_security_group"]; ok {
		insights = append(insights, fmt.Sprintf("  • %d security groups - audit for overly permissive rules", len(sgs)))
	}

	// Secrets management
	if secrets, ok := resourcesByType["Secret"]; ok && len(secrets) > 0 {
		insights = append(insights, fmt.Sprintf("  • %d Kubernetes secrets - ensure proper RBAC", len(secrets)))
	}

	// Public resources warning
	if buckets, ok := resourcesByType["aws_s3_bucket"]; ok && len(buckets) > 0 {
		insights = append(insights, "  • S3 buckets found - verify public access settings")
	}

	return strings.Join(insights, "\n")
}
