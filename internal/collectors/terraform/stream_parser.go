package terraform

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// StreamingParser handles large Terraform state files efficiently
type StreamingParser struct {
	maxMemory int64 // Maximum memory to use (bytes)
}

// NewStreamingParser creates a new streaming parser
func NewStreamingParser() *StreamingParser {
	return &StreamingParser{
		maxMemory: 100 * 1024 * 1024, // 100MB default
	}
}

// ParseStateFile parses a Terraform state file with streaming for large files
func (sp *StreamingParser) ParseStateFile(filename string) (*TerraformState, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("terraform state file not found at %s. Run 'terraform init' first or check the file path", filename)
		}
		return nil, fmt.Errorf("failed to open terraform state file %s: %w", filename, err)
	}
	defer file.Close()

	// Check file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", filename, err)
	}

	fileSize := stat.Size()

	// For very large files, use streaming parser
	if fileSize > sp.maxMemory {
		return sp.parseStreamingMode(file, filename, fileSize)
	}

	// For smaller files, use standard JSON parsing
	return sp.parseStandardMode(file, filename)
}

// parseStandardMode handles normal-sized files with standard JSON parsing
func (sp *StreamingParser) parseStandardMode(file *os.File, filename string) (*TerraformState, error) {
	var state TerraformState

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&state); err != nil {
		return nil, sp.createJSONError(err, filename)
	}

	return sp.validateAndNormalizeState(&state, filename)
}

// parseStreamingMode handles large files with true streaming JSON parsing
func (sp *StreamingParser) parseStreamingMode(file *os.File, filename string, fileSize int64) (*TerraformState, error) {
	fmt.Printf("Using streaming parser for large state file (%d MB)\n", fileSize/(1024*1024))

	// Reset file to beginning
	file.Seek(0, 0)

	// Use incremental parsing to handle large files
	state, err := sp.parseIncrementally(file, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse large state file %s: %w", filename, err)
	}

	return sp.validateAndNormalizeState(state, filename)
}

// parseIncrementally uses json.Decoder for incremental parsing
func (sp *StreamingParser) parseIncrementally(reader io.Reader, filename string) (*TerraformState, error) {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber() // Preserve number precision

	state := &TerraformState{}

	// Start decoding the JSON object
	token, err := decoder.Token()
	if err != nil {
		return nil, sp.createJSONError(err, filename)
	}

	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected '{' at start of state file %s", filename)
	}

	// Parse the state file field by field
	for decoder.More() {
		// Get the field name
		token, err := decoder.Token()
		if err != nil {
			return nil, sp.createJSONError(err, filename)
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		switch key {
		case "version":
			if err := decoder.Decode(&state.Version); err != nil {
				return nil, fmt.Errorf("failed to parse version in %s: %w", filename, err)
			}

		case "terraform_version":
			if err := decoder.Decode(&state.TerraformVersion); err != nil {
				return nil, fmt.Errorf("failed to parse terraform_version in %s: %w", filename, err)
			}

		case "serial":
			if err := decoder.Decode(&state.Serial); err != nil {
				return nil, fmt.Errorf("failed to parse serial in %s: %w", filename, err)
			}

		case "lineage":
			if err := decoder.Decode(&state.Lineage); err != nil {
				return nil, fmt.Errorf("failed to parse lineage in %s: %w", filename, err)
			}

		case "outputs":
			if err := decoder.Decode(&state.Outputs); err != nil {
				return nil, fmt.Errorf("failed to parse outputs in %s: %w", filename, err)
			}

		case "resources":
			// Stream parse resources array
			resources, err := sp.streamParseResources(decoder)
			if err != nil {
				return nil, fmt.Errorf("failed to parse resources in %s: %w", filename, err)
			}
			state.Resources = resources

		case "modules":
			// For version 3 compatibility
			if err := decoder.Decode(&state.Modules); err != nil {
				return nil, fmt.Errorf("failed to parse modules in %s: %w", filename, err)
			}

		default:
			// Skip unknown fields by decoding into raw message
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				return nil, fmt.Errorf("failed to skip field %s in %s: %w", key, filename, err)
			}
		}
	}

	// Consume closing brace
	token, err = decoder.Token()
	if err != nil {
		return nil, sp.createJSONError(err, filename)
	}

	if delim, ok := token.(json.Delim); !ok || delim != '}' {
		return nil, fmt.Errorf("expected '}' at end of state file %s", filename)
	}

	return state, nil
}

// streamParseResources parses the resources array incrementally
func (sp *StreamingParser) streamParseResources(decoder *json.Decoder) ([]TerraformResource, error) {
	var resources []TerraformResource

	// Expect array start
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return nil, fmt.Errorf("expected '[' at start of resources array")
	}

	// Parse each resource
	resourceCount := 0
	for decoder.More() {
		var resource TerraformResource
		if err := decoder.Decode(&resource); err != nil {
			return nil, fmt.Errorf("failed to parse resource %d: %w", resourceCount, err)
		}
		resources = append(resources, resource)
		resourceCount++

		// Progress feedback for large files
		if resourceCount%100 == 0 {
			fmt.Printf("  Terraform: Parsed %d resources...\n", resourceCount)
		}
	}

	// Expect array end
	token, err = decoder.Token()
	if err != nil {
		return nil, err
	}

	if delim, ok := token.(json.Delim); !ok || delim != ']' {
		return nil, fmt.Errorf("expected ']' at end of resources array")
	}

	if resourceCount > 0 {
		fmt.Printf("  Terraform: Successfully parsed %d resources\n", resourceCount)
	}

	return resources, nil
}

