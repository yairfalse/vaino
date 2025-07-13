package exports

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultRouter implements the ExportRouter interface with rule-based routing
type DefaultRouter struct {
	mu      sync.RWMutex
	routes  []*Route
	plugins map[string]ExportPlugin
	config  RouterConfig
}

// RouterConfig configures router behavior
type RouterConfig struct {
	DefaultPlugin       string               `json:"default_plugin"`
	FailoverEnabled     bool                 `json:"failover_enabled"`
	LoadBalancing       bool                 `json:"load_balancing"`
	LoadBalanceStrategy string               `json:"load_balance_strategy"` // round_robin, least_connections, weighted
	CircuitBreaker      CircuitBreakerConfig `json:"circuit_breaker"`
	HealthCheckEnabled  bool                 `json:"health_check_enabled"`
	RoutingTimeout      time.Duration        `json:"routing_timeout"`
}

// CircuitBreakerConfig configures circuit breaker for plugin failures
type CircuitBreakerConfig struct {
	Enabled          bool          `json:"enabled"`
	FailureThreshold int           `json:"failure_threshold"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	HalfOpenRequests int           `json:"half_open_requests"`
}

// Route represents a routing rule
type Route struct {
	ID          string       `json:"id"`
	Pattern     RoutePattern `json:"pattern"`
	PluginName  string       `json:"plugin_name"`
	Priority    int          `json:"priority"`
	Enabled     bool         `json:"enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	MatchCount  int64        `json:"match_count"`
	LastMatched time.Time    `json:"last_matched"`

	// Internal fields
	compiledConditions []*CompiledCondition `json:"-"`
}

// CompiledCondition represents a pre-compiled routing condition
type CompiledCondition struct {
	Field    string
	Operator string
	Value    interface{}
	Regex    *regexp.Regexp // For regex operators
}

// CircuitBreaker tracks plugin health for circuit breaker functionality
type CircuitBreaker struct {
	mu           sync.RWMutex
	pluginStates map[string]*PluginState
	config       CircuitBreakerConfig
}

// PluginState tracks the state of a plugin for circuit breaker
type PluginState struct {
	State         string    `json:"state"` // closed, open, half_open
	FailureCount  int       `json:"failure_count"`
	LastFailure   time.Time `json:"last_failure"`
	LastSuccess   time.Time `json:"last_success"`
	HalfOpenCount int       `json:"half_open_count"`
	RecoveryTime  time.Time `json:"recovery_time"`
}

// NewDefaultRouter creates a new default router
func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{
		routes:  make([]*Route, 0),
		plugins: make(map[string]ExportPlugin),
		config: RouterConfig{
			FailoverEnabled:    true,
			LoadBalancing:      false,
			HealthCheckEnabled: true,
			RoutingTimeout:     5 * time.Second,
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				RecoveryTimeout:  30 * time.Second,
				HalfOpenRequests: 3,
			},
		},
	}
}

// NewRouterWithConfig creates a router with custom configuration
func NewRouterWithConfig(config RouterConfig) *DefaultRouter {
	router := NewDefaultRouter()
	router.config = config
	return router
}

// Route finds the appropriate plugin for an export request
func (r *DefaultRouter) Route(ctx context.Context, request *ExportRequest) (ExportPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Sort routes by priority (highest first)
	sortedRoutes := make([]*Route, len(r.routes))
	copy(sortedRoutes, r.routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].Priority > sortedRoutes[j].Priority
	})

	// Try to match routes in priority order
	var matchedRoutes []*Route
	for _, route := range sortedRoutes {
		if !route.Enabled {
			continue
		}

		if r.matchesRoute(request, route) {
			// Check if plugin is available (circuit breaker)
			if r.config.CircuitBreaker.Enabled {
				if !r.isPluginAvailable(route.PluginName) {
					continue
				}
			}

			matchedRoutes = append(matchedRoutes, route)

			// Update route statistics
			route.MatchCount++
			route.LastMatched = time.Now()

			// If not load balancing, return first match
			if !r.config.LoadBalancing {
				break
			}
		}
	}

	if len(matchedRoutes) == 0 {
		// No routes matched, try default plugin
		if r.config.DefaultPlugin != "" {
			plugin, exists := r.plugins[r.config.DefaultPlugin]
			if exists && r.isPluginAvailable(r.config.DefaultPlugin) {
				return plugin, nil
			}
		}
		return nil, fmt.Errorf("no suitable plugin found for request")
	}

	// Select plugin from matched routes
	selectedRoute := r.selectRoute(matchedRoutes, request)
	plugin, exists := r.plugins[selectedRoute.PluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", selectedRoute.PluginName)
	}

	return plugin, nil
}

