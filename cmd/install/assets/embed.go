package assets

import (
	_ "embed"
	"encoding/json"
)

// Embedded installation scripts and configurations

//go:embed scripts/install.sh
var InstallScriptUnix string

//go:embed scripts/install.ps1
var InstallScriptWindows string

//go:embed configs/default.json
var DefaultConfigJSON []byte

//go:embed configs/mirrors.json
var MirrorsConfigJSON []byte

// Config represents the embedded default configuration
type Config struct {
	DefaultVersion   string            `json:"defaultVersion"`
	DefaultMirrors   []string          `json:"defaultMirrors"`
	RetryAttempts    int               `json:"retryAttempts"`
	TimeoutSeconds   int               `json:"timeoutSeconds"`
	ValidationLevel  string            `json:"validationLevel"`
	Features         []string          `json:"features"`
	PlatformSettings map[string]string `json:"platformSettings"`
}

// Mirrors represents the embedded mirror configuration
type Mirrors struct {
	Primary  []string            `json:"primary"`
	Fallback []string            `json:"fallback"`
	Regional map[string][]string `json:"regional"`
}

// LoadDefaultConfig loads the embedded default configuration
func LoadDefaultConfig() (*Config, error) {
	var config Config
	if err := json.Unmarshal(DefaultConfigJSON, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// LoadMirrors loads the embedded mirror configuration
func LoadMirrors() (*Mirrors, error) {
	var mirrors Mirrors
	if err := json.Unmarshal(MirrorsConfigJSON, &mirrors); err != nil {
		return nil, err
	}
	return &mirrors, nil
}

// GetInstallScript returns the appropriate installation script for the platform
func GetInstallScript(platform string) string {
	switch platform {
	case "windows":
		return InstallScriptWindows
	default:
		return InstallScriptUnix
	}
}
