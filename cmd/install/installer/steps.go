package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// extractStep extracts the binary from archive
type extractStep struct {
	installer *BinaryInstaller
}

func (s *extractStep) Name() string {
	return "Extract"
}

func (s *extractStep) Execute(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))

	// Determine extraction method based on file type
	if strings.HasSuffix(tempFile, ".tar.gz") || strings.HasSuffix(tempFile, ".tgz") {
		return s.extractTarGz(ctx, tempFile, binary)
	} else if strings.HasSuffix(tempFile, ".zip") {
		return s.extractZip(ctx, tempFile, binary)
	} else {
		// Assume it's a raw binary
		return s.extractBinary(ctx, tempFile, binary)
	}
}

func (s *extractStep) extractBinary(ctx context.Context, source string, binary Binary) error {
	// For raw binaries, just move to temp location
	extractedPath := source + ".extracted"

	// Copy file
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(extractedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	return nil
}

func (s *extractStep) extractTarGz(ctx context.Context, source string, binary Binary) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	extractedPath := source + ".extracted"

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for the binary
		if header.Typeflag == tar.TypeReg && strings.Contains(header.Name, "tapio") {
			dst, err := os.OpenFile(extractedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create extracted file: %w", err)
			}

			if _, err := io.Copy(dst, tr); err != nil {
				dst.Close()
				return fmt.Errorf("failed to extract binary: %w", err)
			}
			dst.Close()
			return nil
		}
	}

	return fmt.Errorf("binary not found in archive")
}

func (s *extractStep) extractZip(ctx context.Context, source string, binary Binary) error {
	// ZIP extraction would be implemented here
	return fmt.Errorf("ZIP extraction not yet implemented")
}

func (s *extractStep) Rollback(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))
	extractedPath := tempFile + ".extracted"
	return os.RemoveAll(extractedPath)
}

func (s *extractStep) Validate(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))
	extractedPath := tempFile + ".extracted"
	_, err := os.Stat(extractedPath)
	return err
}

// installStep moves the binary to its final location
type installStep struct {
	installer *BinaryInstaller
}

func (s *installStep) Name() string {
	return "Install"
}

func (s *installStep) Execute(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))
	extractedPath := tempFile + ".extracted"

	// Ensure install directory exists
	installDir := filepath.Dir(binary.Path)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Backup existing binary if it exists
	if _, err := os.Stat(binary.Path); err == nil {
		backupPath := binary.Path + ".backup"
		if err := os.Rename(binary.Path, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing binary: %w", err)
		}
	}

	// Move binary to final location atomically
	if err := os.Rename(extractedPath, binary.Path); err != nil {
		// If rename fails (cross-device), fall back to copy
		if err := s.copyFile(extractedPath, binary.Path); err != nil {
			return fmt.Errorf("failed to install binary: %w", err)
		}
		os.Remove(extractedPath)
	}

	return nil
}

func (s *installStep) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func (s *installStep) Rollback(ctx context.Context, binary Binary) error {
	// Restore backup if it exists
	backupPath := binary.Path + ".backup"
	if _, err := os.Stat(backupPath); err == nil {
		os.Remove(binary.Path)
		return os.Rename(backupPath, binary.Path)
	}
	return os.Remove(binary.Path)
}

func (s *installStep) Validate(ctx context.Context, binary Binary) error {
	info, err := os.Stat(binary.Path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", binary.Path)
	}
	return nil
}

// pathStep adds the binary to PATH
type pathStep struct {
	installer *BinaryInstaller
}

func (s *pathStep) Name() string {
	return "Configure PATH"
}

func (s *pathStep) Execute(ctx context.Context, binary Binary) error {
	installDir := filepath.Dir(binary.Path)

	switch runtime.GOOS {
	case "darwin", "linux":
		return s.configureUnixPath(installDir)
	case "windows":
		return s.configureWindowsPath(installDir)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (s *pathStep) configureUnixPath(installDir string) error {
	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	var rcFile string
	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
	case strings.Contains(shell, "bash"):
		rcFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
	case strings.Contains(shell, "fish"):
		rcFile = filepath.Join(os.Getenv("HOME"), ".config", "fish", "config.fish")
	default:
		rcFile = filepath.Join(os.Getenv("HOME"), ".profile")
	}

	// Check if PATH already contains install directory
	currentPath := os.Getenv("PATH")
	if strings.Contains(currentPath, installDir) {
		return nil // Already in PATH
	}

	// Add to shell configuration
	pathLine := fmt.Sprintf("\n# Added by Tapio installer\nexport PATH=\"%s:$PATH\"\n", installDir)

	file, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shell config: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(pathLine); err != nil {
		return fmt.Errorf("failed to update PATH: %w", err)
	}

	fmt.Printf("\nℹ️  Added %s to PATH in %s\n", installDir, rcFile)
	fmt.Println("   Please restart your shell or run: source " + rcFile)

	return nil
}

func (s *pathStep) configureWindowsPath(installDir string) error {
	// Use PowerShell to update user PATH
	script := fmt.Sprintf(`
$path = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($path -notlike "*%s*") {
    [Environment]::SetEnvironmentVariable("PATH", "$path;%s", "User")
}
`, installDir, installDir)

	cmd := exec.Command("powershell", "-Command", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update PATH: %w", err)
	}

	fmt.Printf("\nℹ️  Added %s to PATH\n", installDir)
	fmt.Println("   Please restart your terminal for changes to take effect")

	return nil
}

func (s *pathStep) Rollback(ctx context.Context, binary Binary) error {
	// PATH changes are not rolled back to avoid breaking other installations
	return nil
}

func (s *pathStep) Validate(ctx context.Context, binary Binary) error {
	// Check if binary is accessible via PATH
	if _, err := exec.LookPath("tapio"); err != nil {
		// Not in PATH yet, but that's okay
		return nil
	}
	return nil
}

// permissionsStep ensures correct file permissions
type permissionsStep struct {
	installer *BinaryInstaller
}

func (s *permissionsStep) Name() string {
	return "Set Permissions"
}

func (s *permissionsStep) Execute(ctx context.Context, binary Binary) error {
	// Ensure binary is executable
	return os.Chmod(binary.Path, 0755)
}

func (s *permissionsStep) Rollback(ctx context.Context, binary Binary) error {
	return nil // Permission changes don't need rollback
}

func (s *permissionsStep) Validate(ctx context.Context, binary Binary) error {
	info, err := os.Stat(binary.Path)
	if err != nil {
		return err
	}

	// Check if executable
	mode := info.Mode()
	if mode&0111 == 0 {
		return fmt.Errorf("binary is not executable")
	}

	return nil
}
