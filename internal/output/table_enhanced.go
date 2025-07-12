package output

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// EnhancedTableRenderer provides enhanced table output with colors and alignment
type EnhancedTableRenderer struct {
	noColor     bool
	maxWidth    int
	showSummary bool
}

// NewEnhancedTableRenderer creates a new enhanced table renderer
func NewEnhancedTableRenderer(noColor bool, maxWidth int) *EnhancedTableRenderer {
	if maxWidth == 0 {
		maxWidth = 120 // Default terminal width
	}
	return &EnhancedTableRenderer{
		noColor:     noColor,
		maxWidth:    maxWidth,
		showSummary: true,
	}
}

// RenderDriftReport renders a drift report as a formatted table
func (r *EnhancedTableRenderer) RenderDriftReport(report *differ.DriftReport) string {
	var output strings.Builder

	// Render summary header
	output.WriteString(r.renderSummaryHeader(report))
	output.WriteString("\n")

	if len(report.ResourceChanges) == 0 {
		output.WriteString(r.colorize("No drift detected - infrastructure matches baseline\n", color.FgGreen))
		return output.String()
	}

	// Sort changes by severity (critical first)
	sortedChanges := make([]differ.ResourceDiff, len(report.ResourceChanges))
	copy(sortedChanges, report.ResourceChanges)
	sort.Slice(sortedChanges, func(i, j int) bool {
		return r.severityWeight(sortedChanges[i].Severity) > r.severityWeight(sortedChanges[j].Severity)
	})

	// Render main changes table
	output.WriteString(r.renderChangesTable(sortedChanges))
	output.WriteString("\n")

	// Render summary statistics
	output.WriteString(r.renderSummaryStats(report))

	return output.String()
}

func (r *EnhancedTableRenderer) renderSummaryHeader(report *differ.DriftReport) string {
	var header strings.Builder

	header.WriteString(r.colorize("Infrastructure Drift Report\n", color.FgCyan, color.Bold))
	header.WriteString(r.colorize("═══════════════════════════════\n", color.FgCyan))

	// Overall risk indicator
	riskIcon := r.getRiskIcon(report.Summary.OverallRisk)
	riskColor := r.getRiskColor(report.Summary.OverallRisk)
	header.WriteString(fmt.Sprintf("%s Overall Risk: %s (Score: %.2f)\n",
		riskIcon,
		r.colorize(string(report.Summary.OverallRisk), riskColor, color.Bold),
		report.Summary.RiskScore))

	return header.String()
}

func (r *EnhancedTableRenderer) renderChangesTable(changes []differ.ResourceDiff) string {
	if len(changes) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := r.calculateColumnWidths(changes)

	var table strings.Builder

	// Table header
	table.WriteString(r.renderTableHeader(colWidths))
	table.WriteString(r.renderTableSeparator(colWidths, "├", "┼", "┤"))

	// Table rows
	for _, change := range changes {
		table.WriteString(r.renderTableRow(change, colWidths))
	}

	// Table footer
	table.WriteString(r.renderTableSeparator(colWidths, "└", "┴", "┘"))

	return table.String()
}

func (r *EnhancedTableRenderer) calculateColumnWidths(changes []differ.ResourceDiff) map[string]int {
	widths := map[string]int{
		"resource": len("Resource"),
		"change":   len("Change"),
		"severity": len("Severity"),
		"category": len("Category"),
		"impact":   len("Impact"),
	}

	for _, change := range changes {
		// Truncate long resource IDs for display
		resourceDisplay := r.truncateResource(change.ResourceID, change.ResourceType)
		if len(resourceDisplay) > widths["resource"] {
			widths["resource"] = len(resourceDisplay)
		}
		if len(string(change.DriftType)) > widths["change"] {
			widths["change"] = len(string(change.DriftType))
		}
		if len(string(change.Severity)) > widths["severity"] {
			widths["severity"] = len(string(change.Severity))
		}
		if len(string(change.Category)) > widths["category"] {
			widths["category"] = len(string(change.Category))
		}
		if len(change.Description) > widths["impact"] && len(change.Description) < 30 {
			widths["impact"] = len(change.Description)
		}
	}

	// Apply max width constraints
	maxResourceWidth := r.maxWidth / 3
	maxImpactWidth := r.maxWidth / 4

	if widths["resource"] > maxResourceWidth {
		widths["resource"] = maxResourceWidth
	}
	if widths["impact"] > maxImpactWidth {
		widths["impact"] = maxImpactWidth
	}

	return widths
}

func (r *EnhancedTableRenderer) renderTableHeader(colWidths map[string]int) string {
	var header strings.Builder

	// Top border
	header.WriteString(r.renderTableSeparator(colWidths, "┌", "┬", "┐"))

	// Header row
	header.WriteString("│ ")
	header.WriteString(r.colorize(r.padString("Resource", colWidths["resource"]), color.FgWhite, color.Bold))
	header.WriteString(" │ ")
	header.WriteString(r.colorize(r.padString("Change", colWidths["change"]), color.FgWhite, color.Bold))
	header.WriteString(" │ ")
	header.WriteString(r.colorize(r.padString("Severity", colWidths["severity"]), color.FgWhite, color.Bold))
	header.WriteString(" │ ")
	header.WriteString(r.colorize(r.padString("Category", colWidths["category"]), color.FgWhite, color.Bold))
	header.WriteString(" │ ")
	header.WriteString(r.colorize(r.padString("Impact", colWidths["impact"]), color.FgWhite, color.Bold))
	header.WriteString(" │\n")

	return header.String()
}

