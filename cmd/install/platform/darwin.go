//go:build darwin
// +build darwin

package platform

import (
	"os/exec"
	"runtime"
	"strings"
)

func (d *detector) detectPlatformSpecific(info *Info) error {
	// Get macOS version
	if version, err := d.getMacOSVersion(); err == nil {
		info.Version = version
	}

	// Set distribution to macOS
	info.Distribution = "macos"

	return nil
}

func (d *detector) getMacOSVersion() (string, error) {
	// Use sw_vers command to get macOS version
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// HomebrewInstalled checks if Homebrew is installed
func HomebrewInstalled() bool {
	detector := &detector{}
	return detector.checkCommand("brew")
}

// HomebrewPrefix returns the Homebrew installation prefix
func HomebrewPrefix() string {
	// Check for Apple Silicon Mac
	if runtime.GOARCH == "arm64" {
		return "/opt/homebrew"
	}
	return "/usr/local"
}
