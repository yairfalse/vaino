package differ

import (
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestDifferEngine_FullWorkflow(t *testing.T) {
	// Create a comprehensive test scenario that simulates a real infrastructure drift
	baseline := &types.Snapshot{
		ID:        "baseline-prod-v1.0",
		Timestamp: time.Now().Add(-24 * time.Hour),
		Provider:  "aws",
		Resources: []types.Resource{
			// Web server instance
			{
				ID:       "i-web-server-1",
				Type:     "instance",
				Name:     "web-server-1",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"instance_type":      "t3.medium",
					"state":              "running",
					"security_group_ids": []interface{}{"sg-web-123"},
					"subnet_id":          "subnet-public-1",
					"public_ip":          "52.1.2.3",
					"monitoring_enabled": true,
				},
				Tags: map[string]string{
					"Environment": "production",
					"Application": "web",
					"Team":        "backend",
				},
			},
			// Database instance
			{
				ID:       "i-db-server-1",
				Type:     "instance",
				Name:     "db-server-1",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"instance_type":      "db.t3.large",
					"state":              "running",
					"security_group_ids": []interface{}{"sg-db-456"},
					"subnet_id":          "subnet-private-1",
					"backup_enabled":     true,
					"encryption_enabled": true,
				},
				Tags: map[string]string{
					"Environment": "production",
					"Application": "database",
					"Team":        "backend",
				},
			},
			// Security group for web
			{
				ID:       "sg-web-123",
				Type:     "security_group",
				Name:     "web-sg",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"ingress_rules": []interface{}{
						map[string]interface{}{
							"port":        80,
							"protocol":    "tcp",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
						map[string]interface{}{
							"port":        443,
							"protocol":    "tcp",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
					},
					"egress_rules": []interface{}{
						map[string]interface{}{
							"port":        0,
							"protocol":    "all",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
					},
				},
			},
		},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    3,
		},
	}

	// Current state with various types of drift
	current := &types.Snapshot{
		ID:        "current-scan-12345",
		Timestamp: time.Now(),
		Provider:  "aws",
		Resources: []types.Resource{
			// Web server - instance type changed (CRITICAL)
			{
				ID:       "i-web-server-1",
				Type:     "instance",
				Name:     "web-server-1",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"instance_type":      "t3.xlarge", // CRITICAL: Instance type changed
					"state":              "running",
					"security_group_ids": []interface{}{"sg-web-123"},
					"subnet_id":          "subnet-public-1",
					"public_ip":          "52.1.2.4", // MEDIUM: IP changed
					"monitoring_enabled": false,      // HIGH: Monitoring disabled
				},
				Tags: map[string]string{
					"Environment": "production",
					"Application": "web",
					"Team":        "frontend", // LOW: Team tag changed
				},
			},
			// Database - encryption disabled (CRITICAL)
			{
				ID:       "i-db-server-1",
				Type:     "instance",
				Name:     "db-server-1",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"instance_type":      "db.t3.large",
					"state":              "running",
					"security_group_ids": []interface{}{"sg-db-456"},
					"subnet_id":          "subnet-private-1",
					"backup_enabled":     true,
					"encryption_enabled": false, // CRITICAL: Encryption disabled
				},
				Tags: map[string]string{
					"Environment": "production",
					"Application": "database",
					"Team":        "backend",
				},
			},
			// Security group - SSH port added (CRITICAL)
			{
				ID:       "sg-web-123",
				Type:     "security_group",
				Name:     "web-sg",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"ingress_rules": []interface{}{
						map[string]interface{}{
							"port":        80,
							"protocol":    "tcp",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
						map[string]interface{}{
							"port":        443,
							"protocol":    "tcp",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
						map[string]interface{}{
							"port":        22, // CRITICAL: SSH port added
							"protocol":    "tcp",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
					},
					"egress_rules": []interface{}{
						map[string]interface{}{
							"port":        0,
							"protocol":    "all",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
					},
				},
			},
			// New load balancer added
			{
				ID:       "elb-new-123",
				Type:     "load_balancer",
				Name:     "web-lb",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"scheme":             "internet-facing",
					"load_balancer_type": "application",
					"subnets":            []interface{}{"subnet-public-1", "subnet-public-2"},
				},
				Tags: map[string]string{
					"Environment": "production",
					"Application": "load-balancer",
				},
			},
		},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    4,
		},
	}

	// Test different diff options
	testCases := []struct {
		name           string
		options        DiffOptions
		expectCritical int
		expectHigh     int
		expectMedium   int
		expectLow      int
		expectAdded    int
		expectModified int
	}{
		{
			name:           "full comparison",
			options:        DiffOptions{},
			expectCritical: 3, // instance_type, encryption_enabled, SSH port
			expectHigh:     1, // monitoring_enabled
			expectMedium:   1, // public_ip
			expectLow:      1, // team tag
			expectAdded:    1, // load balancer
			expectModified: 3, // web server, db server, security group
		},
		{
			name: "ignore monitoring fields",
			options: DiffOptions{
				IgnoreFields: []string{"monitoring_enabled"},
			},
			expectCritical: 3, // instance_type, encryption_enabled, SSH port
			expectHigh:     0, // monitoring ignored
			expectMedium:   1, // public_ip
			expectLow:      1, // team tag
			expectAdded:    1, // load balancer
			expectModified: 3, // still 3 resources modified (other changes)
		},
		{
			name: "only high risk and above",
			options: DiffOptions{
				MinRiskLevel: RiskLevelHigh,
			},
			expectCritical: 3, // instance_type, encryption_enabled, SSH port
			expectHigh:     1, // monitoring_enabled
			expectMedium:   0, // filtered out
			expectLow:      0, // filtered out
			expectAdded:    1, // load balancer (always included)
			expectModified: 3, // all resources still have high+ risk changes
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewDifferEngine(tc.options)

			report, err := engine.Compare(baseline, current)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify basic report structure
			if report.BaselineID != baseline.ID {
				t.Errorf("expected BaselineID %s, got %s", baseline.ID, report.BaselineID)
			}
			if report.CurrentID != current.ID {
				t.Errorf("expected CurrentID %s, got %s", current.ID, report.CurrentID)
			}

			// Verify summary counts
			if report.Summary.AddedResources != tc.expectAdded {
				t.Errorf("expected %d added resources, got %d", tc.expectAdded, report.Summary.AddedResources)
			}
			if report.Summary.ModifiedResources != tc.expectModified {
				t.Errorf("expected %d modified resources, got %d", tc.expectModified, report.Summary.ModifiedResources)
			}

			// Verify severity distribution
			if report.Summary.ChangesBySeverity[RiskLevelCritical] != tc.expectCritical {
				t.Errorf("expected %d critical changes, got %d", tc.expectCritical, report.Summary.ChangesBySeverity[RiskLevelCritical])
			}
			if report.Summary.ChangesBySeverity[RiskLevelHigh] != tc.expectHigh {
				t.Errorf("expected %d high changes, got %d", tc.expectHigh, report.Summary.ChangesBySeverity[RiskLevelHigh])
			}
			if report.Summary.ChangesBySeverity[RiskLevelMedium] != tc.expectMedium {
				t.Errorf("expected %d medium changes, got %d", tc.expectMedium, report.Summary.ChangesBySeverity[RiskLevelMedium])
			}
			if report.Summary.ChangesBySeverity[RiskLevelLow] != tc.expectLow {
				t.Errorf("expected %d low changes, got %d", tc.expectLow, report.Summary.ChangesBySeverity[RiskLevelLow])
			}

			// Verify category distribution
			expectedCategories := []DriftCategory{
				DriftCategoryConfig,   // instance_type, team tag
				DriftCategorySecurity, // SSH port, encryption_enabled
				DriftCategoryNetwork,  // public_ip
				DriftCategoryState,    // monitoring_enabled
			}

			for _, category := range expectedCategories {
				if count, exists := report.Summary.ChangesByCategory[category]; !exists || count == 0 {
					// Only check if we expect changes in this category for this test case
					if tc.name == "full comparison" {
						t.Errorf("expected changes in category %s", category)
					}
				}
			}

			// Verify overall risk assessment
			if report.Summary.OverallRisk == RiskLevelLow {
				t.Error("expected overall risk to be higher than low with critical security changes")
			}

			// Verify specific resource changes
			foundWebServer := false
			foundDatabase := false
			foundSecurityGroup := false
			foundLoadBalancer := false

			for _, resourceChange := range report.ResourceChanges {
				switch resourceChange.ResourceID {
				case "i-web-server-1":
					foundWebServer = true
					if resourceChange.DriftType != ChangeTypeModified {
						t.Errorf("expected web server to be modified, got %s", resourceChange.DriftType)
					}
				case "i-db-server-1":
					foundDatabase = true
					if resourceChange.DriftType != ChangeTypeModified {
						t.Errorf("expected database to be modified, got %s", resourceChange.DriftType)
					}
				case "sg-web-123":
					foundSecurityGroup = true
					if resourceChange.DriftType != ChangeTypeModified {
						t.Errorf("expected security group to be modified, got %s", resourceChange.DriftType)
					}
				case "elb-new-123":
					foundLoadBalancer = true
					if resourceChange.DriftType != ChangeTypeAdded {
						t.Errorf("expected load balancer to be added, got %s", resourceChange.DriftType)
					}
				}
			}

			if !foundWebServer {
				t.Error("expected to find web server changes")
			}
			if !foundDatabase {
				t.Error("expected to find database changes")
			}
			if !foundSecurityGroup {
				t.Error("expected to find security group changes")
			}
			if !foundLoadBalancer {
				t.Error("expected to find load balancer addition")
			}
		})
	}
}

