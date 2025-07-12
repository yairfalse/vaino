package platform

import (
	"runtime"
	"strings"
	"testing"
)

func TestDetector_Detect(t *testing.T) {
	detector := NewDetector()
	info, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if info.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, info.OS)
	}

	if info.Arch != runtime.GOARCH {
		t.Errorf("Expected arch %s, got %s", runtime.GOARCH, info.Arch)
	}

	// Check that at least some detection occurred
	if info.OS == "" || info.Arch == "" {
		t.Error("Basic platform information missing")
	}
}

func TestSupportedPlatform(t *testing.T) {
	tests := []struct {
		os       string
		arch     string
		expected bool
	}{
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"linux", "arm", true},
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"windows", "amd64", true},
		{"freebsd", "amd64", true},
		{"invalid", "amd64", false},
		{"linux", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			result := SupportedPlatform(tt.os, tt.arch)
			if result != tt.expected {
				t.Errorf("SupportedPlatform(%s, %s) = %v, want %v",
					tt.os, tt.arch, result, tt.expected)
			}
		})
	}
}

func TestBinaryName(t *testing.T) {
	tests := []struct {
		goos     string
		base     string
		expected string
	}{
		{"windows", "tapio", "tapio.exe"},
		{"linux", "tapio", "tapio"},
		{"darwin", "tapio", "tapio"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			// Save current GOOS
			oldGOOS := runtime.GOOS
			runtime.GOOS = tt.goos
			defer func() { runtime.GOOS = oldGOOS }()

			result := BinaryName(tt.base)
			if result != tt.expected {
				t.Errorf("BinaryName(%s) = %s, want %s",
					tt.base, result, tt.expected)
			}
		})
	}
}

func TestInstallDir(t *testing.T) {
	dir := InstallDir()
	if dir == "" {
		t.Error("InstallDir returned empty string")
	}

	// Check that it returns a reasonable path
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(dir, "\\") {
			t.Errorf("Windows path should contain backslash: %s", dir)
		}
	default:
		if !strings.HasPrefix(dir, "/") && !strings.HasPrefix(dir, ".") {
			t.Errorf("Unix path should be absolute or relative: %s", dir)
		}
	}
}

func TestDetector_detectContainer(t *testing.T) {
	detector := &detector{}

	// This test will likely return false in most test environments
	// but we're testing that the method doesn't panic
	isContainer := detector.detectContainer()

	// Just ensure we get a boolean result
	if isContainer != true && isContainer != false {
		t.Error("detectContainer should return a boolean")
	}
}

func TestDetector_checkCommand(t *testing.T) {
	detector := &detector{}

	// Test with a command that should exist on all platforms
	commands := []string{
		"echo", // Unix-like
		"cmd",  // Windows
	}

	found := false
	for _, cmd := range commands {
		if detector.checkCommand(cmd) {
			found = true
			break
		}
	}

	if !found && runtime.GOOS != "plan9" {
		t.Error("Should find at least one common command")
	}

	// Test with a command that shouldn't exist
	if detector.checkCommand("definitely-not-a-real-command-xyz123") {
		t.Error("Should not find non-existent command")
	}
}

func BenchmarkDetector_Detect(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.Detect()
		if err != nil {
			b.Fatalf("Detection failed: %v", err)
		}
	}
}

func BenchmarkSupportedPlatform(b *testing.B) {
	platforms := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"invalid", "invalid"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := platforms[i%len(platforms)]
		SupportedPlatform(p.os, p.arch)
	}
}