// parseInChunks is a legacy method kept for backward compatibility
func (sp *StreamingParser) parseInChunks(reader *bufio.Reader, filename string) (*TerraformState, error) {
	// Redirect to the new incremental parser
	return sp.parseIncrementally(reader, filename)
}

// validateAndNormalizeState performs validation and basic normalization
func (sp *StreamingParser) validateAndNormalizeState(state *TerraformState, filename string) (*TerraformState, error) {
	// Validate required fields
	if state.Version == 0 {
		return nil, fmt.Errorf("invalid terraform state file %s: missing or invalid 'version' field", filename)
	}

	// Validate Terraform version format
	if state.TerraformVersion == "" {
		return nil, fmt.Errorf("invalid terraform state file %s: missing 'terraform_version' field", filename)
	}

	// Check for supported versions
	if state.Version < 3 || state.Version > 4 {
		return nil, fmt.Errorf("unsupported terraform state version %d in %s. Supported versions: 3, 4", state.Version, filename)
	}

	// Validate structure based on version
	if state.Version == 4 && state.Resources == nil {
		return nil, fmt.Errorf("invalid terraform state file %s: version 4 requires 'resources' array", filename)
	}

	if state.Version == 3 && state.Modules == nil {
		return nil, fmt.Errorf("invalid terraform state file %s: version 3 requires 'modules' array", filename)
	}

	return state, nil
}

// createJSONError creates helpful error messages for JSON parsing errors
func (sp *StreamingParser) createJSONError(err error, filename string) error {
	errStr := err.Error()

	// Common JSON error patterns and helpful messages
	switch {
	case strings.Contains(errStr, "unexpected end of JSON"):
		return fmt.Errorf("terraform state file %s appears to be truncated or corrupted. Try regenerating with 'terraform refresh'", filename)

	case strings.Contains(errStr, "invalid character"):
		return fmt.Errorf("terraform state file %s contains invalid JSON. Check for corruption or manual edits", filename)

	case strings.Contains(errStr, "cannot unmarshal"):
		return fmt.Errorf("terraform state file %s has unexpected structure. This may be from an unsupported Terraform version", filename)

	default:
		return fmt.Errorf("failed to parse terraform state file %s: %w", filename, err)
	}
}

// ResourceCounter provides streaming resource counting for large files
type ResourceCounter struct {
	totalResources int
	resourceTypes  map[string]int
}

// CountResources counts resources without loading the entire file into memory
func (sp *StreamingParser) CountResources(filename string) (*ResourceCounter, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file for counting: %w", err)
	}
	defer file.Close()

	counter := &ResourceCounter{
		resourceTypes: make(map[string]int),
	}

	// Use streaming JSON decoder
	decoder := json.NewDecoder(file)
	decoder.UseNumber()

	// Navigate to resources array
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse state file: %w", err)
		}

		// Look for "resources" key
		if key, ok := token.(string); ok && key == "resources" {
			// Next token should be array start
			token, err := decoder.Token()
			if err != nil {
				return nil, err
			}

			if delim, ok := token.(json.Delim); ok && delim == '[' {
				// Count resources in the array
				for decoder.More() {
					var resource struct {
						Type      string          `json:"type"`
						Instances json.RawMessage `json:"instances"`
					}

					if err := decoder.Decode(&resource); err != nil {
						continue // Skip problematic resources
					}

					// Count instances
					if len(resource.Instances) > 0 {
						// Parse instances array to count them
						var instances []json.RawMessage
						if err := json.Unmarshal(resource.Instances, &instances); err == nil {
							instanceCount := len(instances)
							counter.totalResources += instanceCount
							counter.resourceTypes[resource.Type] += instanceCount
						}
					}
				}
				break
			}
		}
	}

	return counter, nil
}

// ValidateStateFile performs basic validation without full parsing
func (sp *StreamingParser) ValidateStateFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("terraform state file not found at %s. Run 'terraform init' first or check the file path", filename)
		}
		return fmt.Errorf("cannot access terraform state file %s: %w", filename, err)
	}
	defer file.Close()

	// Check if file is empty
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot get file info for %s: %w", filename, err)
	}

	if stat.Size() == 0 {
		return fmt.Errorf("terraform state file %s is empty. Run 'terraform apply' to create resources", filename)
	}

	// Check if it's valid JSON by reading just the beginning
	reader := bufio.NewReader(file)
	firstByte, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("cannot read terraform state file %s: %w", filename, err)
	}

	if firstByte != '{' {
		return fmt.Errorf("terraform state file %s is not valid JSON (does not start with '{')", filename)
	}

	// Try to parse just the version field
	file.Seek(0, 0) // Reset to beginning
	var versionCheck struct {
		Version int `json:"version"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&versionCheck); err != nil {
		return fmt.Errorf("terraform state file %s contains invalid JSON or unsupported format: %w", filename, err)
	}

	if versionCheck.Version < 3 || versionCheck.Version > 4 {
		return fmt.Errorf("unsupported terraform state version %d in %s. Supported versions: 3, 4", versionCheck.Version, filename)
	}

	return nil
}
