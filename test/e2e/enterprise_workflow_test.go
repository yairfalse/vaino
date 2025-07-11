//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EEnterpriseWorkflow tests the complete enterprise workflow
func TestE2EEnterpriseWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup test environment
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	t.Run("CompleteEnterpriseWorkflow", func(t *testing.T) {
		testCompleteEnterpriseWorkflow(t, env)
	})

	t.Run("BaselineManagement", func(t *testing.T) {
		testBaselineManagement(t, env)
	})

	t.Run("ComplianceReporting", func(t *testing.T) {
		testComplianceReporting(t, env)
	})

	t.Run("PerformanceValidation", func(t *testing.T) {
		testPerformanceValidation(t, env)
	})
}

func testCompleteEnterpriseWorkflow(t *testing.T, env *e2eTestEnvironment) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 1: Create initial infrastructure snapshot
	t.Log("Step 1: Creating initial infrastructure snapshot")
	initialSnapshot := env.createTestSnapshot("initial", 1000)
	env.saveSnapshot(initialSnapshot, "initial.json")

	// Step 2: Create baseline from initial snapshot
	t.Log("Step 2: Creating baseline from snapshot")
	output, err := env.runCommand(ctx, "baseline", "create", "production",
		"--snapshot", filepath.Join(env.tmpDir, "initial.json"),
		"--description", "Production baseline Q4 2024",
		"--created-by", "test@company.com",
		"--max-changes", "50",
		"--max-risk", "0.8",
		"--enable-alerts")

	require.NoError(t, err, "Failed to create baseline")
	assert.Contains(t, output, "✅ Baseline created successfully")
	assert.Contains(t, output, "production")

	// Step 3: Approve baseline
	t.Log("Step 3: Approving baseline")
	output, err = env.runCommand(ctx, "baseline", "approve", "production",
		"--approved-by", "manager@company.com")

	require.NoError(t, err, "Failed to approve baseline")
	assert.Contains(t, output, "approved")

	// Step 4: Simulate infrastructure changes
	t.Log("Step 4: Simulating infrastructure changes")
	currentSnapshot := env.createTestSnapshot("current", 1000)
	env.simulateInfrastructureChanges(currentSnapshot, 0.1) // 10% changes
	env.saveSnapshot(currentSnapshot, "current.json")

	// Step 5: Detect drift using enterprise engine
	t.Log("Step 5: Detecting drift with enterprise engine")
	output, err = env.runCommand(ctx, "diff",
		"--baseline", "production",
		"--to", filepath.Join(env.tmpDir, "current.json"),
		"--enterprise",
		"--correlation",
		"--risk-assessment",
		"--compliance-report", filepath.Join(env.tmpDir, "compliance.json"),
		"--executive-summary")

	require.NoError(t, err, "Failed to detect drift")
	assert.Contains(t, output, "ENTERPRISE INFRASTRUCTURE ANALYSIS")
	assert.Contains(t, output, "RISK BREAKDOWN")

	// Step 6: Verify compliance report was generated
	t.Log("Step 6: Verifying compliance report")
	complianceFile := filepath.Join(env.tmpDir, "compliance.json")
	assert.FileExists(t, complianceFile)

	complianceData, err := os.ReadFile(complianceFile)
	require.NoError(t, err)

	var compliance map[string]interface{}
	err = json.Unmarshal(complianceData, &compliance)
	require.NoError(t, err)
	assert.Contains(t, compliance, "overall_status")
	assert.Contains(t, compliance, "score")

	// Step 7: Test executive summary output
	t.Log("Step 7: Testing executive summary")
	output, err = env.runCommand(ctx, "diff",
		"--baseline", "production",
		"--to", filepath.Join(env.tmpDir, "current.json"),
		"--executive-summary")

	require.NoError(t, err, "Failed to generate executive summary")
	assert.Contains(t, output, "EXECUTIVE INFRASTRUCTURE DRIFT SUMMARY")
	assert.Contains(t, output, "Overall Risk Level")
	assert.Contains(t, output, "KEY FINDINGS")
	assert.Contains(t, output, "RECOMMENDATIONS")

	// Step 8: Test streaming mode
	t.Log("Step 8: Testing streaming mode")
	output, err = env.runCommand(ctx, "diff",
		"--from", filepath.Join(env.tmpDir, "initial.json"),
		"--to", filepath.Join(env.tmpDir, "current.json"),
		"--streaming",
		"--enterprise")

	require.NoError(t, err, "Failed to run streaming diff")
	assert.Contains(t, output, "Real-time Change Stream")

	t.Log("✅ Complete enterprise workflow test passed")
}

