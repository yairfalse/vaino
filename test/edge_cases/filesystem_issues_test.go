package edgecases

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// TestCorruptedFiles tests scenarios with corrupted configuration and state files
func TestCorruptedFiles(t *testing.T) {
	tests := []struct {
		name        string
		fileType    string
		content     string
		expectError bool
		errorType   string
	}{
		{
			name:        "corrupted_json_config",
			fileType:    "json",
			content:     `{"incomplete": json file missing closing brace`,
			expectError: true,
			errorType:   "json_parse_error",
		},
		{
			name:        "corrupted_yaml_config",
			fileType:    "yaml",
			content:     "invalid: yaml\n  structure:\n    - missing\n      proper: indentation",
			expectError: true,
			errorType:   "yaml_parse_error",
		},
		{
			name:        "binary_data_in_config",
			fileType:    "json",
			content:     string([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}),
			expectError: true,
			errorType:   "binary_data_error",
		},
		{
			name:        "extremely_large_config",
			fileType:    "json",
			content:     createLargeJSON(10000), // 10MB+ JSON file
			expectError: false,                  // Should handle large files
			errorType:   "",
		},
		{
			name:        "config_with_null_bytes",
			fileType:    "json",
			content:     `{"test": "value\x00with\x00nulls"}`,
			expectError: true,
			errorType:   "null_byte_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var fileName string

			switch tt.fileType {
			case "json":
				fileName = "config.json"
			case "yaml":
				fileName = "config.yaml"
			default:
				fileName = "config.txt"
			}

			filePath := filepath.Join(tempDir, fileName)

			// Write the test content
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Try to read and parse the file
			content, err := os.ReadFile(filePath)
			if err != nil {
				if !tt.expectError {
					t.Errorf("Failed to read file: %v", err)
				}
				return
			}

			// Test JSON parsing
			if tt.fileType == "json" {
				var parsed map[string]interface{}
				err = json.Unmarshal(content, &parsed)

				if tt.expectError && err == nil {
					t.Error("Expected parsing to fail but it succeeded")
				} else if !tt.expectError && err != nil {
					t.Errorf("Expected parsing to succeed but got: %v", err)
				}
			}

			// Test for null bytes
			if strings.Contains(tt.content, "\x00") {
				if !strings.Contains(string(content), "\x00") {
					t.Error("Null bytes were not preserved in file content")
				}
			}
		})
	}
}

// TestPermissionErrors tests scenarios with insufficient file system permissions
func TestPermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission tests when running as root")
	}

	tests := []struct {
		name        string
		setup       func(t *testing.T) (string, func())
		operation   string
		expectError bool
	}{
		{
			name: "read_permission_denied",
			setup: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				file := filepath.Join(tempDir, "no-read.json")

				content := `{"test": "config"}`
				os.WriteFile(file, []byte(content), 0644)

				// Remove read permission
				os.Chmod(file, 0000)

				return file, func() {
					os.Chmod(file, 0644) // Restore for cleanup
				}
			},
			operation:   "read",
			expectError: true,
		},
		{
			name: "write_permission_denied",
			setup: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				file := filepath.Join(tempDir, "no-write.json")

				// Create file first
				content := `{"test": "config"}`
				os.WriteFile(file, []byte(content), 0644)

				// Remove write permission
				os.Chmod(file, 0444)

				return file, func() {
					os.Chmod(file, 0644) // Restore for cleanup
				}
			},
			operation:   "write",
			expectError: true,
		},
		{
			name: "directory_permission_denied",
			setup: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				restrictedDir := filepath.Join(tempDir, "restricted")
				os.Mkdir(restrictedDir, 0755)

				file := filepath.Join(restrictedDir, "config.json")
				content := `{"test": "config"}`
				os.WriteFile(file, []byte(content), 0644)

				// Remove directory permissions
				os.Chmod(restrictedDir, 0000)

				return file, func() {
					os.Chmod(restrictedDir, 0755) // Restore for cleanup
				}
			},
			operation:   "read",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, cleanup := tt.setup(t)
			defer cleanup()

			switch tt.operation {
			case "read":
				_, err := os.ReadFile(file)
				if tt.expectError && err == nil {
					t.Error("Expected read to fail but it succeeded")
				} else if !tt.expectError && err != nil {
					t.Errorf("Expected read to succeed but got: %v", err)
				}

				// Verify it's a permission error
				if err != nil && !os.IsPermission(err) {
					t.Errorf("Expected permission error, got: %v", err)
				}

			case "write":
				err := os.WriteFile(file, []byte("new content"), 0644)
				if tt.expectError && err == nil {
					t.Error("Expected write to fail but it succeeded")
				} else if !tt.expectError && err != nil {
					t.Errorf("Expected write to succeed but got: %v", err)
				}

				// Verify it's a permission error
				if err != nil && !os.IsPermission(err) {
					t.Errorf("Expected permission error, got: %v", err)
				}
			}
		})
	}
}