// RegisterRoute registers a new routing rule
func (r *DefaultRouter) RegisterRoute(pattern RoutePattern, plugin ExportPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate unique ID
	routeID := fmt.Sprintf("route_%d", time.Now().UnixNano())

	// Compile conditions
	compiledConditions, err := r.compileConditions(pattern.Conditions)
	if err != nil {
		return fmt.Errorf("failed to compile route conditions: %w", err)
	}

	route := &Route{
		ID:                 routeID,
		Pattern:            pattern,
		PluginName:         plugin.Name(),
		Priority:           int(pattern.Priority),
		Enabled:            true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		compiledConditions: compiledConditions,
	}

	r.routes = append(r.routes, route)
	r.plugins[plugin.Name()] = plugin

	return nil
}

// UnregisterRoute removes a routing rule
func (r *DefaultRouter) UnregisterRoute(pattern RoutePattern) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, route := range r.routes {
		if r.patternsMatch(route.Pattern, pattern) {
			// Remove route from slice
			r.routes = append(r.routes[:i], r.routes[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("route not found")
}

// ListRoutes returns information about all routing rules
func (r *DefaultRouter) ListRoutes() []RouteInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]RouteInfo, len(r.routes))
	for i, route := range r.routes {
		routes[i] = RouteInfo{
			ID:          route.ID,
			Pattern:     route.Pattern,
			PluginName:  route.PluginName,
			Priority:    route.Priority,
			Enabled:     route.Enabled,
			CreatedAt:   route.CreatedAt,
			UpdatedAt:   route.UpdatedAt,
			MatchCount:  route.MatchCount,
			LastMatched: route.LastMatched,
		}
	}

	return routes
}

// RegisterPlugin registers a plugin with the router
func (r *DefaultRouter) RegisterPlugin(plugin ExportPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[plugin.Name()] = plugin
}

// UnregisterPlugin removes a plugin from the router
func (r *DefaultRouter) UnregisterPlugin(pluginName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, pluginName)
}

// matchesRoute checks if a request matches a routing rule
func (r *DefaultRouter) matchesRoute(request *ExportRequest, route *Route) bool {
	pattern := route.Pattern

	// Check data type
	if pattern.DataType != "" && pattern.DataType != request.DataType {
		return false
	}

	// Check format
	if pattern.Format != "" && pattern.Format != request.Format {
		return false
	}

	// Check destination
	if pattern.Destination != "" && !r.matchesDestination(request.Options.Destination, pattern.Destination) {
		return false
	}

	// Check priority
	if pattern.Priority != 0 && pattern.Priority != request.Priority {
		return false
	}

	// Check tags
	if len(pattern.Tags) > 0 {
		if !r.matchesTags(request.Options.FilterTags, pattern.Tags) {
			return false
		}
	}

	// Check custom conditions
	for _, condition := range route.compiledConditions {
		if !r.evaluateCondition(request, condition) {
			return false
		}
	}

	return true
}

// matchesDestination checks if destination patterns match
func (r *DefaultRouter) matchesDestination(requestDest, patternDest string) bool {
	if patternDest == "*" {
		return true
	}

	// Support glob-like patterns
	if strings.Contains(patternDest, "*") {
		pattern := strings.ReplaceAll(patternDest, "*", ".*")
		matched, _ := regexp.MatchString("^"+pattern+"$", requestDest)
		return matched
	}

	return requestDest == patternDest
}

// matchesTags checks if tag patterns match
func (r *DefaultRouter) matchesTags(requestTags, patternTags map[string]string) bool {
	for key, value := range patternTags {
		requestValue, exists := requestTags[key]
		if !exists {
			return false
		}

		if value != "*" && value != requestValue {
			return false
		}
	}
	return true
}

