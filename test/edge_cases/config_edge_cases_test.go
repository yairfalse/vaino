package edgecases

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"gopkg.in/yaml.v3"
)

// TestInvalidYAMLStructures tests various malformed YAML configurations
func TestInvalidYAMLStructures(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		errorType   string
	}{
		{
			name: "missing_closing_bracket",
			yamlContent: `
storage:
  base_path: /tmp/wgo
  [missing_bracket: true
output:
  format: table`,
			expectError: true,
			errorType:   "yaml_syntax_error",
		},
		{
			name: "inconsistent_indentation",
			yamlContent: `
storage:
  base_path: /tmp/wgo
    type: local
 retention_days: 30
output:
  format: table`,
			expectError: true,
			errorType:   "yaml_indentation_error",
		},
		{
			name:        "tabs_instead_of_spaces",
			yamlContent: "storage:\n\tbase_path: /tmp/wgo\n\ttype: local\noutput:\n\tformat: table",
			expectError: true,
			errorType:   "yaml_tab_error",
		},
		{
			name: "duplicate_keys",
			yamlContent: `
storage:
  base_path: /tmp/wgo
  type: local
storage:
  base_path: /other/path
output:
  format: table`,
			expectError: false, // YAML allows duplicate keys (last one wins)
			errorType:   "",
		},
		{
			name: "unquoted_special_chars",
			yamlContent: `
storage:
  base_path: /tmp/wgo@#$%^&*()
  special_key: value:with:colons
output:
  format: @table`,
			expectError: false, // Most special chars are fine unquoted
			errorType:   "",
		},
		{
			name: "extremely_nested_structure",
			yamlContent: `
level1:
  level2:
    level3:
      level4:
        level5:
          level6:
            level7:
              level8:
                level9:
                  level10:
                    deep_value: "found"`,
			expectError: false,
			errorType:   "",
		},
		{
			name:        "binary_data_in_yaml",
			yamlContent: "storage:\n  base_path: \x00\x01\x02\x03",
			expectError: true,
			errorType:   "binary_data_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")

			err := os.WriteFile(configFile, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Try to parse the YAML
			content, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			var config map[string]interface{}
			err = yaml.Unmarshal(content, &config)

			if tt.expectError && err == nil {
				t.Error("Expected YAML parsing to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected YAML parsing to succeed but got: %v", err)
			}

			if err != nil {
				t.Logf("Got expected YAML error: %v", err)
			}
		})
	}
}

// TestMissingRequiredFields tests scenarios where required configuration fields are missing
func TestMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		expectError  bool
		missingField string
	}{
		{
			name: "missing_storage_config",
			config: map[string]interface{}{
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError:  true,
			missingField: "storage",
		},
		{
			name: "missing_storage_base_path",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"type": "local",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError:  true,
			missingField: "storage.base_path",
		},
		{
			name: "missing_output_format",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
					"type":      "local",
				},
				"output": map[string]interface{}{
					"pretty": true,
				},
			},
			expectError:  true,
			missingField: "output.format",
		},
		{
			name: "valid_minimal_config",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")

			// Convert config to YAML and write
			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal test config: %v", err)
			}

			err = os.WriteFile(configFile, yamlData, 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Read and validate the config
			content, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			var parsedConfig map[string]interface{}
			err = yaml.Unmarshal(content, &parsedConfig)
			if err != nil {
				t.Fatalf("Failed to parse config: %v", err)
			}

			// Validate required fields
			err = validateRequiredFields(parsedConfig)

			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for missing %s but got none", tt.missingField)
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected validation to pass but got: %v", err)
			}
		})
	}
}

// TestInvalidFieldTypes tests scenarios with incorrect field types
func TestInvalidFieldTypes(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		expectError  bool
		invalidField string
	}{
		{
			name: "storage_base_path_as_number",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": 12345,
					"type":      "local",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError:  true,
			invalidField: "storage.base_path",
		},
		{
			name: "output_format_as_boolean",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
				},
				"output": map[string]interface{}{
					"format": true,
				},
			},
			expectError:  true,
			invalidField: "output.format",
		},
		{
			name: "output_pretty_as_string",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
				},
				"output": map[string]interface{}{
					"format": "table",
					"pretty": "yes", // Should be boolean
				},
			},
			expectError:  true,
			invalidField: "output.pretty",
		},
		{
			name: "array_where_object_expected",
			config: map[string]interface{}{
				"storage": []interface{}{"path1", "path2"}, // Should be object
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError:  true,
			invalidField: "storage",
		},
		{
			name: "nested_array_confusion",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
					"options":   []interface{}{1, 2, 3}, // Arrays in wrong place
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError: false, // Might be valid depending on schema
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")

			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal test config: %v", err)
			}

			err = os.WriteFile(configFile, yamlData, 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Read and validate types
			content, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			var parsedConfig map[string]interface{}
			err = yaml.Unmarshal(content, &parsedConfig)
			if err != nil {
				t.Fatalf("Failed to parse config: %v", err)
			}

			err = validateFieldTypes(parsedConfig)

			if tt.expectError && err == nil {
				t.Errorf("Expected type validation error for %s but got none", tt.invalidField)
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected type validation to pass but got: %v", err)
			}
		})
	}
}

