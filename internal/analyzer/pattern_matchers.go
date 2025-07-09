package analyzer

import (
	"fmt"
	"strings"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
)

// ScalingPatternMatcher identifies deployment/statefulset scaling patterns
type ScalingPatternMatcher struct {
	timeWindow time.Duration
}

func (s *ScalingPatternMatcher) Match(changes []differ.SimpleChange) []ChangeGroup {
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
						if s.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
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

func (s *ScalingPatternMatcher) GetPatternType() string {
	return "scaling"
}

func (s *ScalingPatternMatcher) GetConfidence() string {
	return "high"
}

func (s *ScalingPatternMatcher) isWithinTimeWindow(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= s.timeWindow
}

// ConfigUpdatePatternMatcher identifies configuration update patterns
type ConfigUpdatePatternMatcher struct {
	timeWindow time.Duration
}

func (c *ConfigUpdatePatternMatcher) Match(changes []differ.SimpleChange) []ChangeGroup {
	groups := []ChangeGroup{}
	used := make(map[string]bool)

	for _, change := range changes {
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
			}

			// Look for deployments that restarted after this config change
			for _, other := range changes {
				if used[other.ResourceID] {
					continue
				}

				// Look for deployment restarts (generation changes)
				if other.ResourceType == "deployment" && other.Type == "modified" {
					// Check if this happened after config change
					if other.Timestamp.After(change.Timestamp) &&
						other.Timestamp.Sub(change.Timestamp) <= 2*time.Minute {
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
					// Check if this pod restart happened after config change
					if other.Timestamp.After(change.Timestamp) &&
						other.Timestamp.Sub(change.Timestamp) <= 2*time.Minute {
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
				groups = append(groups, group)
				// Mark all changes in this group as used
				for _, change := range group.Changes {
					used[change.ResourceID] = true
				}
			}
		}
	}

	return groups
}

func (c *ConfigUpdatePatternMatcher) GetPatternType() string {
	return "config_update"
}

func (c *ConfigUpdatePatternMatcher) GetConfidence() string {
	return "medium"
}

// ServiceDeploymentPatternMatcher identifies new service deployment patterns
type ServiceDeploymentPatternMatcher struct {
	timeWindow time.Duration
}

func (s *ServiceDeploymentPatternMatcher) Match(changes []differ.SimpleChange) []ChangeGroup {
	groups := []ChangeGroup{}
	used := make(map[string]bool)

	for _, change := range changes {
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
			}

			// Look for related resources with similar names
			baseName := strings.TrimSuffix(change.ResourceName, "-service")
			for _, other := range changes {
				if used[other.ResourceID] || other.ResourceID == change.ResourceID {
					continue
				}

				// Must be in same namespace and created around same time
				if other.Namespace == change.Namespace &&
					other.Type == "added" &&
					s.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
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
				// Mark all changes in this group as used
				for _, change := range group.Changes {
					used[change.ResourceID] = true
				}
			}
		}
	}

	return groups
}

func (s *ServiceDeploymentPatternMatcher) GetPatternType() string {
	return "service_deployment"
}

func (s *ServiceDeploymentPatternMatcher) GetConfidence() string {
	return "medium"
}

func (s *ServiceDeploymentPatternMatcher) isWithinTimeWindow(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= s.timeWindow
}

// NetworkPatternMatcher identifies network-related change patterns
type NetworkPatternMatcher struct {
	timeWindow time.Duration
}

func (n *NetworkPatternMatcher) Match(changes []differ.SimpleChange) []ChangeGroup {
	groups := []ChangeGroup{}
	used := make(map[string]bool)

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
			}

			// Look for related service changes
			for _, other := range changes {
				if used[other.ResourceID] || other.ResourceID == change.ResourceID {
					continue
				}

				if other.ResourceType == "service" &&
					other.Namespace == change.Namespace &&
					n.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
					group.Changes = append(group.Changes, other)
					group.Description += fmt.Sprintf(", service %s updated", other.ResourceName)
				}
			}

			if len(group.Changes) > 1 {
				groups = append(groups, group)
				// Mark all changes in this group as used
				for _, change := range group.Changes {
					used[change.ResourceID] = true
				}
			}
		}
	}

	return groups
}

func (n *NetworkPatternMatcher) GetPatternType() string {
	return "network_changes"
}

func (n *NetworkPatternMatcher) GetConfidence() string {
	return "medium"
}

func (n *NetworkPatternMatcher) isWithinTimeWindow(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= n.timeWindow
}

// StoragePatternMatcher identifies storage-related change patterns
type StoragePatternMatcher struct {
	timeWindow time.Duration
}

func (s *StoragePatternMatcher) Match(changes []differ.SimpleChange) []ChangeGroup {
	groups := []ChangeGroup{}
	used := make(map[string]bool)

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
			}

			// Look for matching PV
			for _, other := range changes {
				if used[other.ResourceID] {
					continue
				}

				if other.ResourceType == "persistentvolume" &&
					other.Type == "added" &&
					s.isWithinTimeWindow(change.Timestamp, other.Timestamp) {
					// Check if PV name matches PVC
					if strings.Contains(other.ResourceName, "pvc-") {
						group.Changes = append(group.Changes, other)
						group.Description += ", PV provisioned"
					}
				}
			}

			if len(group.Changes) > 1 {
				groups = append(groups, group)
				// Mark all changes in this group as used
				for _, change := range group.Changes {
					used[change.ResourceID] = true
				}
			}
		}
	}

	return groups
}

func (s *StoragePatternMatcher) GetPatternType() string {
	return "storage_changes"
}

func (s *StoragePatternMatcher) GetConfidence() string {
	return "medium"
}

func (s *StoragePatternMatcher) isWithinTimeWindow(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= s.timeWindow
}

// SecurityPatternMatcher identifies security-related change patterns
type SecurityPatternMatcher struct {
	timeWindow time.Duration
}

func (s *SecurityPatternMatcher) Match(changes []differ.SimpleChange) []ChangeGroup {
	groups := []ChangeGroup{}
	used := make(map[string]bool)

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
				if !s.isWithinTimeWindow(secrets[0].Timestamp, secrets[i].Timestamp) {
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
				}
				groups = append(groups, group)

				// Mark all changes in this group as used
				for _, change := range secrets {
					used[change.ResourceID] = true
				}
			}
		}
	}

	return groups
}

func (s *SecurityPatternMatcher) GetPatternType() string {
	return "security_changes"
}

func (s *SecurityPatternMatcher) GetConfidence() string {
	return "medium"
}

func (s *SecurityPatternMatcher) isWithinTimeWindow(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= s.timeWindow
}