// compileConditions pre-compiles routing conditions for performance
func (r *DefaultRouter) compileConditions(conditions []RouteCondition) ([]*CompiledCondition, error) {
	compiled := make([]*CompiledCondition, len(conditions))

	for i, condition := range conditions {
		compiled[i] = &CompiledCondition{
			Field:    condition.Field,
			Operator: condition.Operator,
			Value:    condition.Value,
		}

		// Pre-compile regex patterns
		if condition.Operator == "regex" {
			if pattern, ok := condition.Value.(string); ok {
				regex, err := regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("invalid regex pattern %s: %w", pattern, err)
				}
				compiled[i].Regex = regex
			} else {
				return nil, fmt.Errorf("regex operator requires string value")
			}
		}
	}

	return compiled, nil
}

// evaluateCondition evaluates a single routing condition
func (r *DefaultRouter) evaluateCondition(request *ExportRequest, condition *CompiledCondition) bool {
	fieldValue := r.getFieldValue(request, condition.Field)
	if fieldValue == nil {
		return false
	}

	switch condition.Operator {
	case "eq":
		return r.compareValues(fieldValue, condition.Value) == 0
	case "ne":
		return r.compareValues(fieldValue, condition.Value) != 0
	case "gt":
		return r.compareValues(fieldValue, condition.Value) > 0
	case "lt":
		return r.compareValues(fieldValue, condition.Value) < 0
	case "gte":
		return r.compareValues(fieldValue, condition.Value) >= 0
	case "lte":
		return r.compareValues(fieldValue, condition.Value) <= 0
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := condition.Value.(string); ok {
				return strings.Contains(str, substr)
			}
		}
		return false
	case "regex":
		if str, ok := fieldValue.(string); ok && condition.Regex != nil {
			return condition.Regex.MatchString(str)
		}
		return false
	case "in":
		if list, ok := condition.Value.([]interface{}); ok {
			for _, item := range list {
				if r.compareValues(fieldValue, item) == 0 {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

// getFieldValue extracts a field value from the request
func (r *DefaultRouter) getFieldValue(request *ExportRequest, field string) interface{} {
	parts := strings.Split(field, ".")

	switch parts[0] {
	case "data_type":
		return string(request.DataType)
	case "format":
		return string(request.Format)
	case "priority":
		return int(request.Priority)
	case "plugin_name":
		return request.PluginName
	case "async":
		return request.Async
	case "options":
		if len(parts) > 1 {
			return r.getOptionsValue(request.Options, parts[1:])
		}
		return nil
	case "metadata":
		if len(parts) > 1 {
			if value, exists := request.Metadata[parts[1]]; exists {
				return value
			}
		}
		return nil
	default:
		return nil
	}
}

// getOptionsValue extracts a value from export options
func (r *DefaultRouter) getOptionsValue(options ExportOptions, path []string) interface{} {
	switch path[0] {
	case "destination":
		return options.Destination
	case "compress":
		return options.Compress
	case "pretty":
		return options.Pretty
	case "async":
		return options.Async
	case "timeout":
		return options.Timeout
	case "filter_level":
		return options.FilterLevel
	default:
		if len(path) == 1 {
			// Check plugin options
			if value, exists := options.PluginOptions[path[0]]; exists {
				return value
			}
		}
		return nil
	}
}

// compareValues compares two values and returns -1, 0, or 1
func (r *DefaultRouter) compareValues(a, b interface{}) int {
	// Convert to strings for comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// Try numeric comparison first
	if aNum, aErr := strconv.ParseFloat(aStr, 64); aErr == nil {
		if bNum, bErr := strconv.ParseFloat(bStr, 64); bErr == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// selectRoute selects a route from multiple matched routes based on load balancing strategy
func (r *DefaultRouter) selectRoute(routes []*Route, request *ExportRequest) *Route {
	if len(routes) == 1 {
		return routes[0]
	}

	switch r.config.LoadBalanceStrategy {
	case "round_robin":
		return r.selectRoundRobin(routes)
	case "least_connections":
		return r.selectLeastConnections(routes)
	case "weighted":
		return r.selectWeighted(routes)
	default:
		return routes[0] // Default to first match
	}
}

// selectRoundRobin implements round-robin load balancing
func (r *DefaultRouter) selectRoundRobin(routes []*Route) *Route {
	// Simple round-robin based on match count
	minMatches := routes[0].MatchCount
	selectedRoute := routes[0]

	for _, route := range routes[1:] {
		if route.MatchCount < minMatches {
			minMatches = route.MatchCount
			selectedRoute = route
		}
	}

	return selectedRoute
}

// selectLeastConnections selects the plugin with least active connections
func (r *DefaultRouter) selectLeastConnections(routes []*Route) *Route {
	// In a real implementation, this would check active connections
	// For now, use the route with least recent activity
	oldestActivity := routes[0].LastMatched
	selectedRoute := routes[0]

	for _, route := range routes[1:] {
		if route.LastMatched.Before(oldestActivity) {
			oldestActivity = route.LastMatched
			selectedRoute = route
		}
	}

	return selectedRoute
}

// selectWeighted implements weighted load balancing
func (r *DefaultRouter) selectWeighted(routes []*Route) *Route {
	// Use priority as weight
	totalWeight := 0
	for _, route := range routes {
		totalWeight += route.Priority
	}

	if totalWeight == 0 {
		return routes[0]
	}

	// Simple weighted selection (in production would use proper algorithm)
	target := int(time.Now().UnixNano()) % totalWeight
	current := 0

	for _, route := range routes {
		current += route.Priority
		if current > target {
			return route
		}
	}

	return routes[0]
}

// isPluginAvailable checks if a plugin is available (not in circuit breaker open state)
func (r *DefaultRouter) isPluginAvailable(pluginName string) bool {
	if !r.config.CircuitBreaker.Enabled {
		return true
	}

	// In a real implementation, this would check circuit breaker state
	// For now, assume all plugins are available
	return true
}

// patternsMatch checks if two route patterns are equivalent
func (r *DefaultRouter) patternsMatch(a, b RoutePattern) bool {
	return a.DataType == b.DataType &&
		a.Format == b.Format &&
		a.Destination == b.Destination &&
		a.Priority == b.Priority &&
		r.tagsMatch(a.Tags, b.Tags) &&
		r.conditionsMatch(a.Conditions, b.Conditions)
}

// tagsMatch checks if two tag maps are equivalent
func (r *DefaultRouter) tagsMatch(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for key, value := range a {
		if bValue, exists := b[key]; !exists || bValue != value {
			return false
		}
	}

	return true
}

// conditionsMatch checks if two condition slices are equivalent
func (r *DefaultRouter) conditionsMatch(a, b []RouteCondition) bool {
	if len(a) != len(b) {
		return false
	}

	for i, condA := range a {
		condB := b[i]
		if condA.Field != condB.Field ||
			condA.Operator != condB.Operator ||
			fmt.Sprintf("%v", condA.Value) != fmt.Sprintf("%v", condB.Value) {
			return false
		}
	}

	return true
}

// UpdateConfig updates the router configuration
func (r *DefaultRouter) UpdateConfig(config RouterConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
}

// GetConfig returns the current router configuration
func (r *DefaultRouter) GetConfig() RouterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// GetStats returns routing statistics
func (r *DefaultRouter) GetStats() RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalMatches := int64(0)
	activeRoutes := 0

	for _, route := range r.routes {
		if route.Enabled {
			activeRoutes++
		}
		totalMatches += route.MatchCount
	}

	return RouterStats{
		TotalRoutes:  len(r.routes),
		ActiveRoutes: activeRoutes,
		TotalMatches: totalMatches,
		TotalPlugins: len(r.plugins),
		LastUpdated:  time.Now(),
	}
}

// RouterStats contains router performance statistics
type RouterStats struct {
	TotalRoutes  int       `json:"total_routes"`
	ActiveRoutes int       `json:"active_routes"`
	TotalMatches int64     `json:"total_matches"`
	TotalPlugins int       `json:"total_plugins"`
	LastUpdated  time.Time `json:"last_updated"`
}