// TestSpecialCharactersInFields tests configuration with special characters
func TestSpecialCharactersInFields(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		description string
	}{
		{
			name: "unicode_in_paths",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo/测试/مجلد/папка",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError: false,
			description: "Unicode characters in paths should be supported",
		},
		{
			name: "emoji_in_config",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo/📁test",
					"name":      "storage🚀config",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError: false,
			description: "Emoji characters should be handled gracefully",
		},
		{
			name: "control_characters",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo\t\n\r",
					"notes":     "config\x00with\x01control\x02chars",
				},
			},
			expectError: true,
			description: "Control characters should be rejected",
		},
		{
			name: "extremely_long_values",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path":   strings.Repeat("a", 10000),
					"description": strings.Repeat("This is a very long description. ", 1000),
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError: false,
			description: "Very long strings should be handled",
		},
		{
			name: "special_yaml_characters",
			config: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
					"query":     "find: [*.json, *.yaml] | exclude: {temp: true}",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			expectError: false,
			description: "YAML special characters in values should work when quoted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")

			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal test config: %v", err)
			}

			err = os.WriteFile(configFile, yamlData, 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Try to read and parse
			content, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			var parsedConfig map[string]interface{}
			err = yaml.Unmarshal(content, &parsedConfig)

			if tt.expectError && err == nil {
				t.Errorf("Expected parsing to fail for %s but it succeeded", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected parsing to succeed for %s but got: %v", tt.description, err)
			}

			// Test for control characters specifically
			if strings.Contains(fmt.Sprintf("%v", tt.config), "\x00") {
				if err == nil {
					t.Error("Should have failed to parse config with null bytes")
				}
			}
		})
	}
}

// TestConfigurationOverrides tests complex override scenarios
func TestConfigurationOverrides(t *testing.T) {
	tests := []struct {
		name        string
		baseConfig  map[string]interface{}
		overrides   []map[string]interface{}
		expectError bool
		description string
	}{
		{
			name: "environment_variable_override",
			baseConfig: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
				},
				"output": map[string]interface{}{
					"format": "table",
				},
			},
			overrides: []map[string]interface{}{
				{
					"storage": map[string]interface{}{
						"base_path": "${VAINO_STORAGE_PATH:/tmp/override}",
					},
				},
			},
			expectError: false,
			description: "Environment variable substitution",
		},
		{
			name: "conflicting_types_override",
			baseConfig: map[string]interface{}{
				"storage": map[string]interface{}{
					"base_path": "/tmp/wgo",
				},
			},
			overrides: []map[string]interface{}{
				{
					"storage": "invalid_string", // Conflicts with object type
				},
			},
			expectError: true,
			description: "Type conflict during merge",
		},
		{
			name: "deep_nested_override",
			baseConfig: map[string]interface{}{
				"providers": map[string]interface{}{
					"aws": map[string]interface{}{
						"region": "us-east-1",
						"endpoints": map[string]interface{}{
							"ec2": "https://ec2.amazonaws.com",
						},
					},
				},
			},
			overrides: []map[string]interface{}{
				{
					"providers": map[string]interface{}{
						"aws": map[string]interface{}{
							"endpoints": map[string]interface{}{
								"s3": "https://s3.amazonaws.com",
							},
						},
					},
				},
			},
			expectError: false,
			description: "Deep nested merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Write base config
			baseFile := filepath.Join(tempDir, "base-config.yaml")
			baseData, err := yaml.Marshal(tt.baseConfig)
			if err != nil {
				t.Fatalf("Failed to marshal base config: %v", err)
			}
			os.WriteFile(baseFile, baseData, 0644)

			// Write override configs
			var overrideFiles []string
			for i, override := range tt.overrides {
				overrideFile := filepath.Join(tempDir, fmt.Sprintf("override-%d.yaml", i))
				overrideData, err := yaml.Marshal(override)
				if err != nil {
					t.Fatalf("Failed to marshal override config %d: %v", i, err)
				}
				os.WriteFile(overrideFile, overrideData, 0644)
				overrideFiles = append(overrideFiles, overrideFile)
			}

			// Simulate merging configs
			finalConfig := make(map[string]interface{})

			// Start with base
			for k, v := range tt.baseConfig {
				finalConfig[k] = v
			}

			// Apply overrides
			for _, override := range tt.overrides {
				err := mergeConfig(finalConfig, override)
				if tt.expectError && err == nil {
					t.Errorf("Expected merge error for %s but got none", tt.description)
					return
				} else if !tt.expectError && err != nil {
					t.Errorf("Expected merge to succeed for %s but got: %v", tt.description, err)
					return
				}
			}

			// Validate final result
			if !tt.expectError {
				yamlData, err := yaml.Marshal(finalConfig)
				if err != nil {
					t.Errorf("Failed to marshal final config: %v", err)
				} else {
					t.Logf("Final merged config:\n%s", yamlData)
				}
			}
		})
	}
}