func TestDifferEngine_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		baseline *types.Snapshot
		current  *types.Snapshot
		validate func(t *testing.T, report *DriftReport)
	}{
		{
			name: "kubernetes deployment scaling",
			baseline: &types.Snapshot{
				ID: "k8s-baseline",
				Resources: []types.Resource{
					{
						ID:       "deployment-web",
						Type:     "deployment",
						Provider: "kubernetes",
						Configuration: map[string]interface{}{
							"replicas": 3,
							"image":    "nginx:1.20",
						},
					},
				},
			},
			current: &types.Snapshot{
				ID: "k8s-current",
				Resources: []types.Resource{
					{
						ID:       "deployment-web",
						Type:     "deployment",
						Provider: "kubernetes",
						Configuration: map[string]interface{}{
							"replicas": 5,            // Scaled up
							"image":    "nginx:1.21", // Image updated
						},
					},
				},
			},
			validate: func(t *testing.T, report *DriftReport) {
				if report.Summary.ModifiedResources != 1 {
					t.Errorf("expected 1 modified resource, got %d", report.Summary.ModifiedResources)
				}
				// Scaling should be medium risk, image update low risk
				if report.Summary.ChangesBySeverity[RiskLevelMedium] < 1 {
					t.Error("expected at least one medium risk change for scaling")
				}
			},
		},
		{
			name: "terraform state drift",
			baseline: &types.Snapshot{
				ID: "tf-baseline",
				Resources: []types.Resource{
					{
						ID:       "aws_s3_bucket.data",
						Type:     "s3_bucket",
						Provider: "aws",
						Configuration: map[string]interface{}{
							"versioning_enabled":      true,
							"public_read_prevented":   true,
							"public_write_prevented":  true,
							"public_access_blocked":   true,
							"restrict_public_buckets": true,
						},
					},
				},
			},
			current: &types.Snapshot{
				ID: "tf-current",
				Resources: []types.Resource{
					{
						ID:       "aws_s3_bucket.data",
						Type:     "s3_bucket",
						Provider: "aws",
						Configuration: map[string]interface{}{
							"versioning_enabled":      true,
							"public_read_prevented":   false, // CRITICAL: Public read allowed
							"public_write_prevented":  true,
							"public_access_blocked":   true,
							"restrict_public_buckets": true,
						},
					},
				},
			},
			validate: func(t *testing.T, report *DriftReport) {
				if report.Summary.ModifiedResources != 1 {
					t.Errorf("expected 1 modified resource, got %d", report.Summary.ModifiedResources)
				}
				// Public access change should be critical
				if report.Summary.ChangesBySeverity[RiskLevelCritical] < 1 {
					t.Error("expected at least one critical change for public access")
				}
				if report.Summary.ChangesByCategory[DriftCategorySecurity] < 1 {
					t.Error("expected security category changes")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewDifferEngine(DiffOptions{})
			report, err := engine.Compare(tt.baseline, tt.current)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.validate(t, report)
		})
	}
}