func (r *EnhancedTableRenderer) renderTableRow(change differ.ResourceDiff, colWidths map[string]int) string {
	var row strings.Builder

	// Resource column with type prefix
	resourceDisplay := r.truncateResource(change.ResourceID, change.ResourceType)
	resourceColor := color.FgWhite

	row.WriteString("│ ")
	row.WriteString(r.colorize(r.padString(resourceDisplay, colWidths["resource"]), resourceColor))
	row.WriteString(" │ ")

	// Change type with appropriate color
	changeColor := r.getChangeTypeColor(change.DriftType)
	changeIcon := r.getChangeTypeIcon(change.DriftType)
	changeDisplay := fmt.Sprintf("%s %s", changeIcon, strings.Title(string(change.DriftType)))
	row.WriteString(r.colorize(r.padString(changeDisplay, colWidths["change"]), changeColor))
	row.WriteString(" │ ")

	// Severity with color and icon
	severityColor := r.getRiskColor(change.Severity)
	severityIcon := r.getRiskIcon(change.Severity)
	severityDisplay := fmt.Sprintf("%s %s", severityIcon, strings.ToUpper(string(change.Severity)))
	row.WriteString(r.colorize(r.padString(severityDisplay, colWidths["severity"]), severityColor, color.Bold))
	row.WriteString(" │ ")

	// Category
	categoryColor := r.getCategoryColor(change.Category)
	row.WriteString(r.colorize(r.padString(strings.Title(string(change.Category)), colWidths["category"]), categoryColor))
	row.WriteString(" │ ")

	// Impact description
	impact := r.truncateString(change.Description, colWidths["impact"])
	row.WriteString(r.colorize(r.padString(impact, colWidths["impact"]), color.FgWhite))
	row.WriteString(" │\n")

	return row.String()
}

func (r *EnhancedTableRenderer) renderTableSeparator(colWidths map[string]int, left, mid, right string) string {
	var sep strings.Builder

	sep.WriteString(left)
	sep.WriteString(strings.Repeat("─", colWidths["resource"]+2))
	sep.WriteString(mid)
	sep.WriteString(strings.Repeat("─", colWidths["change"]+2))
	sep.WriteString(mid)
	sep.WriteString(strings.Repeat("─", colWidths["severity"]+2))
	sep.WriteString(mid)
	sep.WriteString(strings.Repeat("─", colWidths["category"]+2))
	sep.WriteString(mid)
	sep.WriteString(strings.Repeat("─", colWidths["impact"]+2))
	sep.WriteString(right)
	sep.WriteString("\n")

	return sep.String()
}

func (r *EnhancedTableRenderer) renderSummaryStats(report *differ.DriftReport) string {
	var stats strings.Builder

	stats.WriteString(r.colorize("📈 Change Summary\n", color.FgCyan, color.Bold))
	stats.WriteString(r.colorize("─────────────────\n", color.FgCyan))

	// Resource counts
	stats.WriteString(fmt.Sprintf("Total Resources: %s\n",
		r.colorize(strconv.Itoa(report.Summary.TotalResources), color.FgWhite, color.Bold)))
	stats.WriteString(fmt.Sprintf("Changed Resources: %s\n",
		r.colorize(strconv.Itoa(report.Summary.ChangedResources), color.FgYellow, color.Bold)))

	if report.Summary.AddedResources > 0 {
		stats.WriteString(fmt.Sprintf("  ➕ Added: %s\n",
			r.colorize(strconv.Itoa(report.Summary.AddedResources), color.FgGreen, color.Bold)))
	}
	if report.Summary.RemovedResources > 0 {
		stats.WriteString(fmt.Sprintf("  ➖ Removed: %s\n",
			r.colorize(strconv.Itoa(report.Summary.RemovedResources), color.FgRed, color.Bold)))
	}
	if report.Summary.ModifiedResources > 0 {
		stats.WriteString(fmt.Sprintf("  🔄 Modified: %s\n",
			r.colorize(strconv.Itoa(report.Summary.ModifiedResources), color.FgYellow, color.Bold)))
	}

	// Severity breakdown
	if len(report.Summary.ChangesBySeverity) > 0 {
		stats.WriteString("\nSeverity Breakdown:\n")
		severities := []differ.RiskLevel{differ.RiskLevelCritical, differ.RiskLevelHigh, differ.RiskLevelMedium, differ.RiskLevelLow}
		for _, severity := range severities {
			if count, exists := report.Summary.ChangesBySeverity[severity]; exists && count > 0 {
				icon := r.getRiskIcon(severity)
				stats.WriteString(fmt.Sprintf("  %s %s: %d\n",
					icon,
					strings.Title(string(severity)),
					count))
			}
		}
	}

	return stats.String()
}