func testBaselineManagement(t *testing.T, env *e2eTestEnvironment) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test baseline creation
	t.Log("Testing baseline creation")
	snapshot := env.createTestSnapshot("baseline-test", 500)
	env.saveSnapshot(snapshot, "baseline-test.json")

	output, err := env.runCommand(ctx, "baseline", "create", "test-baseline",
		"--snapshot", filepath.Join(env.tmpDir, "baseline-test.json"),
		"--description", "Test baseline for validation",
		"--created-by", "test-user",
		"--tags", "env=test,team=platform")

	require.NoError(t, err, "Failed to create test baseline")
	assert.Contains(t, output, "✅ Baseline created successfully")

	// Test baseline listing
	t.Log("Testing baseline listing")
	output, err = env.runCommand(ctx, "baseline", "list")
	require.NoError(t, err, "Failed to list baselines")
	assert.Contains(t, output, "test-baseline")

	// Test baseline show
	t.Log("Testing baseline show")
	output, err = env.runCommand(ctx, "baseline", "show", "test-baseline")
	require.NoError(t, err, "Failed to show baseline")
	assert.Contains(t, output, "test-baseline")
	assert.Contains(t, output, "Test baseline for validation")

	// Test baseline approval
	t.Log("Testing baseline approval")
	output, err = env.runCommand(ctx, "baseline", "approve", "test-baseline",
		"--approved-by", "approver@company.com")
	require.NoError(t, err, "Failed to approve baseline")
	assert.Contains(t, output, "approved")

	t.Log("✅ Baseline management test passed")
}

func testComplianceReporting(t *testing.T, env *e2eTestEnvironment) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create snapshots with security issues
	baseline := env.createTestSnapshot("compliance-baseline", 200)
	current := env.createTestSnapshot("compliance-current", 200)
	env.simulateSecurityChanges(current) // Add security violations

	env.saveSnapshot(baseline, "compliance-baseline.json")
	env.saveSnapshot(current, "compliance-current.json")

	// Test compliance reporting with different frameworks
	frameworks := []string{"SOC2", "PCI-DSS", "NIST"}

	for _, framework := range frameworks {
		t.Logf("Testing %s compliance framework", framework)

		output, err := env.runCommand(ctx, "diff",
			"--from", filepath.Join(env.tmpDir, "compliance-baseline.json"),
			"--to", filepath.Join(env.tmpDir, "compliance-current.json"),
			"--enterprise",
			"--compliance-report", filepath.Join(env.tmpDir, fmt.Sprintf("compliance-%s.json", strings.ToLower(framework))),
			"--policy-framework", framework)

		require.NoError(t, err, "Failed to generate compliance report for %s", framework)

		// Verify compliance report file was created
		complianceFile := filepath.Join(env.tmpDir, fmt.Sprintf("compliance-%s.json", strings.ToLower(framework)))
		assert.FileExists(t, complianceFile)
	}

	// Test compliance output format
	t.Log("Testing compliance output format")
	output, err := env.runCommand(ctx, "diff",
		"--from", filepath.Join(env.tmpDir, "compliance-baseline.json"),
		"--to", filepath.Join(env.tmpDir, "compliance-current.json"),
		"--format", "compliance")

	require.NoError(t, err, "Failed to generate compliance output")
	assert.Contains(t, output, "COMPLIANCE REPORT")
	assert.Contains(t, output, "VIOLATIONS")

	t.Log("✅ Compliance reporting test passed")
}

func testPerformanceValidation(t *testing.T, env *e2eTestEnvironment) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Test performance with different dataset sizes
	testCases := []struct {
		name          string
		resourceCount int
		maxDuration   time.Duration
	}{
		{"1K Resources", 1000, 2 * time.Second},
		{"5K Resources", 5000, 5 * time.Second},
		{"10K Resources", 10000, 10 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing performance with %d resources", tc.resourceCount)

			// Create large snapshots
			baseline := env.createTestSnapshot("perf-baseline", tc.resourceCount)
			current := env.createTestSnapshot("perf-current", tc.resourceCount)
			env.simulateInfrastructureChanges(current, 0.05) // 5% changes

			env.saveSnapshot(baseline, fmt.Sprintf("perf-baseline-%d.json", tc.resourceCount))
			env.saveSnapshot(current, fmt.Sprintf("perf-current-%d.json", tc.resourceCount))

			// Measure performance
			start := time.Now()
			output, err := env.runCommand(ctx, "diff",
				"--from", filepath.Join(env.tmpDir, fmt.Sprintf("perf-baseline-%d.json", tc.resourceCount)),
				"--to", filepath.Join(env.tmpDir, fmt.Sprintf("perf-current-%d.json", tc.resourceCount)),
				"--enterprise",
				"--progress")
			duration := time.Since(start)

			require.NoError(t, err, "Failed performance test for %s", tc.name)

			if duration > tc.maxDuration {
				t.Errorf("Performance requirement failed for %s: %v > %v", tc.name, duration, tc.maxDuration)
			} else {
				t.Logf("✅ Performance passed for %s: %v (< %v)", tc.name, duration, tc.maxDuration)
			}

			// Verify analysis completed
			assert.Contains(t, output, "Diff analysis complete")
		})
	}

	t.Log("✅ Performance validation test passed")
}

// e2eTestEnvironment provides test environment setup
type e2eTestEnvironment struct {
	t         *testing.T
	tmpDir    string
	wgoBinary string
}

