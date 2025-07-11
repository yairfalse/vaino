package analyzer

import (
	"fmt"
	"strings"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
)

// ChangeGroup represents a group of related changes
type ChangeGroup struct {
	Timestamp   time.Time
	Title       string
	Description string
	Changes     []differ.SimpleChange
	Reason      string // Why these are grouped
	Confidence  string // high, medium, low
}

// Correlator groups related changes together
type Correlator struct {
	timeWindow time.Duration
}

// NewCorrelator creates a new change correlator
func NewCorrelator() *Correlator {
	return &Correlator{
		timeWindow: 30 * time.Second, // Changes within 30s are likely related
	}
}

// GroupChanges analyzes changes and groups related ones
func (c *Correlator) GroupChanges(changes []differ.SimpleChange) []ChangeGroup {
	if len(changes) == 0 {
		return nil
	}

	// Track which changes have been grouped
	used := make(map[string]bool)
	groups := []ChangeGroup{}

	// Pattern 1: Deployment scaling and pod changes
	for _, group := range c.findScalingGroups(changes) {
		groups = append(groups, group)
		for _, change := range group.Changes {
			used[change.ResourceID] = true
		}
	}

	// Pattern 2: Service and related resources
	for _, group := range c.findServiceGroups(changes, used) {
		groups = append(groups, group)
		for _, change := range group.Changes {
			used[change.ResourceID] = true
		}
	}

	// Pattern 3: ConfigMap/Secret changes and pod restarts
	for _, group := range c.findConfigUpdateGroups(changes, used) {
		groups = append(groups, group)
		for _, change := range group.Changes {
			used[change.ResourceID] = true
		}
	}

	// Pattern 4: Network changes (ingress/service modifications)
	for _, group := range c.findNetworkGroups(changes, used) {
		groups = append(groups, group)
		for _, change := range group.Changes {
			used[change.ResourceID] = true
		}
	}

	// Pattern 5: Storage changes
	for _, group := range c.findStorageGroups(changes, used) {
		groups = append(groups, group)
		for _, change := range group.Changes {
			used[change.ResourceID] = true
		}
	}

	// Pattern 6: Security changes (RBAC, secrets, service accounts)
	for _, group := range c.findSecurityGroups(changes, used) {
		groups = append(groups, group)
		for _, change := range group.Changes {
			used[change.ResourceID] = true
		}
	}

	// Pattern 7: Remaining ungrouped changes
	var ungrouped []differ.SimpleChange
	for _, change := range changes {
		if !used[change.ResourceID] {
			ungrouped = append(ungrouped, change)
		}
	}

	if len(ungrouped) > 0 {
		groups = append(groups, ChangeGroup{
			Timestamp:   ungrouped[0].Timestamp,
			Title:       "Other Changes",
			Description: "Individual resource changes",
			Changes:     ungrouped,
			Confidence:  "low",
		})
	}

	return groups
}

// findScalingGroups finds deployment/statefulset scaling and related pod changes
func (c *Correlator) findScalingGroups(changes []differ.SimpleChange) []ChangeGroup {
	groups := []ChangeGroup{}

	for _, change := range changes {
		// Look for deployment/statefulset modifications with replica changes
		if change.Type == "modified" &&
			(change.ResourceType == "deployment" || change.ResourceType == "statefulset") {

			// Check if replicas changed
			var replicaChange *differ.SimpleFieldChange
			for _, detail := range change.Details {
				if detail.Field == "replicas" {
					replicaChange = &detail
					break
				}
			}

			if replicaChange != nil {
				group := ChangeGroup{
					Timestamp:   change.Timestamp,
					Title:       fmt.Sprintf("%s Scaling", change.ResourceName),
					Description: fmt.Sprintf("Scaled from %v to %v replicas", replicaChange.OldValue, replicaChange.NewValue),
					Changes:     []differ.SimpleChange{change},
					Reason:      "Deployment scaling detected",
					Confidence:  "high",
				}

				// Find related pod changes
				podPrefix := change.ResourceName
				for _, other := range changes {
					if other.ResourceType == "pod" && strings.HasPrefix(other.ResourceName, podPrefix) {
						if c.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
							group.Changes = append(group.Changes, other)
						}
					}
				}

				// Check for HPA triggers
				for _, other := range changes {
					if other.ResourceType == "horizontalpodautoscaler" &&
						other.ResourceName == change.ResourceName+"-hpa" {
						group.Changes = append(group.Changes, other)
						group.Description += " (HPA triggered)"
					}
				}

				groups = append(groups, group)
			}
		}
	}

	return groups
}