// Helper methods for colors and formatting

func (r *EnhancedTableRenderer) colorize(text string, attrs ...color.Attribute) string {
	if r.noColor {
		return text
	}
	return color.New(attrs...).Sprint(text)
}

func (r *EnhancedTableRenderer) getRiskColor(risk differ.RiskLevel) color.Attribute {
	switch risk {
	case differ.RiskLevelCritical:
		return color.FgRed
	case differ.RiskLevelHigh:
		return color.FgYellow
	case differ.RiskLevelMedium:
		return color.FgBlue
	case differ.RiskLevelLow:
		return color.FgGreen
	default:
		return color.FgWhite
	}
}

func (r *EnhancedTableRenderer) getRiskIcon(risk differ.RiskLevel) string {
	switch risk {
	case differ.RiskLevelCritical:
		return "🔴"
	case differ.RiskLevelHigh:
		return "🟡"
	case differ.RiskLevelMedium:
		return "🔵"
	case differ.RiskLevelLow:
		return "🟢"
	default:
		return "⚪"
	}
}

func (r *EnhancedTableRenderer) getChangeTypeColor(changeType differ.ChangeType) color.Attribute {
	switch changeType {
	case differ.ChangeTypeAdded:
		return color.FgGreen
	case differ.ChangeTypeRemoved:
		return color.FgRed
	case differ.ChangeTypeModified:
		return color.FgYellow
	default:
		return color.FgWhite
	}
}

func (r *EnhancedTableRenderer) getChangeTypeIcon(changeType differ.ChangeType) string {
	switch changeType {
	case differ.ChangeTypeAdded:
		return "➕"
	case differ.ChangeTypeRemoved:
		return "➖"
	case differ.ChangeTypeModified:
		return "🔄"
	case differ.ChangeTypeMoved:
		return "🔀"
	default:
		return "❓"
	}
}

func (r *EnhancedTableRenderer) getCategoryColor(category differ.DriftCategory) color.Attribute {
	switch category {
	case differ.DriftCategorySecurity:
		return color.FgRed
	case differ.DriftCategoryCost:
		return color.FgYellow
	case differ.DriftCategoryNetwork:
		return color.FgBlue
	case differ.DriftCategoryStorage:
		return color.FgMagenta
	case differ.DriftCategoryCompute:
		return color.FgCyan
	default:
		return color.FgWhite
	}
}

func (r *EnhancedTableRenderer) severityWeight(severity differ.RiskLevel) int {
	switch severity {
	case differ.RiskLevelCritical:
		return 4
	case differ.RiskLevelHigh:
		return 3
	case differ.RiskLevelMedium:
		return 2
	case differ.RiskLevelLow:
		return 1
	default:
		return 0
	}
}

func (r *EnhancedTableRenderer) truncateResource(id, resourceType string) string {
	// Show type prefix for clarity
	prefix := ""
	if resourceType != "" {
		prefix = resourceType + ":"
	}

	maxLen := 25 // Max length for resource display
	if len(prefix)+len(id) <= maxLen {
		return prefix + id
	}

	// Truncate ID to fit
	availableLen := maxLen - len(prefix) - 3 // 3 for "..."
	if availableLen <= 0 {
		return r.truncateString(id, maxLen)
	}

	return prefix + id[:availableLen] + "..."
}

func (r *EnhancedTableRenderer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (r *EnhancedTableRenderer) padString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// RenderResourceList renders a simple list of resources with geo info only
func (r *EnhancedTableRenderer) RenderResourceList(resources []types.Resource) string {
	if len(resources) == 0 {
		return r.colorize("No resources found.\n", color.FgYellow)
	}

	var output strings.Builder

	output.WriteString(r.colorize(fmt.Sprintf("Found %d resources:\n", len(resources)), color.FgCyan, color.Bold))

	// Group by provider
	byProvider := make(map[string][]types.Resource)
	for _, resource := range resources {
		byProvider[resource.Provider] = append(byProvider[resource.Provider], resource)
	}

	for provider, providerResources := range byProvider {
		output.WriteString(r.colorize(fmt.Sprintf("\n%s (%d resources):\n",
			strings.ToUpper(provider), len(providerResources)), color.FgBlue, color.Bold))

		// Group by type within provider
		byType := make(map[string]int)
		for _, resource := range providerResources {
			byType[resource.Type]++
		}

		for resourceType, count := range byType {
			output.WriteString(fmt.Sprintf("  %s: %s\n",
				resourceType,
				r.colorize(strconv.Itoa(count), color.FgWhite, color.Bold)))
		}
	}

	// Show geographic distribution if regions are available
	regionCounts := make(map[string]int)
	for _, resource := range resources {
		if resource.Region != "" {
			regionCounts[resource.Region]++
		}
	}

	if len(regionCounts) > 0 {
		output.WriteString(r.colorize("\nGeographic distribution:\n", color.FgCyan, color.Bold))
		for region, count := range regionCounts {
			output.WriteString(fmt.Sprintf("  %s: %d resources\n", region, count))
		}
	}

	return output.String()
}
