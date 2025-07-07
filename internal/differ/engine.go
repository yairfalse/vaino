package differ

import (
	"fmt"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// DifferEngine is the main implementation of the Differ interface
type DifferEngine struct {
	matcher    ResourceMatcher
	comparer   Comparer
	classifier ChangeClassifier
	options    DiffOptions
}

// NewDifferEngine creates a new differ engine with default components
func NewDifferEngine(options ...DiffOptions) *DifferEngine {
	opts := DiffOptions{}
	if len(options) > 0 {
		opts = options[0]
	}

	return &DifferEngine{
		matcher:    &DefaultResourceMatcher{},
		comparer:   &DefaultComparer{options: opts},
		classifier: &DefaultClassifier{},
		options:    opts,
	}
}

// Compare compares two snapshots and returns a comprehensive drift report
func (d *DifferEngine) Compare(baseline, current *types.Snapshot) (*DriftReport, error) {
	if baseline == nil {
		return nil, fmt.Errorf("baseline snapshot is required")
	}
	if current == nil {
		return nil, fmt.Errorf("current snapshot is required")
	}

	startTime := time.Now()

	// Match resources between snapshots
	matches, added, removed := d.matcher.Match(baseline.Resources, current.Resources)

	var resourceChanges []ResourceDiff
	var allChanges []Change

	// Process removed resources
	for _, resource := range removed {
		change := Change{
			Type:        ChangeTypeRemoved,
			ResourceID:  resource.ID,
			Path:        "resource",
			Field:       "existence",
			OldValue:    resource,
			NewValue:    nil,
			Description: fmt.Sprintf("Resource %s was removed", resource.ID),
		}

		category, severity, riskScore := d.classifier.ClassifyChange(change)
		change.Category = category
		change.Severity = severity

		resourceDiff := ResourceDiff{
			ResourceID:   resource.ID,
			ResourceType: resource.Type,
			Provider:     resource.Provider,
			DriftType:    ChangeTypeRemoved,
			Changes:      []Change{change},
			Severity:     severity,
			Category:     category,
			RiskScore:    riskScore,
			Description:  change.Description,
		}

		resourceChanges = append(resourceChanges, resourceDiff)
		allChanges = append(allChanges, change)
	}

	// Process added resources
	for _, resource := range added {
		change := Change{
			Type:        ChangeTypeAdded,
			ResourceID:  resource.ID,
			Path:        "resource",
			Field:       "existence",
			OldValue:    nil,
			NewValue:    resource,
			Description: fmt.Sprintf("Resource %s was added", resource.ID),
		}

		category, severity, riskScore := d.classifier.ClassifyChange(change)
		change.Category = category
		change.Severity = severity

		resourceDiff := ResourceDiff{
			ResourceID:   resource.ID,
			ResourceType: resource.Type,
			Provider:     resource.Provider,
			DriftType:    ChangeTypeAdded,
			Changes:      []Change{change},
			Severity:     severity,
			Category:     category,
			RiskScore:    riskScore,
			Description:  change.Description,
		}

		resourceChanges = append(resourceChanges, resourceDiff)
		allChanges = append(allChanges, change)
	}

	// Process modified resources
	baselineResourceMap := make(map[string]types.Resource)
	for _, resource := range baseline.Resources {
		baselineResourceMap[resource.ID] = resource
	}

	currentResourceMap := make(map[string]types.Resource)
	for _, resource := range current.Resources {
		currentResourceMap[resource.ID] = resource
	}

	for baselineID, currentID := range matches {
		baselineResource := baselineResourceMap[baselineID]
		currentResource := currentResourceMap[currentID]

		changes := d.comparer.CompareResources(baselineResource, currentResource)
		if len(changes) > 0 {
			// Classify all changes for this resource
			for i := range changes {
				category, severity, _ := d.classifier.ClassifyChange(changes[i])
				changes[i].Category = category
				changes[i].Severity = severity
			}

			// Calculate overall risk for this resource
			resourceSeverity, resourceRiskScore := d.classifier.CalculateResourceRisk(changes)

			resourceDiff := ResourceDiff{
				ResourceID:   currentResource.ID,
				ResourceType: currentResource.Type,
				Provider:     currentResource.Provider,
				DriftType:    ChangeTypeModified,
				Changes:      changes,
				Severity:     resourceSeverity,
				Category:     d.getPrimaryCategory(changes),
				RiskScore:    resourceRiskScore,
				Description:  fmt.Sprintf("Resource %s has %d configuration changes", currentResource.ID, len(changes)),
			}

			resourceChanges = append(resourceChanges, resourceDiff)
			allChanges = append(allChanges, changes...)
		}
	}

	// Calculate drift summary
	summary := d.CalculateDrift(allChanges)

	// Calculate overall risk
	overallRisk, overallRiskScore := d.classifier.CalculateOverallRisk(summary)
	summary.OverallRisk = overallRisk
	summary.RiskScore = overallRiskScore

	report := &DriftReport{
		ID:              fmt.Sprintf("drift-%d", time.Now().Unix()),
		BaselineID:      baseline.ID,
		CurrentID:       current.ID,
		Timestamp:       time.Now(),
		Summary:         summary,
		ResourceChanges: resourceChanges,
		Metadata: ReportMetadata{
			ComparisonDuration: time.Since(startTime),
			BaselineTimestamp:  baseline.Timestamp,
			CurrentTimestamp:   current.Timestamp,
			DifferVersion:      "1.0.0",
		},
	}

	return report, nil
}

// CalculateDrift calculates summary statistics from a list of changes
func (d *DifferEngine) CalculateDrift(changes []Change) DriftSummary {
	summary := DriftSummary{
		ChangesByCategory: make(map[DriftCategory]int),
		ChangesBySeverity: make(map[RiskLevel]int),
	}

	resourceSet := make(map[string]bool)
	addedResources := make(map[string]bool)
	removedResources := make(map[string]bool)
	modifiedResources := make(map[string]bool)

	for _, change := range changes {
		resourceSet[change.ResourceID] = true

		// Count by category
		summary.ChangesByCategory[change.Category]++

		// Count by severity
		summary.ChangesBySeverity[change.Severity]++

		// Track change types
		switch change.Type {
		case ChangeTypeAdded:
			addedResources[change.ResourceID] = true
		case ChangeTypeRemoved:
			removedResources[change.ResourceID] = true
		case ChangeTypeModified:
			modifiedResources[change.ResourceID] = true
		}
	}

	summary.TotalResources = len(resourceSet)
	summary.ChangedResources = len(resourceSet)
	summary.AddedResources = len(addedResources)
	summary.RemovedResources = len(removedResources)
	summary.ModifiedResources = len(modifiedResources)

	return summary
}

// ClassifyChange classifies a change by type and returns its category
func (d *DifferEngine) ClassifyChange(change Change) ChangeType {
	return change.Type
}

// getPrimaryCategory determines the primary drift category for a resource
func (d *DifferEngine) getPrimaryCategory(changes []Change) DriftCategory {
	categoryCount := make(map[DriftCategory]int)
	for _, change := range changes {
		categoryCount[change.Category]++
	}

	var primaryCategory DriftCategory
	maxCount := 0
	for category, count := range categoryCount {
		if count > maxCount {
			maxCount = count
			primaryCategory = category
		}
	}

	if primaryCategory == "" {
		return DriftCategoryConfig
	}

	return primaryCategory
}