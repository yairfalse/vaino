//go:build windows
// +build windows

package platform

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func (d *detector) detectPlatformSpecific(info *Info) error {
	// Get Windows version
	if version, err := d.getWindowsVersion(); err == nil {
		info.Version = version
	}

	// Set distribution to windows
	info.Distribution = "windows"

	return nil
}

func (d *detector) getWindowsVersion() (string, error) {
	// Use PowerShell to get Windows version
	cmd := exec.Command("powershell", "-Command",
		"[System.Environment]::OSVersion.Version.ToString()")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.Output()
	if err != nil {
		// Fallback to wmic
		cmd = exec.Command("wmic", "os", "get", "version", "/value")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Version=") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Version=")), nil
			}
		}
		return "", fmt.Errorf("could not parse Windows version")
	}

	return strings.TrimSpace(string(output)), nil
}

// ChocolateyInstalled checks if Chocolatey is installed
func ChocolateyInstalled() bool {
	detector := &detector{}
	return detector.checkCommand("choco")
}

// ScoopInstalled checks if Scoop is installed
func ScoopInstalled() bool {
	detector := &detector{}
	return detector.checkCommand("scoop")
}

// IsAdmin checks if running with administrator privileges
func IsAdmin() bool {
	// This is a simplified check
	// In a real implementation, would use Windows API
	cmd := exec.Command("net", "session")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	return err == nil
}
