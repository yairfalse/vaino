package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TerraformDiscovery handles auto-discovery of Terraform state files
type TerraformDiscovery struct {
	maxDepth int
	maxFiles int
}

// NewTerraformDiscovery creates a new Terraform discovery service
func NewTerraformDiscovery() *TerraformDiscovery {
	return &TerraformDiscovery{
		maxDepth: 5, // Don't go too deep to avoid performance issues
		maxFiles: 50, // Limit number of files to avoid overwhelming
	}
}

// StateFile represents a discovered Terraform state file
type StateFile struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Size         int64  `json:"size"`
	ResourceCount int   `json:"resource_count,omitempty"`
}

// DiscoverStateFiles finds all Terraform state files in the current directory and subdirectories
func (td *TerraformDiscovery) DiscoverStateFiles(rootPath string) ([]StateFile, error) {
	if rootPath == "" {
		var err error
		rootPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	var stateFiles []StateFile

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't read
			return nil
		}

		// Skip if we've reached max files
		if len(stateFiles) >= td.maxFiles {
			return nil
		}

		// Calculate depth
		relPath, _ := filepath.Rel(rootPath, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > td.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories and non-tfstate files
		if info.IsDir() || !td.isTerraformStateFile(path) {
			return nil
		}

		// Skip if file is too small (likely empty)
		if info.Size() < 10 {
			return nil
		}

		// Skip common non-state patterns
		if td.shouldSkipFile(path) {
			return nil
		}

		stateFile := StateFile{
			Path:         path,
			RelativePath: relPath,
			Size:         info.Size(),
		}

		// Try to get resource count from state file
		if count, err := td.getResourceCount(path); err == nil {
			stateFile.ResourceCount = count
		}

		stateFiles = append(stateFiles, stateFile)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Sort by preference (main files first, then by path)
	td.sortStateFiles(stateFiles)

	return stateFiles, nil
}

// isTerraformStateFile checks if a file is a Terraform state file
func (td *TerraformDiscovery) isTerraformStateFile(path string) bool {
	filename := filepath.Base(path)
	
	// Check for .tfstate extension
	if strings.HasSuffix(filename, ".tfstate") {
		return true
	}
	
	// Check for .tfstate.backup extension
	if strings.HasSuffix(filename, ".tfstate.backup") {
		return true
	}

	return false
}

// shouldSkipFile determines if a file should be skipped based on patterns
func (td *TerraformDiscovery) shouldSkipFile(path string) bool {
	filename := filepath.Base(path)
	
	// Skip temporary files
	if strings.HasPrefix(filename, ".terraform.tfstate.") {
		return true
	}
	
	// Skip lock files
	if strings.Contains(filename, ".lock") {
		return true
	}
	
	// Skip hidden directories (but allow .terraform)
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != ".terraform" {
			return true
		}
	}

	return false
}

// getResourceCount attempts to count resources in a state file
func (td *TerraformDiscovery) getResourceCount(path string) (int, error) {
	// This is a simple count - could be improved with proper JSON parsing
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Simple heuristic: count occurrences of "resources"
	contentStr := string(content)
	if strings.Contains(contentStr, `"resources"`) {
		// Count resource objects
		count := strings.Count(contentStr, `"mode": "managed"`)
		if count > 0 {
			return count, nil
		}
		
		// Fallback: count resource blocks
		count = strings.Count(contentStr, `"type":`)
		if count > 0 {
			return count, nil
		}
	}

	return 0, nil
}

// sortStateFiles sorts state files by preference
func (td *TerraformDiscovery) sortStateFiles(files []StateFile) {
	// Priority order:
	// 1. terraform.tfstate in root
	// 2. *.tfstate in root  
	// 3. terraform.tfstate in subdirs
	// 4. Others by path
	
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if td.compareStateFiles(files[i], files[j]) > 0 {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

// compareStateFiles returns -1 if a should come before b, 1 if after, 0 if equal
func (td *TerraformDiscovery) compareStateFiles(a, b StateFile) int {
	aScore := td.getFileScore(a)
	bScore := td.getFileScore(b)
	
	if aScore != bScore {
		return bScore - aScore // Higher score comes first
	}
	
	// If scores are equal, sort by path
	if a.RelativePath < b.RelativePath {
		return -1
	} else if a.RelativePath > b.RelativePath {
		return 1
	}
	
	return 0
}

// getFileScore assigns a priority score to a state file
func (td *TerraformDiscovery) getFileScore(file StateFile) int {
	filename := filepath.Base(file.RelativePath)
	dir := filepath.Dir(file.RelativePath)
	
	score := 0
	
	// Prefer terraform.tfstate
	if filename == "terraform.tfstate" {
		score += 1000
	}
	
	// Prefer root directory
	if dir == "." {
		score += 500
	}
	
	// Prefer main terraform directories
	if strings.Contains(dir, "terraform") {
		score += 100
	}
	
	// Prefer files with more resources
	score += file.ResourceCount
	
	// Prefer non-backup files
	if !strings.Contains(filename, "backup") {
		score += 50
	}
	
	// Prefer larger files (more likely to have content)
	if file.Size > 1000 {
		score += 10
	}
	
	return score
}

// GetPreferredStateFiles returns the most relevant state files up to maxFiles
func (td *TerraformDiscovery) GetPreferredStateFiles(files []StateFile, maxFiles int) []StateFile {
	if len(files) <= maxFiles {
		return files
	}
	
	return files[:maxFiles]
}

// AutoDiscoverTerraformFiles is a convenience function for discovery
func AutoDiscoverTerraformFiles() ([]string, error) {
	discovery := NewTerraformDiscovery()
	stateFiles, err := discovery.DiscoverStateFiles("")
	if err != nil {
		return nil, err
	}
	
	var paths []string
	for _, file := range stateFiles {
		paths = append(paths, file.Path)
	}
	
	return paths, nil
}

// AutoDiscoverWithDetails returns detailed information about discovered files
func AutoDiscoverWithDetails() ([]StateFile, error) {
	discovery := NewTerraformDiscovery()
	return discovery.DiscoverStateFiles("")
}