// TestCircularReferences tests configuration files with circular references
func TestCircularReferences(t *testing.T) {
	tempDir := t.TempDir()

	// Create config files that reference each other
	config1 := `
include: config2.yaml
storage:
  base_path: /tmp/wgo1
`

	config2 := `
include: config1.yaml
storage:
  base_path: /tmp/wgo2
`

	file1 := filepath.Join(tempDir, "config1.yaml")
	file2 := filepath.Join(tempDir, "config2.yaml")

	os.WriteFile(file1, []byte(config1), 0644)
	os.WriteFile(file2, []byte(config2), 0644)

	t.Run("detect_circular_reference", func(t *testing.T) {
		// Simulate a config loader that detects circular references
		visited := make(map[string]bool)

		err := checkCircularReference(file1, visited)
		if err == nil {
			t.Error("Expected circular reference detection to fail but it didn't")
		} else {
			t.Logf("Successfully detected circular reference: %v", err)
		}
	})
}

// Helper functions

func validateRequiredFields(config map[string]interface{}) error {
	// Check for storage section
	storage, exists := config["storage"]
	if !exists {
		return fmt.Errorf("missing required field: storage")
	}

	storageMap, ok := storage.(map[string]interface{})
	if !ok {
		return fmt.Errorf("storage must be an object")
	}

	// Check for base_path
	if _, exists := storageMap["base_path"]; !exists {
		return fmt.Errorf("missing required field: storage.base_path")
	}

	// Check for output section
	output, exists := config["output"]
	if !exists {
		return fmt.Errorf("missing required field: output")
	}

	outputMap, ok := output.(map[string]interface{})
	if !ok {
		return fmt.Errorf("output must be an object")
	}

	// Check for format
	if _, exists := outputMap["format"]; !exists {
		return fmt.Errorf("missing required field: output.format")
	}

	return nil
}

func validateFieldTypes(config map[string]interface{}) error {
	// Check storage.base_path is string
	if storage, exists := config["storage"]; exists {
		if storageMap, ok := storage.(map[string]interface{}); ok {
			if basePath, exists := storageMap["base_path"]; exists {
				if _, ok := basePath.(string); !ok {
					return fmt.Errorf("storage.base_path must be a string")
				}
			}
		} else {
			return fmt.Errorf("storage must be an object")
		}
	}

	// Check output.format is string
	if output, exists := config["output"]; exists {
		if outputMap, ok := output.(map[string]interface{}); ok {
			if format, exists := outputMap["format"]; exists {
				if _, ok := format.(string); !ok {
					return fmt.Errorf("output.format must be a string")
				}
			}

			// Check output.pretty is boolean if it exists
			if pretty, exists := outputMap["pretty"]; exists {
				if _, ok := pretty.(bool); !ok {
					return fmt.Errorf("output.pretty must be a boolean")
				}
			}
		}
	}

	return nil
}

func mergeConfig(base, override map[string]interface{}) error {
	for key, value := range override {
		if baseValue, exists := base[key]; exists {
			// Check for type conflicts
			baseType := fmt.Sprintf("%T", baseValue)
			overrideType := fmt.Sprintf("%T", value)

			if baseType != overrideType {
				// Special case: both are maps
				if baseMap, ok := baseValue.(map[string]interface{}); ok {
					if overrideMap, ok := value.(map[string]interface{}); ok {
						// Recursive merge
						return mergeConfig(baseMap, overrideMap)
					}
				}
				return fmt.Errorf("type conflict for key %s: base is %s, override is %s", key, baseType, overrideType)
			}
		}
		base[key] = value
	}
	return nil
}

func checkCircularReference(file string, visited map[string]bool) error {
	if visited[file] {
		return fmt.Errorf("circular reference detected: %s", file)
	}

	visited[file] = true

	// Read file and look for includes (simplified)
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "include:") {
			includedFile := strings.TrimSpace(strings.TrimPrefix(line, "include:"))
			includedFile = filepath.Join(filepath.Dir(file), includedFile)

			if err := checkCircularReference(includedFile, visited); err != nil {
				return err
			}
		}
	}

	delete(visited, file) // Remove from visited when done
	return nil
}

func containsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}
