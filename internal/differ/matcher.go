package differ

import (
	"fmt"
	"strings"

	"github.com/yairfalse/vaino/pkg/types"
)

// DefaultResourceMatcher implements ResourceMatcher using ID-based matching
type DefaultResourceMatcher struct{}

// Match matches resources between baseline and current snapshots
// Returns matched resource pairs, plus slices of added and removed resources
func (m *DefaultResourceMatcher) Match(baseline, current []types.Resource) ([]ResourceMatch, []types.Resource, []types.Resource) {
	matches := make(map[string]string)
	baselineMap := make(map[string]types.Resource)
	currentMap := make(map[string]types.Resource)

	// Build maps for efficient lookup
	for _, resource := range baseline {
		baselineMap[resource.ID] = resource
	}
	for _, resource := range current {
		currentMap[resource.ID] = resource
	}

	// Try direct ID matching first
	for baselineID, baselineResource := range baselineMap {
		if currentResource, exists := currentMap[baselineID]; exists {
			// Verify it's the same type of resource
			if baselineResource.Type == currentResource.Type &&
				baselineResource.Provider == currentResource.Provider {
				matches[baselineID] = baselineID
			}
		}
	}

	// For unmatched resources, try fuzzy matching based on name and type
	unmatchedBaseline := make(map[string]types.Resource)
	unmatchedCurrent := make(map[string]types.Resource)

	for id, resource := range baselineMap {
		if _, matched := matches[id]; !matched {
			unmatchedBaseline[id] = resource
		}
	}

	for id, resource := range currentMap {
		if _, matched := findByValue(matches, id); !matched {
			unmatchedCurrent[id] = resource
		}
	}

	// Try matching by name and type for unmatched resources
	m.fuzzyMatch(unmatchedBaseline, unmatchedCurrent, matches)

	// Determine added and removed resources
	var added, removed []types.Resource

	// Resources in current but not matched are added
	for id, resource := range currentMap {
		if _, matched := findByValue(matches, id); !matched {
			added = append(added, resource)
		}
	}

	// Resources in baseline but not matched are removed
	for id, resource := range baselineMap {
		if _, matched := matches[id]; !matched {
			removed = append(removed, resource)
		}
	}

	// Convert map matches to ResourceMatch slice
	var resourceMatches []ResourceMatch
	for baselineID, currentID := range matches {
		if baselineResource, exists := baselineMap[baselineID]; exists {
			if currentResource, exists := currentMap[currentID]; exists {
				resourceMatches = append(resourceMatches, ResourceMatch{
					Baseline: baselineResource,
					Current:  currentResource,
				})
			}
		}
	}

	return resourceMatches, added, removed
}

// fuzzyMatch attempts to match resources based on name, type, and other attributes
func (m *DefaultResourceMatcher) fuzzyMatch(baseline, current map[string]types.Resource, matches map[string]string) {
	for baselineID, baselineResource := range baseline {
		bestMatch := ""
		bestScore := 0.0

		for currentID, currentResource := range current {
			// Skip if already matched
			if _, matched := findByValue(matches, currentID); matched {
				continue
			}

			score := m.calculateSimilarity(baselineResource, currentResource)
			if score > bestScore && score > 0.7 { // Threshold for considering a match
				bestScore = score
				bestMatch = currentID
			}
		}

		if bestMatch != "" {
			matches[baselineID] = bestMatch
		}
	}
}

// calculateSimilarity calculates how similar two resources are (0.0 to 1.0)
func (m *DefaultResourceMatcher) calculateSimilarity(baseline, current types.Resource) float64 {
	score := 0.0
	factors := 0

	// Type must match for any similarity
	if baseline.Type != current.Type {
		return 0.0
	}

	// Provider must match
	if baseline.Provider != current.Provider {
		return 0.0
	}

	// Name similarity (most important for fuzzy matching)
	if baseline.Name != "" && current.Name != "" {
		nameScore := m.stringSimilarity(baseline.Name, current.Name)
		score += nameScore * 0.5
		factors++
	}

	// Region similarity
	if baseline.Region != "" && current.Region != "" {
		if baseline.Region == current.Region {
			score += 0.2
		}
		factors++
	}

	// Namespace similarity (for Kubernetes resources)
	if baseline.Namespace != "" && current.Namespace != "" {
		if baseline.Namespace == current.Namespace {
			score += 0.2
		}
		factors++
	}

	// Tag similarity
	if len(baseline.Tags) > 0 && len(current.Tags) > 0 {
		tagScore := m.tagSimilarity(baseline.Tags, current.Tags)
		score += tagScore * 0.1
		factors++
	}

	if factors == 0 {
		return 0.0
	}

	return score / float64(factors)
}

// stringSimilarity calculates string similarity using a simple approach
func (m *DefaultResourceMatcher) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Simple approach: check if one contains the other
	s1Lower := strings.ToLower(s1)
	s2Lower := strings.ToLower(s2)

	if strings.Contains(s1Lower, s2Lower) || strings.Contains(s2Lower, s1Lower) {
		shorter := len(s1)
		longer := len(s2)
		if len(s2) < len(s1) {
			shorter = len(s2)
			longer = len(s1)
		}
		return float64(shorter) / float64(longer)
	}

	return 0.0
}

// tagSimilarity calculates how similar two tag sets are
func (m *DefaultResourceMatcher) tagSimilarity(tags1, tags2 map[string]string) float64 {
	if len(tags1) == 0 && len(tags2) == 0 {
		return 1.0
	}

	commonTags := 0
	totalTags := len(tags1)
	if len(tags2) > totalTags {
		totalTags = len(tags2)
	}

	for key, value1 := range tags1 {
		if value2, exists := tags2[key]; exists && value1 == value2 {
			commonTags++
		}
	}

	if totalTags == 0 {
		return 0.0
	}

	return float64(commonTags) / float64(totalTags)
}

