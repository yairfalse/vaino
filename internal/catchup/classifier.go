package catchup

import (
	"strings"
	"time"
)

// Classifier categorizes changes as planned, unplanned, or routine
type Classifier struct {
	// Patterns for identifying change types
	plannedPatterns   []string
	unplannedPatterns []string
	routinePatterns   []string

	// Time-based rules
	businessHours BusinessHours
}

// BusinessHours defines typical working hours
type BusinessHours struct {
	StartHour int // 9 AM
	EndHour   int // 5 PM
	Weekdays  []time.Weekday
}

// NewClassifier creates a new change classifier
func NewClassifier() *Classifier {
	return &Classifier{
		plannedPatterns: []string{
			"deployment",
			"release",
			"upgrade",
			"migration",
			"scheduled",
			"maintenance",
			"rollout",
			"update",
			"patch",
			"feature",
		},
		unplannedPatterns: []string{
			"incident",
			"emergency",
			"hotfix",
			"rollback",
			"failure",
			"outage",
			"critical",
			"urgent",
			"recovery",
			"fix",
			"restore",
			"revert",
		},
		routinePatterns: []string{
			"scaling",
			"backup",
			"snapshot",
			"cleanup",
			"rotation",
			"refresh",
			"sync",
			"health check",
			"monitoring",
			"log rotation",
			"certificate renewal",
			"cache clear",
		},
		businessHours: BusinessHours{
			StartHour: 9,
			EndHour:   17,
			Weekdays: []time.Weekday{
				time.Monday,
				time.Tuesday,
				time.Wednesday,
				time.Thursday,
				time.Friday,
			},
		},
	}
}

// Classify determines the type of a change
func (c *Classifier) Classify(change Change) ChangeType {
	// Convert description and tags to lowercase for matching
	lowerDesc := strings.ToLower(change.Description)
	lowerTags := make([]string, len(change.Tags))
	for i, tag := range change.Tags {
		lowerTags[i] = strings.ToLower(tag)
	}

	// Check for explicit tags first
	for _, tag := range lowerTags {
		if tag == "planned" || tag == "scheduled" {
			return ChangeTypePlanned
		}
		if tag == "incident" || tag == "emergency" || tag == "unplanned" {
			return ChangeTypeUnplanned
		}
		if tag == "routine" || tag == "automated" {
			return ChangeTypeRoutine
		}
	}

	// Check patterns in description
	// Priority: unplanned > planned > routine

	// Check for unplanned patterns (highest priority)
	for _, pattern := range c.unplannedPatterns {
		if strings.Contains(lowerDesc, pattern) {
			return ChangeTypeUnplanned
		}
	}

	// Check for planned patterns
	for _, pattern := range c.plannedPatterns {
		if strings.Contains(lowerDesc, pattern) {
			// Additional check: if it's during business hours, more likely planned
			if c.isDuringBusinessHours(change.Timestamp) {
				return ChangeTypePlanned
			}
			// Outside business hours but has planned keywords - check if it's routine
			for _, routinePattern := range c.routinePatterns {
				if strings.Contains(lowerDesc, routinePattern) {
					return ChangeTypeRoutine
				}
			}
			return ChangeTypePlanned
		}
	}

	// Check for routine patterns
	for _, pattern := range c.routinePatterns {
		if strings.Contains(lowerDesc, pattern) {
			return ChangeTypeRoutine
		}
	}

	// Use heuristics based on time and resource type
	return c.classifyByHeuristics(change)
}

// classifyByHeuristics uses time-based and resource-based rules
func (c *Classifier) classifyByHeuristics(change Change) ChangeType {
	// Changes during business hours are more likely planned
	if c.isDuringBusinessHours(change.Timestamp) {
		// Check resource type
		resourceType := strings.ToLower(change.Resource.Type)

		// Certain resource types are more likely to be routine
		routineResources := []string{
			"autoscaling",
			"backup",
			"snapshot",
			"log",
			"metric",
			"alarm",
			"cloudwatch",
		}

		for _, routine := range routineResources {
			if strings.Contains(resourceType, routine) {
				return ChangeTypeRoutine
			}
		}

		// Default to planned during business hours
		return ChangeTypePlanned
	}

	// Changes outside business hours
	// Check if it's a typical automated/routine operation
	hour := change.Timestamp.Hour()

	// Late night operations (12 AM - 4 AM) are often automated
	if hour >= 0 && hour <= 4 {
		return ChangeTypeRoutine
	}

	// Weekend changes are often planned maintenance
	if change.Timestamp.Weekday() == time.Saturday || change.Timestamp.Weekday() == time.Sunday {
		return ChangeTypePlanned
	}

	// Default to unplanned for off-hours manual changes
	return ChangeTypeUnplanned
}

// isDuringBusinessHours checks if a timestamp is during typical business hours
func (c *Classifier) isDuringBusinessHours(t time.Time) bool {
	// Check if it's a weekday
	isWeekday := false
	for _, weekday := range c.businessHours.Weekdays {
		if t.Weekday() == weekday {
			isWeekday = true
			break
		}
	}

	if !isWeekday {
		return false
	}

	// Check if it's during business hours
	hour := t.Hour()
	return hour >= c.businessHours.StartHour && hour < c.businessHours.EndHour
}

// AddPlannedPattern adds a custom pattern for identifying planned changes
func (c *Classifier) AddPlannedPattern(pattern string) {
	c.plannedPatterns = append(c.plannedPatterns, strings.ToLower(pattern))
}

// AddUnplannedPattern adds a custom pattern for identifying unplanned changes
func (c *Classifier) AddUnplannedPattern(pattern string) {
	c.unplannedPatterns = append(c.unplannedPatterns, strings.ToLower(pattern))
}

// AddRoutinePattern adds a custom pattern for identifying routine changes
func (c *Classifier) AddRoutinePattern(pattern string) {
	c.routinePatterns = append(c.routinePatterns, strings.ToLower(pattern))
}

// SetBusinessHours updates the business hours configuration
func (c *Classifier) SetBusinessHours(start, end int, weekdays []time.Weekday) {
	c.businessHours = BusinessHours{
		StartHour: start,
		EndHour:   end,
		Weekdays:  weekdays,
	}
}