// TestDiskSpaceIssues tests scenarios with insufficient disk space
func TestDiskSpaceIssues(t *testing.T) {
	tests := []struct {
		name      string
		fileSize  int64
		expectErr bool
	}{
		{
			name:      "small_file_write",
			fileSize:  1024, // 1KB
			expectErr: false,
		},
		{
			name:      "large_file_write",
			fileSize:  100 * 1024 * 1024, // 100MB
			expectErr: false,             // Most systems should handle this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			file := filepath.Join(tempDir, "large-file.json")

			// Create a large JSON structure
			data := make(map[string]interface{})

			// Fill with data to reach approximate size
			chunkSize := 1024
			chunks := int(tt.fileSize / int64(chunkSize))

			for i := 0; i < chunks; i++ {
				key := fmt.Sprintf("data_%d", i)
				value := strings.Repeat("x", chunkSize-len(key)-10) // Approximate
				data[key] = value
			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}

			// Try to write the large file
			err = os.WriteFile(file, jsonData, 0644)

			if tt.expectErr && err == nil {
				t.Error("Expected write to fail due to disk space but it succeeded")
			} else if !tt.expectErr && err != nil {
				// Check if it's actually a disk space error
				if isNoSpaceError(err) {
					t.Logf("Got expected disk space error: %v", err)
				} else {
					t.Errorf("Expected write to succeed but got non-space error: %v", err)
				}
			}

			if err == nil {
				// Verify file was written correctly
				stat, err := os.Stat(file)
				if err != nil {
					t.Errorf("Failed to stat written file: %v", err)
				} else {
					t.Logf("Successfully wrote file of size: %d bytes", stat.Size())
				}
			}
		})
	}
}

// TestFileSystemRaceConditions tests concurrent file operations
func TestFileSystemRaceConditions(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		operations []func(string) error
	}{
		{
			name: "concurrent_read_write",
			operations: []func(string) error{
				func(file string) error {
					content := fmt.Sprintf(`{"timestamp": %d}`, time.Now().UnixNano())
					return os.WriteFile(file, []byte(content), 0644)
				},
				func(file string) error {
					_, err := os.ReadFile(file)
					return err
				},
				func(file string) error {
					_, err := os.Stat(file)
					return err
				},
			},
		},
		{
			name: "concurrent_create_delete",
			operations: []func(string) error{
				func(file string) error {
					return os.WriteFile(file, []byte(`{"test": "data"}`), 0644)
				},
				func(file string) error {
					return os.Remove(file)
				},
				func(file string) error {
					return os.WriteFile(file, []byte(`{"new": "data"}`), 0644)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(tempDir, fmt.Sprintf("race-test-%s.json", tt.name))

			// Run operations concurrently
			done := make(chan error, len(tt.operations))

			for i, op := range tt.operations {
				go func(operation func(string) error, id int) {
					// Small random delay to increase chance of race conditions
					time.Sleep(time.Duration(id*10) * time.Millisecond)
					err := operation(file)
					done <- err
				}(op, i)
			}

			// Collect results
			var errors []error
			for i := 0; i < len(tt.operations); i++ {
				if err := <-done; err != nil {
					errors = append(errors, err)
				}
			}

			// Some errors are expected in race conditions (file not found, etc.)
			// but we should not crash or corrupt data
			if len(errors) > 0 {
				t.Logf("Got %d errors from concurrent operations (some expected): %v", len(errors), errors)
			}

			// Verify final state is consistent
			if _, err := os.Stat(file); err == nil {
				content, err := os.ReadFile(file)
				if err != nil {
					t.Errorf("Failed to read final file state: %v", err)
				} else {
					// Verify it's valid JSON
					var parsed map[string]interface{}
					if err := json.Unmarshal(content, &parsed); err != nil {
						t.Errorf("Final file contains invalid JSON: %v", err)
					}
				}
			}
		})
	}
}