// findByValue finds a key in the map that has the given value
func findByValue(m map[string]string, value string) (string, bool) {
	for k, v := range m {
		if v == value {
			return k, true
		}
	}
	return "", false
}

// SmartResourceMatcher is an enhanced matcher that uses multiple strategies
type SmartResourceMatcher struct {
	strategies []matchingStrategy
}

type matchingStrategy interface {
	match(baseline, current []types.Resource) map[string]string
	priority() int
}

// NewSmartResourceMatcher creates a matcher with multiple strategies
func NewSmartResourceMatcher() *SmartResourceMatcher {
	return &SmartResourceMatcher{
		strategies: []matchingStrategy{
			&idMatchingStrategy{},
			&nameAndTypeMatchingStrategy{},
			&configurationMatchingStrategy{},
		},
	}
}

// Match uses multiple strategies to find the best matches
func (m *SmartResourceMatcher) Match(baseline, current []types.Resource) ([]ResourceMatch, []types.Resource, []types.Resource) {
	matches := make(map[string]string)

	// Apply strategies in order of priority
	for _, strategy := range m.strategies {
		strategyMatches := strategy.match(baseline, current)

		// Merge non-conflicting matches
		for baselineID, currentID := range strategyMatches {
			if _, exists := matches[baselineID]; !exists {
				if _, valueExists := findByValue(matches, currentID); !valueExists {
					matches[baselineID] = currentID
				}
			}
		}
	}

	// Determine added and removed resources
	baselineMap := make(map[string]types.Resource)
	currentMap := make(map[string]types.Resource)

	for _, resource := range baseline {
		baselineMap[resource.ID] = resource
	}
	for _, resource := range current {
		currentMap[resource.ID] = resource
	}

	var added, removed []types.Resource

	for id, resource := range currentMap {
		if _, matched := findByValue(matches, id); !matched {
			added = append(added, resource)
		}
	}

	for id, resource := range baselineMap {
		if _, matched := matches[id]; !matched {
			removed = append(removed, resource)
		}
	}

	// Convert map matches to ResourceMatch slice
	var resourceMatches []ResourceMatch
	for baselineID, currentID := range matches {
		if baselineResource, exists := baselineMap[baselineID]; exists {
			if currentResource, exists := currentMap[currentID]; exists {
				resourceMatches = append(resourceMatches, ResourceMatch{
					Baseline: baselineResource,
					Current:  currentResource,
				})
			}
		}
	}

	return resourceMatches, added, removed
}

// idMatchingStrategy matches resources by exact ID
type idMatchingStrategy struct{}

func (s *idMatchingStrategy) match(baseline, current []types.Resource) map[string]string {
	matches := make(map[string]string)
	currentMap := make(map[string]types.Resource)

	for _, resource := range current {
		currentMap[resource.ID] = resource
	}

	for _, baselineResource := range baseline {
		if currentResource, exists := currentMap[baselineResource.ID]; exists {
			if baselineResource.Type == currentResource.Type &&
				baselineResource.Provider == currentResource.Provider {
				matches[baselineResource.ID] = currentResource.ID
			}
		}
	}

	return matches
}

func (s *idMatchingStrategy) priority() int { return 1 }

// nameAndTypeMatchingStrategy matches by name and type
type nameAndTypeMatchingStrategy struct{}

func (s *nameAndTypeMatchingStrategy) match(baseline, current []types.Resource) map[string]string {
	matches := make(map[string]string)

	for _, baselineResource := range baseline {
		for _, currentResource := range current {
			if baselineResource.Name == currentResource.Name &&
				baselineResource.Type == currentResource.Type &&
				baselineResource.Provider == currentResource.Provider &&
				baselineResource.Name != "" {
				matches[baselineResource.ID] = currentResource.ID
				break
			}
		}
	}

	return matches
}

func (s *nameAndTypeMatchingStrategy) priority() int { return 2 }

// configurationMatchingStrategy matches by configuration similarity
type configurationMatchingStrategy struct{}

func (s *configurationMatchingStrategy) match(baseline, current []types.Resource) map[string]string {
	matches := make(map[string]string)

	for _, baselineResource := range baseline {
		bestMatch := ""
		bestScore := 0.0

		for _, currentResource := range current {
			if baselineResource.Type == currentResource.Type &&
				baselineResource.Provider == currentResource.Provider {

				score := s.calculateConfigSimilarity(baselineResource.Configuration, currentResource.Configuration)
				if score > bestScore && score > 0.8 {
					bestScore = score
					bestMatch = currentResource.ID
				}
			}
		}

		if bestMatch != "" {
			matches[baselineResource.ID] = bestMatch
		}
	}

	return matches
}

func (s *configurationMatchingStrategy) priority() int { return 3 }

func (s *configurationMatchingStrategy) calculateConfigSimilarity(config1, config2 map[string]interface{}) float64 {
	if len(config1) == 0 && len(config2) == 0 {
		return 1.0
	}

	commonKeys := 0
	totalKeys := len(config1)
	if len(config2) > totalKeys {
		totalKeys = len(config2)
	}

	for key, value1 := range config1 {
		if value2, exists := config2[key]; exists {
			// Simple equality check - could be enhanced for nested objects
			if fmt.Sprintf("%v", value1) == fmt.Sprintf("%v", value2) {
				commonKeys++
			}
		}
	}

	if totalKeys == 0 {
		return 1.0
	}

	return float64(commonKeys) / float64(totalKeys)
}
