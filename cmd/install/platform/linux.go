//go:build linux
// +build linux

package platform

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (d *detector) detectPlatformSpecific(info *Info) error {
	// Detect Linux distribution
	info.Distribution = d.detectLinuxDistribution()

	// Get OS version
	if version, err := d.getLinuxVersion(); err == nil {
		info.Version = version
	}

	return nil
}

func (d *detector) detectLinuxDistribution() string {
	// Try /etc/os-release first (systemd standard)
	if dist := d.parseOSRelease(); dist != "" {
		return dist
	}

	// Try LSB release
	if dist := d.parseLSBRelease(); dist != "" {
		return dist
	}

	// Try distribution-specific files
	distFiles := map[string]string{
		"/etc/debian_version": "debian",
		"/etc/redhat-release": "redhat",
		"/etc/fedora-release": "fedora",
		"/etc/centos-release": "centos",
		"/etc/SuSE-release":   "suse",
		"/etc/arch-release":   "arch",
		"/etc/alpine-release": "alpine",
		"/etc/gentoo-release": "gentoo",
	}

	for file, dist := range distFiles {
		if _, err := os.Stat(file); err == nil {
			return dist
		}
	}

	return "unknown"
}

func (d *detector) parseOSRelease() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	vars := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
			vars[key] = value
		}
	}

	// Prefer ID over NAME
	if id, ok := vars["ID"]; ok {
		return id
	}
	if name, ok := vars["NAME"]; ok {
		return strings.ToLower(strings.Fields(name)[0])
	}

	return ""
}

func (d *detector) parseLSBRelease() string {
	data, err := os.ReadFile("/etc/lsb-release")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "DISTRIB_ID=") {
			return strings.ToLower(strings.TrimPrefix(line, "DISTRIB_ID="))
		}
	}

	return ""
}

func (d *detector) getLinuxVersion() (string, error) {
	// Try uname first
	data, err := os.ReadFile("/proc/version")
	if err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			return parts[2], nil
		}
	}

	// Try /etc/os-release VERSION_ID
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VERSION_ID=") {
			version := strings.TrimPrefix(line, "VERSION_ID=")
			return strings.Trim(version, `"`), nil
		}
	}

	return "", fmt.Errorf("could not determine Linux version")
}

// LinuxPackageManager returns the package manager for the current distribution
func LinuxPackageManager() string {
	// Map of package managers by distribution
	packageManagers := map[string]string{
		"debian": "apt",
		"ubuntu": "apt",
		"rhel":   "yum",
		"centos": "yum",
		"fedora": "dnf",
		"suse":   "zypper",
		"arch":   "pacman",
		"alpine": "apk",
		"gentoo": "emerge",
	}

	detector := &detector{}
	dist := detector.detectLinuxDistribution()

	if pm, ok := packageManagers[dist]; ok {
		return pm
	}

	// Try to detect by checking for package manager binaries
	managers := []string{"apt", "yum", "dnf", "zypper", "pacman", "apk", "emerge"}
	for _, mgr := range managers {
		if detector.checkCommand(mgr) {
			return mgr
		}
	}

	return ""
}
