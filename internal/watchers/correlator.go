package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// ConcurrentCorrelator correlates events across different providers
type ConcurrentCorrelator struct {
	mu               sync.RWMutex
	correlationRules []CorrelationRule
	eventHistory     []WatchEvent
	maxHistorySize   int
	correlationWindow time.Duration
	running          bool
	ctx              context.Context
	cancel           context.CancelFunc
	stats            CorrelatorStats
}

// CorrelationRule defines how to correlate events
type CorrelationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Providers   []string               `json:"providers"`
	EventTypes  []EventType            `json:"event_types"`
	TimeWindow  time.Duration          `json:"time_window"`
	Conditions  []CorrelationCondition `json:"conditions"`
	Actions     []CorrelationAction    `json:"actions"`
	Enabled     bool                   `json:"enabled"`
}

// CorrelationCondition defines conditions for correlation
type CorrelationCondition struct {
	Type        string      `json:"type"`
	Field       string      `json:"field"`
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	CaseSensitive bool      `json:"case_sensitive"`
}

// CorrelationAction defines actions to take when correlation is found
type CorrelationAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// CorrelatedEvent represents a correlated event
type CorrelatedEvent struct {
	ID             string       `json:"id"`
	Type           EventType    `json:"type"`
	Timestamp      time.Time    `json:"timestamp"`
	Provider       string       `json:"provider"`
	Resource       types.Resource `json:"resource"`
	CorrelationID  string       `json:"correlation_id"`
	RelatedEvents  []string     `json:"related_events"`
	Confidence     float64      `json:"confidence"`
	Severity       types.DriftSeverity `json:"severity"`
	Description    string       `json:"description"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CorrelatorStats holds statistics for the correlator
type CorrelatorStats struct {
	TotalProcessed     int64                       `json:"total_processed"`
	CorrelationsFound  int64                       `json:"correlations_found"`
	CorrelationRate    float64                     `json:"correlation_rate"`
	AverageProcessTime time.Duration               `json:"average_process_time"`
	RuleStats          map[string]RuleStats        `json:"rule_stats"`
	LastActivity       time.Time                   `json:"last_activity"`
	ErrorCount         int64                       `json:"error_count"`
}

// RuleStats holds statistics for a specific rule
type RuleStats struct {
	TriggeredCount int64         `json:"triggered_count"`
	SuccessCount   int64         `json:"success_count"`
	LastTriggered  time.Time     `json:"last_triggered"`
	AverageLatency time.Duration `json:"average_latency"`
}

// NewConcurrentCorrelator creates a new concurrent correlator
func NewConcurrentCorrelator() *ConcurrentCorrelator {
	ctx, cancel := context.WithCancel(context.Background())
	
	cc := &ConcurrentCorrelator{
		correlationRules:  []CorrelationRule{},
		eventHistory:      []WatchEvent{},
		maxHistorySize:    10000,
		correlationWindow: 5 * time.Minute,
		running:           false,
		ctx:               ctx,
		cancel:            cancel,
		stats: CorrelatorStats{
			RuleStats: make(map[string]RuleStats),
		},
	}
	
	// Add default correlation rules
	cc.addDefaultRules()
	
	return cc
}

// Start begins event correlation
func (cc *ConcurrentCorrelator) Start() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	if cc.running {
		return fmt.Errorf("correlator is already running")
	}
	
	cc.running = true
	
	// Start cleanup goroutine
	go cc.cleanupLoop()
	
	return nil
}

// Stop stops event correlation
func (cc *ConcurrentCorrelator) Stop() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	if !cc.running {
		return fmt.Errorf("correlator is not running")
	}
	
	cc.cancel()
	cc.running = false
	
	return nil
}

// ProcessEvent processes an event and returns correlated events
func (cc *ConcurrentCorrelator) ProcessEvent(event WatchEvent) []WatchEvent {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	startTime := time.Now()
	
	// Add event to history
	cc.addToHistory(event)
	
	// Find correlations
	correlatedEvents := cc.findCorrelations(event)
	
	// Update stats
	cc.stats.TotalProcessed++
	cc.stats.LastActivity = time.Now()
	
	processTime := time.Since(startTime)
	if cc.stats.AverageProcessTime == 0 {
		cc.stats.AverageProcessTime = processTime
	} else {
		cc.stats.AverageProcessTime = time.Duration((int64(cc.stats.AverageProcessTime) + int64(processTime)) / 2)
	}
	
	if len(correlatedEvents) > 0 {
		cc.stats.CorrelationsFound++
		cc.stats.CorrelationRate = float64(cc.stats.CorrelationsFound) / float64(cc.stats.TotalProcessed) * 100
	}
	
	return correlatedEvents
}

// AddRule adds a new correlation rule
func (cc *ConcurrentCorrelator) AddRule(rule CorrelationRule) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	// Validate rule
	if err := cc.validateRule(rule); err != nil {
		return fmt.Errorf("invalid correlation rule: %w", err)
	}
	
	cc.correlationRules = append(cc.correlationRules, rule)
	cc.stats.RuleStats[rule.ID] = RuleStats{}
	
	return nil
}

// RemoveRule removes a correlation rule
func (cc *ConcurrentCorrelator) RemoveRule(ruleID string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	for i, rule := range cc.correlationRules {
		if rule.ID == ruleID {
			cc.correlationRules = append(cc.correlationRules[:i], cc.correlationRules[i+1:]...)
			delete(cc.stats.RuleStats, ruleID)
			return nil
		}
	}
	
	return fmt.Errorf("rule with ID %s not found", ruleID)
}

// GetRules returns all correlation rules
func (cc *ConcurrentCorrelator) GetRules() []CorrelationRule {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	
	rules := make([]CorrelationRule, len(cc.correlationRules))
	copy(rules, cc.correlationRules)
	return rules
}

// GetStats returns current statistics
func (cc *ConcurrentCorrelator) GetStats() CorrelatorStats {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.stats
}

// IsRunning returns whether the correlator is running
func (cc *ConcurrentCorrelator) IsRunning() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.running
}

// addToHistory adds an event to the history
func (cc *ConcurrentCorrelator) addToHistory(event WatchEvent) {
	cc.eventHistory = append(cc.eventHistory, event)
	
	// Limit history size
	if len(cc.eventHistory) > cc.maxHistorySize {
		cc.eventHistory = cc.eventHistory[len(cc.eventHistory)-cc.maxHistorySize:]
	}
}

// findCorrelations finds correlations for an event
func (cc *ConcurrentCorrelator) findCorrelations(event WatchEvent) []WatchEvent {
	var correlatedEvents []WatchEvent
	
	for _, rule := range cc.correlationRules {
		if !rule.Enabled {
			continue
		}
		
		// Update rule stats
		stats := cc.stats.RuleStats[rule.ID]
		stats.TriggeredCount++
		stats.LastTriggered = time.Now()
		
		// Check if rule applies to this event
		if cc.ruleApplies(rule, event) {
			// Find related events
			relatedEvents := cc.findRelatedEvents(rule, event)
			
			if len(relatedEvents) > 0 {
				// Create correlated event
				correlatedEvent := cc.createCorrelatedEvent(rule, event, relatedEvents)
				correlatedEvents = append(correlatedEvents, correlatedEvent)
				
				stats.SuccessCount++
			}
		}
		
		cc.stats.RuleStats[rule.ID] = stats
	}
	
	return correlatedEvents
}

// ruleApplies checks if a rule applies to an event
func (cc *ConcurrentCorrelator) ruleApplies(rule CorrelationRule, event WatchEvent) bool {
	// Check provider
	providerMatch := false
	for _, provider := range rule.Providers {
		if provider == event.Provider {
			providerMatch = true
			break
		}
	}
	if !providerMatch {
		return false
	}
	
	// Check event type
	eventTypeMatch := false
	for _, eventType := range rule.EventTypes {
		if eventType == event.Type {
			eventTypeMatch = true
			break
		}
	}
	if !eventTypeMatch {
		return false
	}
	
	// Check conditions
	for _, condition := range rule.Conditions {
		if !cc.evaluateCondition(condition, event) {
			return false
		}
	}
	
	return true
}

// evaluateCondition evaluates a correlation condition
func (cc *ConcurrentCorrelator) evaluateCondition(condition CorrelationCondition, event WatchEvent) bool {
	var fieldValue interface{}
	
	// Get field value
	switch condition.Field {
	case "resource.type":
		fieldValue = event.Resource.Type
	case "resource.name":
		fieldValue = event.Resource.Name
	case "resource.region":
		fieldValue = event.Resource.Region
	case "resource.namespace":
		fieldValue = event.Resource.Namespace
	case "provider":
		fieldValue = event.Provider
	case "type":
		fieldValue = event.Type
	default:
		// Check in resource configuration
		if event.Resource.Configuration != nil {
			fieldValue = event.Resource.Configuration[condition.Field]
		}
	}
	
	// Evaluate condition
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if searchStr, ok := condition.Value.(string); ok {
				if condition.CaseSensitive {
					return str == searchStr
				}
				return str == searchStr // Simple implementation
			}
		}
	case "starts_with":
		if str, ok := fieldValue.(string); ok {
			if prefix, ok := condition.Value.(string); ok {
				return len(str) >= len(prefix) && str[:len(prefix)] == prefix
			}
		}
	case "ends_with":
		if str, ok := fieldValue.(string); ok {
			if suffix, ok := condition.Value.(string); ok {
				return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
			}
		}
	}
	
	return false
}

// findRelatedEvents finds events related to the current event
func (cc *ConcurrentCorrelator) findRelatedEvents(rule CorrelationRule, event WatchEvent) []WatchEvent {
	var relatedEvents []WatchEvent
	
	// Look for events within the time window
	timeWindow := rule.TimeWindow
	if timeWindow == 0 {
		timeWindow = cc.correlationWindow
	}
	
	cutoff := event.Timestamp.Add(-timeWindow)
	
	for _, histEvent := range cc.eventHistory {
		if histEvent.ID == event.ID {
			continue // Skip the same event
		}
		
		if histEvent.Timestamp.Before(cutoff) {
			continue // Too old
		}
		
		// Check if this event is related
		if cc.eventsRelated(event, histEvent) {
			relatedEvents = append(relatedEvents, histEvent)
		}
	}
	
	return relatedEvents
}

// eventsRelated checks if two events are related
func (cc *ConcurrentCorrelator) eventsRelated(event1, event2 WatchEvent) bool {
	// Same resource
	if event1.Resource.ID == event2.Resource.ID {
		return true
	}
	
	// Same resource type and name
	if event1.Resource.Type == event2.Resource.Type && 
	   event1.Resource.Name == event2.Resource.Name {
		return true
	}
	
	// Same namespace/region
	if event1.Resource.Namespace != "" && 
	   event1.Resource.Namespace == event2.Resource.Namespace {
		return true
	}
	
	if event1.Resource.Region != "" && 
	   event1.Resource.Region == event2.Resource.Region {
		return true
	}
	
	return false
}

// createCorrelatedEvent creates a correlated event
func (cc *ConcurrentCorrelator) createCorrelatedEvent(rule CorrelationRule, primaryEvent WatchEvent, relatedEvents []WatchEvent) WatchEvent {
	var relatedEventIDs []string
	for _, event := range relatedEvents {
		relatedEventIDs = append(relatedEventIDs, event.ID)
	}
	
	correlatedEvent := WatchEvent{
		ID:        fmt.Sprintf("corr-%s-%d", primaryEvent.ID, time.Now().UnixNano()),
		Type:      primaryEvent.Type,
		Timestamp: time.Now(),
		Provider:  "correlator",
		Resource:  primaryEvent.Resource,
		Metadata: map[string]interface{}{
			"correlation_rule":  rule.ID,
			"rule_name":         rule.Name,
			"primary_event":     primaryEvent.ID,
			"related_events":    relatedEventIDs,
			"correlation_count": len(relatedEvents),
			"confidence":        cc.calculateConfidence(primaryEvent, relatedEvents),
		},
	}
	
	return correlatedEvent
}

// calculateConfidence calculates confidence score for correlation
func (cc *ConcurrentCorrelator) calculateConfidence(primaryEvent WatchEvent, relatedEvents []WatchEvent) float64 {
	if len(relatedEvents) == 0 {
		return 0.0
	}
	
	confidence := 0.5 // Base confidence
	
	// Increase confidence based on number of related events
	confidence += float64(len(relatedEvents)) * 0.1
	
	// Increase confidence if events are from different providers
	providers := make(map[string]bool)
	providers[primaryEvent.Provider] = true
	
	for _, event := range relatedEvents {
		if !providers[event.Provider] {
			providers[event.Provider] = true
			confidence += 0.15
		}
	}
	
	// Increase confidence if events happened close in time
	for _, event := range relatedEvents {
		timeDiff := primaryEvent.Timestamp.Sub(event.Timestamp)
		if timeDiff < time.Minute {
			confidence += 0.1
		}
	}
	
	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// validateRule validates a correlation rule
func (cc *ConcurrentCorrelator) validateRule(rule CorrelationRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	
	if len(rule.Providers) == 0 {
		return fmt.Errorf("at least one provider is required")
	}
	
	if len(rule.EventTypes) == 0 {
		return fmt.Errorf("at least one event type is required")
	}
	
	return nil
}

// addDefaultRules adds default correlation rules
func (cc *ConcurrentCorrelator) addDefaultRules() {
	// Cross-provider resource correlation
	cc.correlationRules = append(cc.correlationRules, CorrelationRule{
		ID:          "cross-provider-resource",
		Name:        "Cross-Provider Resource Correlation",
		Description: "Correlate events for the same resource across different providers",
		Providers:   []string{"terraform", "aws", "gcp", "kubernetes"},
		EventTypes:  []EventType{EventTypeResourceCreated, EventTypeResourceDeleted, EventTypeResourceModified},
		TimeWindow:  5 * time.Minute,
		Conditions: []CorrelationCondition{
			{
				Type:     "resource_match",
				Field:    "resource.name",
				Operator: "equals",
			},
		},
		Enabled: true,
	})
	
	// Namespace-based correlation for Kubernetes
	cc.correlationRules = append(cc.correlationRules, CorrelationRule{
		ID:          "kubernetes-namespace",
		Name:        "Kubernetes Namespace Correlation",
		Description: "Correlate events within the same Kubernetes namespace",
		Providers:   []string{"kubernetes"},
		EventTypes:  []EventType{EventTypeResourceCreated, EventTypeResourceDeleted, EventTypeResourceModified},
		TimeWindow:  2 * time.Minute,
		Conditions: []CorrelationCondition{
			{
				Type:     "namespace_match",
				Field:    "resource.namespace",
				Operator: "equals",
			},
		},
		Enabled: true,
	})
	
	// Region-based correlation for cloud providers
	cc.correlationRules = append(cc.correlationRules, CorrelationRule{
		ID:          "cloud-region",
		Name:        "Cloud Region Correlation",
		Description: "Correlate events within the same cloud region",
		Providers:   []string{"aws", "gcp"},
		EventTypes:  []EventType{EventTypeResourceCreated, EventTypeResourceDeleted, EventTypeResourceModified},
		TimeWindow:  10 * time.Minute,
		Conditions: []CorrelationCondition{
			{
				Type:     "region_match",
				Field:    "resource.region",
				Operator: "equals",
			},
		},
		Enabled: true,
	})
}

// cleanupLoop periodically cleans up old events from history
func (cc *ConcurrentCorrelator) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-cc.ctx.Done():
			return
		case <-ticker.C:
			cc.cleanupOldEvents()
		}
	}
}

// cleanupOldEvents removes old events from history
func (cc *ConcurrentCorrelator) cleanupOldEvents() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	cutoff := time.Now().Add(-cc.correlationWindow * 2)
	
	var newHistory []WatchEvent
	for _, event := range cc.eventHistory {
		if event.Timestamp.After(cutoff) {
			newHistory = append(newHistory, event)
		}
	}
	
	cc.eventHistory = newHistory
}