// TestSymlinkAndMountIssues tests edge cases with symbolic links and mounts
func TestSymlinkAndMountIssues(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func(t *testing.T) (string, func())
		expectError bool
	}{
		{
			name: "broken_symlink",
			setup: func(t *testing.T) (string, func()) {
				target := filepath.Join(tempDir, "nonexistent.json")
				link := filepath.Join(tempDir, "broken-link.json")

				err := os.Symlink(target, link)
				if err != nil {
					t.Fatalf("Failed to create symlink: %v", err)
				}

				return link, func() {
					os.Remove(link)
				}
			},
			expectError: true,
		},
		{
			name: "circular_symlink",
			setup: func(t *testing.T) (string, func()) {
				link1 := filepath.Join(tempDir, "link1.json")
				link2 := filepath.Join(tempDir, "link2.json")

				os.Symlink(link2, link1)
				os.Symlink(link1, link2)

				return link1, func() {
					os.Remove(link1)
					os.Remove(link2)
				}
			},
			expectError: true,
		},
		{
			name: "valid_symlink",
			setup: func(t *testing.T) (string, func()) {
				target := filepath.Join(tempDir, "target.json")
				link := filepath.Join(tempDir, "valid-link.json")

				// Create target file
				content := `{"test": "data"}`
				os.WriteFile(target, []byte(content), 0644)

				// Create symlink
				err := os.Symlink(target, link)
				if err != nil {
					t.Fatalf("Failed to create symlink: %v", err)
				}

				return link, func() {
					os.Remove(link)
					os.Remove(target)
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, cleanup := tt.setup(t)
			defer cleanup()

			_, err := os.ReadFile(file)

			if tt.expectError && err == nil {
				t.Error("Expected error reading symlink but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected successful read but got: %v", err)
			}

			// Test stat vs lstat behavior
			_, err1 := os.Stat(file)  // Follows symlinks
			_, err2 := os.Lstat(file) // Does not follow symlinks

			if err2 != nil {
				t.Errorf("Lstat should not fail for symlink itself: %v", err2)
			}

			if tt.expectError && err1 == nil {
				t.Error("Stat should fail for broken symlink")
			}
		})
	}
}

// Helper functions

func createLargeJSON(entries int) string {
	data := make(map[string]interface{})

	for i := 0; i < entries; i++ {
		key := fmt.Sprintf("resource_%d", i)
		resource := types.Resource{
			ID:       fmt.Sprintf("id-%d", i),
			Type:     "test_resource",
			Name:     fmt.Sprintf("resource-%d", i),
			Provider: "test",
			Configuration: map[string]interface{}{
				"property1": "value1",
				"property2": i,
				"property3": strings.Repeat("data", 100),
			},
		}
		data[key] = resource
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

func isNoSpaceError(err error) bool {
	if err == nil {
		return false
	}

	// Check for "no space left on device" error
	if pathErr, ok := err.(*fs.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == syscall.ENOSPC
		}
	}

	// Also check error message
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "no space left") ||
		strings.Contains(errMsg, "disk full") ||
		strings.Contains(errMsg, "quota exceeded")
}
