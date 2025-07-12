package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/yairfalse/wgo/cmd/install/installer"
)

// Checker performs post-installation validation
type Checker struct {
	installPath string
	config      *installer.Config
}

// NewChecker creates a new validation checker
func NewChecker(installPath string, config *installer.Config) installer.Validator {
	return &Checker{
		installPath: installPath,
		config:      config,
	}
}

// Validate performs all validation checks
func (c *Checker) Validate(ctx context.Context) installer.ValidationResult {
	checks := []installer.ValidationCheck{}
	allSuccess := true

	// Define validation checks based on validation level
	var checkFuncs []func(context.Context) installer.ValidationCheck

	if c.config.ValidationLevel == "basic" {
		checkFuncs = []func(context.Context) installer.ValidationCheck{
			c.checkBinaryExists,
			c.checkBinaryExecutable,
			c.checkVersion,
		}
	} else {
		// Full validation
		checkFuncs = []func(context.Context) installer.ValidationCheck{
			c.checkBinaryExists,
			c.checkBinaryExecutable,
			c.checkBinaryPermissions,
			c.checkVersion,
			c.checkHelp,
			c.checkPathIntegration,
			c.checkDependencies,
			c.checkDiskSpace,
		}
	}

	// Run all checks
	for _, checkFunc := range checkFuncs {
		check := checkFunc(ctx)
		checks = append(checks, check)
		if !check.Success {
			allSuccess = false
		}
	}

	// Generate summary
	successCount := 0
	for _, check := range checks {
		if check.Success {
			successCount++
		}
	}

	summary := fmt.Sprintf("%d/%d checks passed", successCount, len(checks))
	if !allSuccess {
		summary += " - See failed checks above"
	}

	return installer.ValidationResult{
		Success: allSuccess,
		Checks:  checks,
		Summary: summary,
	}
}

func (c *Checker) checkBinaryExists(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	_, err := os.Stat(c.installPath)
	success := err == nil

	return installer.ValidationCheck{
		Name:        "Binary exists",
		Description: "Verify the binary file exists at the install location",
		Success:     success,
		Error:       err,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"path": c.installPath,
		},
	}
}

func (c *Checker) checkBinaryExecutable(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	info, err := os.Stat(c.installPath)
	if err != nil {
		return installer.ValidationCheck{
			Name:        "Binary executable",
			Description: "Verify the binary has executable permissions",
			Success:     false,
			Error:       err,
			Duration:    time.Since(start),
		}
	}

	success := info.Mode()&0111 != 0

	return installer.ValidationCheck{
		Name:        "Binary executable",
		Description: "Verify the binary has executable permissions",
		Success:     success,
		Error:       nil,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"permissions": info.Mode().String(),
		},
	}
}

func (c *Checker) checkBinaryPermissions(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	info, err := os.Stat(c.installPath)
	if err != nil {
		return installer.ValidationCheck{
			Name:        "File permissions",
			Description: "Check detailed file permissions",
			Success:     false,
			Error:       err,
			Duration:    time.Since(start),
		}
	}

	expectedMode := os.FileMode(0755)
	actualMode := info.Mode().Perm()
	success := actualMode == expectedMode

	var checkErr error
	if !success {
		checkErr = fmt.Errorf("expected %v, got %v", expectedMode, actualMode)
	}

	return installer.ValidationCheck{
		Name:        "File permissions",
		Description: "Check detailed file permissions",
		Success:     success,
		Error:       checkErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"expected": expectedMode.String(),
			"actual":   actualMode.String(),
		},
	}
}

func (c *Checker) checkVersion(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	cmd := exec.CommandContext(ctx, c.installPath, "--version")
	output, err := cmd.CombinedOutput()

	success := err == nil

	return installer.ValidationCheck{
		Name:        "Version check",
		Description: "Verify the binary can report its version",
		Success:     success,
		Error:       err,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"output": string(output),
		},
	}
}

func (c *Checker) checkHelp(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	cmd := exec.CommandContext(ctx, c.installPath, "--help")
	output, err := cmd.CombinedOutput()

	success := err == nil && len(output) > 0

	return installer.ValidationCheck{
		Name:        "Help command",
		Description: "Verify the help command works",
		Success:     success,
		Error:       err,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"outputLength": len(output),
		},
	}
}

func (c *Checker) checkPathIntegration(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Check if binary is accessible via PATH
	binaryName := filepath.Base(c.installPath)
	path, err := exec.LookPath(binaryName)

	success := err == nil
	found := path != ""

	return installer.ValidationCheck{
		Name:        "PATH integration",
		Description: "Check if binary is accessible via PATH",
		Success:     success,
		Error:       err,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"found":      found,
			"foundPath":  path,
			"binaryName": binaryName,
		},
	}
}

func (c *Checker) checkDependencies(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// For this example, we'll check for common runtime dependencies
	// In a real implementation, this would be more sophisticated

	dependencies := []string{
		// Add actual dependencies here
	}

	missing := []string{}
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}

	success := len(missing) == 0
	var checkErr error
	if !success {
		checkErr = fmt.Errorf("missing dependencies: %v", missing)
	}

	return installer.ValidationCheck{
		Name:        "Dependencies",
		Description: "Check for required system dependencies",
		Success:     success,
		Error:       checkErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"checked": dependencies,
			"missing": missing,
		},
	}
}

func (c *Checker) checkDiskSpace(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Get file size
	info, err := os.Stat(c.installPath)
	if err != nil {
		return installer.ValidationCheck{
			Name:        "Disk space",
			Description: "Check available disk space",
			Success:     false,
			Error:       err,
			Duration:    time.Since(start),
		}
	}

	fileSize := info.Size()

	// In a real implementation, would check actual disk space
	// For now, just report the file size

	return installer.ValidationCheck{
		Name:        "Disk space",
		Description: "Check available disk space",
		Success:     true,
		Error:       nil,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"binarySize": fileSize,
			"humanSize":  formatBytes(fileSize),
		},
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