// findConfigUpdateGroups finds ConfigMap/Secret updates and related pod restarts
func (c *Correlator) findConfigUpdateGroups(changes []differ.SimpleChange, used map[string]bool) []ChangeGroup {
	groups := []ChangeGroup{}

	for _, change := range changes {
		// Skip if already used
		if used[change.ResourceID] {
			continue
		}

		if change.Type == "modified" &&
			(change.ResourceType == "configmap" || change.ResourceType == "secret") {

			group := ChangeGroup{
				Timestamp:   change.Timestamp,
				Title:       fmt.Sprintf("%s Update", change.ResourceName),
				Description: fmt.Sprintf("%s configuration changed", change.ResourceType),
				Changes:     []differ.SimpleChange{change},
				Reason:      "Configuration update",
				Confidence:  "medium",
			}

			// Look for deployments that actually restarted after this config change
			for _, other := range changes {
				if used[other.ResourceID] {
					continue
				}

				// Look for deployment restarts (generation changes)
				if other.ResourceType == "deployment" && other.Type == "modified" {
					// Check if this happened after config change within time window
					if other.Timestamp.After(change.Timestamp) &&
						other.Timestamp.Sub(change.Timestamp) <= c.timeWindow {
						// Check for generation change (indicates restart)
						for _, detail := range other.Details {
							if detail.Field == "generation" {
								group.Changes = append(group.Changes, other)
								group.Description += fmt.Sprintf(", triggered %s restart", other.ResourceName)
								break
							}
						}
					}
				}

				// Look for pod changes in same namespace
				if other.ResourceType == "pod" && other.Type == "modified" &&
					other.Namespace == change.Namespace {
					// Check if this pod restart happened after config change within time window
					if other.Timestamp.After(change.Timestamp) &&
						other.Timestamp.Sub(change.Timestamp) <= c.timeWindow {
						// Check if pod has restart in its details
						for _, detail := range other.Details {
							if strings.Contains(detail.Field, "restart") ||
								detail.Field == "status.phase" {
								group.Changes = append(group.Changes, other)
								break
							}
						}
					}
				}
			}

			if len(group.Changes) > 1 {
				// If we found related deployment restarts, confidence is high
				group.Confidence = "high"
				groups = append(groups, group)
			}
		}
	}

	return groups
}

// findServiceGroups finds services and their related resources
func (c *Correlator) findServiceGroups(changes []differ.SimpleChange, used map[string]bool) []ChangeGroup {
	groups := []ChangeGroup{}

	for _, change := range changes {
		// Skip if already used
		if used[change.ResourceID] {
			continue
		}

		if change.ResourceType == "service" && change.Type == "added" {
			group := ChangeGroup{
				Timestamp:   change.Timestamp,
				Title:       fmt.Sprintf("New Service: %s", change.ResourceName),
				Description: "Service and related resources created",
				Changes:     []differ.SimpleChange{change},
				Reason:      "New service deployment",
				Confidence:  "medium",
			}

			// Look for related resources with similar names
			// Be more strict - only match exact prefixes and same namespace
			baseName := strings.TrimSuffix(change.ResourceName, "-service")
			for _, other := range changes {
				if used[other.ResourceID] || other.ResourceID == change.ResourceID {
					continue
				}

				// Must be in same namespace and created around same time
				if other.Namespace == change.Namespace &&
					other.Type == "added" &&
					c.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
					// Check for exact base name match
					if other.ResourceName == baseName ||
						other.ResourceName == baseName+"-deployment" ||
						other.ResourceName == baseName+"-configmap" ||
						strings.HasPrefix(other.ResourceName, baseName+"-") {
						group.Changes = append(group.Changes, other)
					}
				}
			}

			if len(group.Changes) > 1 {
				groups = append(groups, group)
			}
		}
	}

	return groups
}

