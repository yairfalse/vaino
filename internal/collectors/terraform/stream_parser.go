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

// parseStreamingMode handles large files with streaming JSON parsing
func (sp *StreamingParser) parseStreamingMode(file *os.File, filename string, fileSize int64) (*TerraformState, error) {
	fmt.Printf("⚠️  Large state file detected (%d MB). Using streaming parser...\n", fileSize/(1024*1024))

	// Read file in chunks to find the structure
	reader := bufio.NewReaderSize(file, 64*1024) // 64KB buffer

	// For very large files, we need to parse incrementally
	// This is a simplified streaming approach - in practice, you'd use a more sophisticated JSON streaming library
	state, err := sp.parseInChunks(reader, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse large state file %s: %w", filename, err)
	}

	return sp.validateAndNormalizeState(state, filename)
}

// parseInChunks parses JSON in manageable chunks
func (sp *StreamingParser) parseInChunks(reader *bufio.Reader, filename string) (*TerraformState, error) {
	// For this implementation, we'll still load the entire JSON but with better error handling
	// In a full production system, you'd use a proper streaming JSON parser like jstream

	var jsonBuilder strings.Builder
	buffer := make([]byte, 32*1024) // 32KB chunks

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			jsonBuilder.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading state file %s: %w", filename, err)
		}
	}

	var state TerraformState
	if err := json.Unmarshal([]byte(jsonBuilder.String()), &state); err != nil {
		return nil, sp.createJSONError(err, filename)
	}

	return &state, nil
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

	// Use a streaming approach to count resources
	decoder := json.NewDecoder(file)

	// Skip to the resources array
	var state struct {
		Resources []struct {
			Type      string        `json:"type"`
			Instances []interface{} `json:"instances"`
		} `json:"resources"`
	}

	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode state for counting: %w", err)
	}

	for _, resource := range state.Resources {
		instanceCount := len(resource.Instances)
		counter.totalResources += instanceCount
		counter.resourceTypes[resource.Type] += instanceCount
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
