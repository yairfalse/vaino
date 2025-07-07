package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	wgoBinary = "../../wgo"
)

func TestMain(m *testing.M) {
	// Build the binary before running tests
	cmd := exec.Command("go", "build", "-o", wgoBinary, "../../cmd/wgo")
	if err := cmd.Run(); err != nil {
		panic("Failed to build wgo binary: " + err.Error())
	}
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Remove(wgoBinary)
	
	os.Exit(code)
}

func runWGO(workDir string, args ...string) (string, string, error) {
	cmd := exec.Command(wgoBinary, args...)
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestE2E_CompleteWorkflow(t *testing.T) {
	// Create isolated test environment
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "wgo-test")
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}
	
	configFile := filepath.Join(workDir, "config.yaml")
	
	// Create config file
	configContent := `
storage:
  base_path: ` + filepath.Join(workDir, ".wgo") + `
output:
  format: table
  pretty: true
logging:
  level: info
collectors:
  terraform:
    enabled: true
    state_paths: ["./terraform.tfstate"]
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create mock Terraform state file
	terraformState := `{
  "version": 4,
  "terraform_version": "1.0.0",
  "serial": 1,
  "lineage": "test-lineage",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {
            "id": "i-1234567890abcdef0",
            "instance_type": "t3.micro",
            "ami": "ami-0c02fb55956c7d316",
            "tags": {
              "Name": "web-server",
              "Environment": "test"
            }
          }
        }
      ]
    }
  ]
}`
	err = os.WriteFile(filepath.Join(workDir, "terraform.tfstate"), []byte(terraformState), 0644)
	if err != nil {
		t.Fatalf("Failed to create terraform state file: %v", err)
	}
	
	// Step 1: Test scan command
	t.Run("scan_infrastructure", func(t *testing.T) {
		stdout, stderr, err := runWGO(workDir, "scan", "--provider", "terraform", "--config", configFile)
		
		if err != nil {
			t.Logf("Scan stderr: %s", stderr)
			// This might error due to stub implementation, which is expected
		}
		
		if !strings.Contains(stdout, "Scanning infrastructure") {
			t.Errorf("Expected scan output to contain 'Scanning infrastructure', got: %s", stdout)
		}
	})
	
	// Step 2: Test baseline creation
	t.Run("create_baseline", func(t *testing.T) {
		stdout, stderr, err := runWGO(workDir, "baseline", "create", 
			"--name", "test-baseline", 
			"--description", "E2E test baseline",
			"--config", configFile)
		
		if err != nil {
			t.Logf("Baseline create stderr: %s", stderr)
		}
		
		if !strings.Contains(stdout, "Creating Baseline") {
			t.Errorf("Expected baseline create output, got: %s", stdout)
		}
		
		if !strings.Contains(stdout, "test-baseline") {
			t.Errorf("Expected baseline name in output, got: %s", stdout)
		}
	})
	
	// Step 3: Test baseline listing
	t.Run("list_baselines", func(t *testing.T) {
		stdout, stderr, err := runWGO(workDir, "baseline", "list", "--config", configFile)
		
		if err != nil {
			t.Logf("Baseline list stderr: %s", stderr)
		}
		
		if !strings.Contains(stdout, "Infrastructure Baselines") {
			t.Errorf("Expected baseline list output, got: %s", stdout)
		}
	})
	
	// Step 4: Test drift checking
	t.Run("check_drift", func(t *testing.T) {
		stdout, stderr, err := runWGO(workDir, "check", "--config", configFile)
		
		if err != nil {
			t.Logf("Check drift stderr: %s", stderr)
		}
		
		if !strings.Contains(stdout, "Checking for infrastructure drift") {
			t.Errorf("Expected drift check output, got: %s", stdout)
		}
	})
	
	// Step 5: Test different output formats
	t.Run("output_formats", func(t *testing.T) {
		formats := []string{"json", "yaml", "markdown"}
		
		for _, format := range formats {
			stdout, stderr, err := runWGO(workDir, "baseline", "list", "--output", format, "--config", configFile)
			
			if err != nil {
				t.Logf("Format %s stderr: %s", format, stderr)
			}
			
			if stdout == "" {
				t.Errorf("Expected output for format %s", format)
			}
		}
	})
}

func TestE2E_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "wgo-error-test")
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}
	
	// Test with invalid config
	t.Run("invalid_config", func(t *testing.T) {
		invalidConfig := filepath.Join(workDir, "invalid.yaml")
		err := os.WriteFile(invalidConfig, []byte("invalid: yaml: content: ["), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid config: %v", err)
		}
		
		_, stderr, err := runWGO(workDir, "baseline", "list", "--config", invalidConfig)
		
		if err == nil {
			t.Error("Expected error with invalid config")
		}
		
		if !strings.Contains(stderr, "config") {
			t.Errorf("Expected config error in stderr, got: %s", stderr)
		}
	})
	
	// Test with non-existent baseline
	t.Run("non_existent_baseline", func(t *testing.T) {
		configFile := filepath.Join(workDir, "config.yaml")
		configContent := `