// findUngroupedChanges returns changes not in any group
func (c *Correlator) findUngroupedChanges(allChanges []differ.SimpleChange, groups []ChangeGroup) []differ.SimpleChange {
	grouped := make(map[string]bool)

	// Mark all grouped changes
	for _, group := range groups {
		for _, change := range group.Changes {
			grouped[change.ResourceID] = true
		}
	}

	// Find ungrouped
	ungrouped := []differ.SimpleChange{}
	for _, change := range allChanges {
		if !grouped[change.ResourceID] {
			ungrouped = append(ungrouped, change)
		}
	}

	return ungrouped
}

// findNetworkGroups finds related network changes (services, ingress, endpoints)
func (c *Correlator) findNetworkGroups(changes []differ.SimpleChange, used map[string]bool) []ChangeGroup {
	groups := []ChangeGroup{}

	// Look for ingress changes
	for _, change := range changes {
		if used[change.ResourceID] {
			continue
		}

		if change.ResourceType == "ingress" && change.Type == "modified" {
			group := ChangeGroup{
				Timestamp:   change.Timestamp,
				Title:       "Network Configuration Change",
				Description: fmt.Sprintf("Ingress %s modified", change.ResourceName),
				Changes:     []differ.SimpleChange{change},
				Reason:      "Routing update",
				Confidence:  "medium",
			}

			// Look for related service changes
			for _, other := range changes {
				if used[other.ResourceID] || other.ResourceID == change.ResourceID {
					continue
				}

				if other.ResourceType == "service" &&
					other.Namespace == change.Namespace &&
					c.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
					group.Changes = append(group.Changes, other)
					group.Description += fmt.Sprintf(", service %s updated", other.ResourceName)
				}
			}

			if len(group.Changes) > 1 {
				groups = append(groups, group)
			}
		}
	}

	return groups
}

// findStorageGroups finds related storage changes (PV, PVC, StorageClass)
func (c *Correlator) findStorageGroups(changes []differ.SimpleChange, used map[string]bool) []ChangeGroup {
	groups := []ChangeGroup{}

	for _, change := range changes {
		if used[change.ResourceID] {
			continue
		}

		if change.ResourceType == "persistentvolumeclaim" && change.Type == "added" {
			group := ChangeGroup{
				Timestamp:   change.Timestamp,
				Title:       "Storage Provisioning",
				Description: fmt.Sprintf("PVC %s created", change.ResourceName),
				Changes:     []differ.SimpleChange{change},
				Reason:      "New storage request",
				Confidence:  "high",
			}

			// Look for matching PV
			for _, other := range changes {
				if used[other.ResourceID] {
					continue
				}

				if other.ResourceType == "persistentvolume" &&
					other.Type == "added" &&
					c.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
					// Check if PV name matches PVC
					if strings.Contains(other.ResourceName, "pvc-") {
						group.Changes = append(group.Changes, other)
						group.Description += ", PV provisioned"
					}
				}
			}

			if len(group.Changes) > 1 {
				groups = append(groups, group)
			}
		}
	}

	return groups
}

// findSecurityGroups finds related security changes
func (c *Correlator) findSecurityGroups(changes []differ.SimpleChange, used map[string]bool) []ChangeGroup {
	groups := []ChangeGroup{}

	// Group secret rotations
	secretChanges := make(map[string][]differ.SimpleChange)
	for _, change := range changes {
		if used[change.ResourceID] {
			continue
		}

		if change.ResourceType == "secret" && change.Type == "modified" {
			secretChanges[change.Namespace] = append(secretChanges[change.Namespace], change)
		}
	}

	// If multiple secrets changed in same namespace at similar time, group them
	for ns, secrets := range secretChanges {
		if len(secrets) > 1 {
			// Check if all within time window
			allRelated := true
			for i := 1; i < len(secrets); i++ {
				if !c.isWithinTimeWindow(secrets[0].Timestamp, secrets[i].Timestamp) {
					allRelated = false
					break
				}
			}

			if allRelated {
				group := ChangeGroup{
					Timestamp:   secrets[0].Timestamp,
					Title:       fmt.Sprintf("Secret Rotation in %s", ns),
					Description: fmt.Sprintf("%d secrets updated", len(secrets)),
					Changes:     secrets,
					Reason:      "Coordinated secret rotation",
					Confidence:  "high",
				}
				groups = append(groups, group)
			}
		}
	}

	return groups
}

// isWithinTimeWindow checks if two timestamps are within the correlation window
func (c *Correlator) isWithinTimeWindow(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= c.timeWindow
}
