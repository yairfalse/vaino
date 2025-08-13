package terraform

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yairfalse/vaino/pkg/types"
)

// HCLParser handles parsing of Terraform .tf files using simple regex
type HCLParser struct {
	resourceRegex *regexp.Regexp
	dataRegex     *regexp.Regexp
	moduleRegex   *regexp.Regexp
	variableRegex *regexp.Regexp
	outputRegex   *regexp.Regexp
	localsRegex   *regexp.Regexp
}

// NewHCLParser creates a new HCL parser with regex patterns
func NewHCLParser() *HCLParser {
	return &HCLParser{
		resourceRegex: regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"`),
		dataRegex:     regexp.MustCompile(`data\s+"([^"]+)"\s+"([^"]+)"`),
		moduleRegex:   regexp.MustCompile(`module\s+"([^"]+)"`),
		variableRegex: regexp.MustCompile(`variable\s+"([^"]+)"`),
		outputRegex:   regexp.MustCompile(`output\s+"([^"]+)"`),
		localsRegex:   regexp.MustCompile(`locals\s+\{`),
	}
}

// ParseTerraformFiles parses .tf files in a directory and returns resources
func (p *HCLParser) ParseTerraformFiles(directory string) ([]types.Resource, error) {
	var resources []types.Resource

	// Find all .tf files in the directory
	files, err := filepath.Glob(filepath.Join(directory, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to find .tf files: %w", err)
	}

	// Also check for .tf files in common subdirectories
	subdirs := []string{"modules", "environments", "config"}
	for _, subdir := range subdirs {
		subFiles, err := filepath.Glob(filepath.Join(directory, subdir, "*.tf"))
		if err == nil {
			files = append(files, subFiles...)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .tf files found in %s", directory)
	}

	// Parse each file
	for _, file := range files {
		fileResources, err := p.ParseFile(file)
		if err != nil {
			// Log warning but continue with other files
			fmt.Printf("Warning: Failed to parse %s: %v\n", file, err)
			continue
		}
		resources = append(resources, fileResources...)
	}

	return resources, nil
}

// ParseFile parses a single .tf file using regex
func (p *HCLParser) ParseFile(filename string) ([]types.Resource, error) {
	var resources []types.Resource

	// Read file content
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)

	// Extract resources
	resourceMatches := p.resourceRegex.FindAllStringSubmatch(contentStr, -1)
	for _, match := range resourceMatches {
		if len(match) >= 3 {
			resourceType := match[1]
			resourceName := match[2]

			// Determine provider from resource type
			provider := determineProvider(resourceType)

			resource := types.Resource{
				ID:       fmt.Sprintf("%s.%s", resourceType, resourceName),
				Name:     resourceName,
				Type:     resourceType,
				Provider: provider,
				Tags: map[string]string{
					"source":  "terraform_config",
					"file":    filepath.Base(filename),
					"managed": "true",
					"status":  "planned",
				},
				Configuration: map[string]interface{}{
					"resource_type": resourceType,
					"resource_name": resourceName,
				},
			}
			resources = append(resources, resource)
		}
	}

	// Extract data sources
	dataMatches := p.dataRegex.FindAllStringSubmatch(contentStr, -1)
	for _, match := range dataMatches {
		if len(match) >= 3 {
			dataType := match[1]
			dataName := match[2]

			resource := types.Resource{
				ID:       fmt.Sprintf("data.%s.%s", dataType, dataName),
				Name:     dataName,
				Type:     fmt.Sprintf("data_%s", dataType),
				Provider: determineProvider(dataType),
				Tags: map[string]string{
					"source": "terraform_config",
					"file":   filepath.Base(filename),
					"data":   "true",
					"status": "data_source",
				},
				Configuration: map[string]interface{}{
					"data_type": dataType,
					"data_name": dataName,
				},
			}
			resources = append(resources, resource)
		}
	}

	// Extract modules
	moduleMatches := p.moduleRegex.FindAllStringSubmatch(contentStr, -1)
	for _, match := range moduleMatches {
		if len(match) >= 2 {
			moduleName := match[1]

			// Try to extract source from the module block
			source := extractModuleSource(contentStr, moduleName)

			resource := types.Resource{
				ID:       fmt.Sprintf("module.%s", moduleName),
				Name:     moduleName,
				Type:     "module",
				Provider: "terraform",
				Tags: map[string]string{
					"source":        "terraform_config",
					"file":          filepath.Base(filename),
					"module":        "true",
					"module_source": source,
					"status":        "module",
				},
				Configuration: map[string]interface{}{
					"module_name":   moduleName,
					"module_source": source,
				},
			}
			resources = append(resources, resource)
		}
	}

	// Extract variables
	variableMatches := p.variableRegex.FindAllStringSubmatch(contentStr, -1)
	for _, match := range variableMatches {
		if len(match) >= 2 {
			varName := match[1]

			resource := types.Resource{
				ID:       fmt.Sprintf("variable.%s", varName),
				Name:     varName,
				Type:     "variable",
				Provider: "terraform",
				Tags: map[string]string{
					"source":   "terraform_config",
					"file":     filepath.Base(filename),
					"variable": "true",
					"status":   "configuration",
				},
				Configuration: map[string]interface{}{
					"variable_name": varName,
				},
			}
			resources = append(resources, resource)
		}
	}

	// Extract outputs
	outputMatches := p.outputRegex.FindAllStringSubmatch(contentStr, -1)
	for _, match := range outputMatches {
		if len(match) >= 2 {
			outputName := match[1]

			resource := types.Resource{
				ID:       fmt.Sprintf("output.%s", outputName),
				Name:     outputName,
				Type:     "output",
				Provider: "terraform",
				Tags: map[string]string{
					"source": "terraform_config",
					"file":   filepath.Base(filename),
					"output": "true",
					"status": "configuration",
				},
				Configuration: map[string]interface{}{
					"output_name": outputName,
				},
			}
			resources = append(resources, resource)
		}
	}

	// Count locals blocks
	localsMatches := p.localsRegex.FindAllString(contentStr, -1)
	if len(localsMatches) > 0 {
		resource := types.Resource{
			ID:       fmt.Sprintf("locals.%s", filepath.Base(filename)),
			Name:     fmt.Sprintf("locals_in_%s", filepath.Base(filename)),
			Type:     "locals",
			Provider: "terraform",
			Tags: map[string]string{
				"source": "terraform_config",
				"file":   filepath.Base(filename),
				"locals": "true",
				"count":  fmt.Sprintf("%d", len(localsMatches)),
				"status": "configuration",
			},
			Configuration: map[string]interface{}{
				"locals_blocks": len(localsMatches),
			},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// determineProvider determines the provider from a resource type
func determineProvider(resourceType string) string {
	lowerType := strings.ToLower(resourceType)

	switch {
	case strings.HasPrefix(lowerType, "aws_"):
		return "aws"
	case strings.HasPrefix(lowerType, "google_") || strings.HasPrefix(lowerType, "gcp_"):
		return "gcp"
	case strings.HasPrefix(lowerType, "azurerm_") || strings.HasPrefix(lowerType, "azuread_"):
		return "azure"
	case strings.HasPrefix(lowerType, "kubernetes_") || strings.HasPrefix(lowerType, "k8s_"):
		return "kubernetes"
	case strings.HasPrefix(lowerType, "helm_"):
		return "helm"
	case strings.HasPrefix(lowerType, "docker_"):
		return "docker"
	case strings.HasPrefix(lowerType, "local_") || strings.HasPrefix(lowerType, "null_") ||
		strings.HasPrefix(lowerType, "random_") || strings.HasPrefix(lowerType, "template_"):
		return "terraform"
	case strings.HasPrefix(lowerType, "github_"):
		return "github"
	case strings.HasPrefix(lowerType, "datadog_"):
		return "datadog"
	case strings.HasPrefix(lowerType, "cloudflare_"):
		return "cloudflare"
	default:
		// Try to extract provider from the first part
		parts := strings.Split(resourceType, "_")
		if len(parts) > 0 {
			return parts[0]
		}
		return "terraform"
	}
}

// extractModuleSource tries to extract the source from a module block
func extractModuleSource(content, moduleName string) string {
	// Create a regex pattern to find the module block and extract source
	pattern := fmt.Sprintf(`module\s+"%s"\s+\{[^}]*source\s*=\s*"([^"]+)"`, regexp.QuoteMeta(moduleName))
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}

	// Try with single quotes
	pattern = fmt.Sprintf(`module\s+"%s"\s+\{[^}]*source\s*=\s*'([^']+)'`, regexp.QuoteMeta(moduleName))
	re = regexp.MustCompile(pattern)

	matches = re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}

	return "unknown"
}