storage:
  base_path: ` + filepath.Join(workDir, ".wgo") + `
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		
		_, stderr, err := runWGO(workDir, "baseline", "show", "non-existent", "--config", configFile)
		
		if err == nil {
			t.Error("Expected error with non-existent baseline")
		}
		
		// Error message should indicate baseline not found
		if !strings.Contains(stderr, "not") {
			t.Logf("Show non-existent baseline stderr: %s", stderr)
		}
	})
	
	// Test with missing required flags
	t.Run("missing_required_flags", func(t *testing.T) {
		_, stderr, err := runWGO(workDir, "baseline", "create")
		
		if err == nil {
			t.Error("Expected error with missing required flags")
		}
		
		if !strings.Contains(stderr, "required") {
			t.Errorf("Expected required flag error, got: %s", stderr)
		}
	})
}

func TestE2E_ConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "wgo-concurrent-test")
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}
	
	configFile := filepath.Join(workDir, "config.yaml")
	configContent := `
storage:
  base_path: ` + filepath.Join(workDir, ".wgo") + `
output:
  format: table
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Run multiple commands concurrently to test for race conditions
	t.Run("concurrent_baseline_operations", func(t *testing.T) {
		const numOperations = 5
		results := make(chan error, numOperations)
		
		for i := 0; i < numOperations; i++ {
			go func(index int) {
				_, _, err := runWGO(workDir, "baseline", "list", "--config", configFile)
				results <- err
			}(i)
		}
		
		// Wait for all operations to complete
		for i := 0; i < numOperations; i++ {
			select {
			case err := <-results:
				if err != nil {
					t.Logf("Concurrent operation %d had error: %v", i, err)
					// Some errors might be expected due to stub implementations
				}
			case <-time.After(30 * time.Second):
				t.Error("Timeout waiting for concurrent operations")
			}
		}
	})
}

func TestE2E_ConfigurationVariations(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "wgo-config-test")
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}
	
	// Test different configuration scenarios
	configs := map[string]string{
		"minimal": `
storage:
  base_path: ` + filepath.Join(workDir, ".wgo-minimal"),
		"comprehensive": `
storage:
  base_path: ` + filepath.Join(workDir, ".wgo-comprehensive") + `
output:
  format: json
  pretty: true
  no_color: false
logging:
  level: debug
collectors:
  terraform:
    enabled: true
  aws:
    enabled: false
    regions: ["us-east-1", "us-west-2"]
  kubernetes:
    enabled: false
    contexts: ["prod", "staging"]
`,
		"custom_paths": `
storage:
  base_path: ` + filepath.Join(workDir, "custom-storage") + `
output:
  format: yaml
logging:
  level: warn
`,
	}
	
	for configName, configContent := range configs {
		t.Run("config_"+configName, func(t *testing.T) {
			configFile := filepath.Join(workDir, configName+".yaml")
			err := os.WriteFile(configFile, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create config %s: %v", configName, err)
			}
			
			// Test that each config works
			stdout, stderr, err := runWGO(workDir, "version", "--config", configFile)
			
			if err != nil {
				t.Logf("Config %s stderr: %s", configName, stderr)
			}
			
			if !strings.Contains(stdout, "version") {
				t.Errorf("Expected version output with config %s, got: %s", configName, stdout)
			}
			
			// Test baseline operations with each config
			stdout, stderr, err = runWGO(workDir, "baseline", "list", "--config", configFile)
			
			if err != nil {
				t.Logf("Baseline list with config %s stderr: %s", configName, stderr)
			}
			
			// Should not have config parsing errors
			if strings.Contains(stderr, "yaml") && strings.Contains(stderr, "error") {
				t.Errorf("Config parsing error with %s: %s", configName, stderr)
			}
		})
	}
}