func setupE2EEnvironment(t *testing.T) *e2eTestEnvironment {
	tmpDir, err := os.MkdirTemp("", "vaino-e2e-enterprise")
	require.NoError(t, err)

	// Build binary if needed
	wgoBinary := "./vaino"
	if _, err := os.Stat(wgoBinary); os.IsNotExist(err) {
		t.Log("Building vaino binary for E2E tests...")
		cmd := exec.Command("go", "build", "-o", wgoBinary, "./cmd/vaino")
		err := cmd.Run()
		require.NoError(t, err, "Failed to build vaino binary")
	}

	return &e2eTestEnvironment{
		t:         t,
		tmpDir:    tmpDir,
		wgoBinary: wgoBinary,
	}
}

func (env *e2eTestEnvironment) cleanup() {
	if env.tmpDir != "" {
		os.RemoveAll(env.tmpDir)
	}
}

func (env *e2eTestEnvironment) runCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, env.wgoBinary, args...)
	cmd.Dir = env.tmpDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (env *e2eTestEnvironment) createTestSnapshot(id string, resourceCount int) map[string]interface{} {
	resources := make([]map[string]interface{}, resourceCount)

	resourceTypes := []string{
		"aws_instance", "aws_security_group", "aws_s3_bucket", "aws_iam_role",
		"kubernetes_deployment", "kubernetes_service", "kubernetes_configmap",
	}

	for i := 0; i < resourceCount; i++ {
		resourceType := resourceTypes[i%len(resourceTypes)]
		resources[i] = map[string]interface{}{
			"id":       fmt.Sprintf("%s-%d", resourceType, i),
			"type":     resourceType,
			"name":     fmt.Sprintf("resource-%d", i),
			"provider": getProviderFromType(resourceType),
			"region":   getRegionForIndex(i),
			"configuration": map[string]interface{}{
				"instance_type": "t3.micro",
				"ami":           "ami-12345678",
				"tags": map[string]string{
					"Name":        fmt.Sprintf("resource-%d", i),
					"Environment": getEnvironmentForIndex(i),
				},
			},
			"tags": map[string]string{
				"Name":        fmt.Sprintf("resource-%d", i),
				"Environment": getEnvironmentForIndex(i),
			},
		}
	}

	return map[string]interface{}{
		"id":        id,
		"timestamp": time.Now().Format(time.RFC3339),
		"provider":  "multi",
		"resources": resources,
	}
}

func (env *e2eTestEnvironment) saveSnapshot(snapshot map[string]interface{}, filename string) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	require.NoError(env.t, err)

	filepath := filepath.Join(env.tmpDir, filename)
	err = os.WriteFile(filepath, data, 0644)
	require.NoError(env.t, err)
}

func (env *e2eTestEnvironment) simulateInfrastructureChanges(snapshot map[string]interface{}, changePercent float64) {
	resources, ok := snapshot["resources"].([]map[string]interface{})
	if !ok {
		return
	}

	changeCount := int(float64(len(resources)) * changePercent)

	for i := 0; i < changeCount && i < len(resources); i++ {
		resource := resources[i]
		if config, ok := resource["configuration"].(map[string]interface{}); ok {
			// Modify instance type
			if i%3 == 0 {
				config["instance_type"] = "t3.small"
			}

			// Add monitoring
			if i%5 == 0 {
				config["monitoring"] = true
			}

			// Update tags
			if tags, ok := config["tags"].(map[string]string); ok {
				tags["Modified"] = "true"
				tags["ModifiedAt"] = time.Now().Format(time.RFC3339)
			}
		}
	}
}

func (env *e2eTestEnvironment) simulateSecurityChanges(snapshot map[string]interface{}) {
	resources, ok := snapshot["resources"].([]map[string]interface{})
	if !ok {
		return
	}

	for i, resource := range resources {
		resourceType, _ := resource["type"].(string)

		if resourceType == "aws_security_group" && i%3 == 0 {
			if config, ok := resource["configuration"].(map[string]interface{}); ok {
				// Add overly permissive rule
				config["ingress_rules"] = []map[string]interface{}{
					{
						"from_port":   80,
						"to_port":     80,
						"protocol":    "tcp",
						"cidr_blocks": []string{"0.0.0.0/0"},
					},
				}
			}
		}

		if resourceType == "aws_iam_role" && i%4 == 0 {
			if config, ok := resource["configuration"].(map[string]interface{}); ok {
				// Add overly broad permissions
				config["policies"] = []string{"arn:aws:iam::aws:policy/PowerUserAccess"}
			}
		}
	}
}

// Helper functions (reuse from other test files)
func getProviderFromType(resourceType string) string {
	if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	}
	if strings.HasPrefix(resourceType, "kubernetes_") {
		return "kubernetes"
	}
	return "unknown"
}

func getRegionForIndex(index int) string {
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	return regions[index%len(regions)]
}

func getEnvironmentForIndex(index int) string {
	envs := []string{"production", "staging", "development"}
	return envs[index%len(envs)]
}
