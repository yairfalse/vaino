package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	mu                sync.RWMutex
	enabled           bool
	metricRegistry    map[string]*Metric
	aggregators       map[string]*MetricAggregator
	exporters         []MetricExporter
	alertManager      *AlertManager
	dashboardManager *DashboardManager
	ctx               context.Context
	cancel            context.CancelFunc
	collectInterval   time.Duration
	retentionPeriod   time.Duration
	stats             MetricsCollectorStats
}

// Metric represents a single metric
type Metric struct {
	mu           sync.RWMutex
	Name         string            `json:"name"`
	Type         MetricType        `json:"type"`
	Description  string            `json:"description"`
	Unit         string            `json:"unit"`
	Tags         map[string]string `json:"tags"`
	Values       []MetricValue     `json:"values"`
	CurrentValue float64           `json:"current_value"`
	LastUpdated  time.Time         `json:"last_updated"`
	Aggregation  AggregationType   `json:"aggregation"`
	Retention    time.Duration     `json:"retention"`
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
	MetricTypeTimer     MetricType = "timer"
)

// AggregationType represents how to aggregate metric values
type AggregationType string

const (
	AggregationSum     AggregationType = "sum"
	AggregationAverage AggregationType = "average"
	AggregationMin     AggregationType = "min"
	AggregationMax     AggregationType = "max"
	AggregationP50     AggregationType = "p50"
	AggregationP95     AggregationType = "p95"
	AggregationP99     AggregationType = "p99"
)

// MetricValue represents a single metric value with timestamp
type MetricValue struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

// MetricAggregator aggregates metric values over time windows
type MetricAggregator struct {
	mu                sync.RWMutex
	MetricName        string                        `json:"metric_name"`
	WindowSize        time.Duration                 `json:"window_size"`
	AggregationType   AggregationType              `json:"aggregation_type"`
	Windows           map[string]*AggregationWindow `json:"windows"`
	MaxWindows        int                           `json:"max_windows"`
	Stats             MetricAggregatorStats         `json:"stats"`
}

// AggregationWindow represents a time window for aggregation
type AggregationWindow struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Values       []float64     `json:"values"`
	Count        int64         `json:"count"`
	Sum          float64       `json:"sum"`
	Min          float64       `json:"min"`
	Max          float64       `json:"max"`
	Average      float64       `json:"average"`
	Percentiles  map[string]float64 `json:"percentiles"`
	AggregatedValue float64    `json:"aggregated_value"`
}

// MetricExporter interface for exporting metrics
type MetricExporter interface {
	Export(metrics map[string]*Metric) error
	Name() string
	IsEnabled() bool
}

// AlertManager manages metric-based alerts
type AlertManager struct {
	mu              sync.RWMutex
	enabled         bool
	alertRules      map[string]*AlertRule
	activeAlerts    map[string]*Alert
	alertHistory    []Alert
	maxHistorySize  int
	checkInterval   time.Duration
	notifications   []NotificationChannel
	stats           AlertManagerStats
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	MetricName  string            `json:"metric_name"`
	Condition   AlertCondition    `json:"condition"`
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"`
	Severity    AlertSeverity     `json:"severity"`
	Tags        map[string]string `json:"tags"`
	Enabled     bool              `json:"enabled"`
	LastTriggered time.Time       `json:"last_triggered"`
	TriggerCount int64            `json:"trigger_count"`
}

// AlertCondition represents alert conditions
type AlertCondition string

const (
	AlertConditionGreaterThan    AlertCondition = "greater_than"
	AlertConditionLessThan       AlertCondition = "less_than"
	AlertConditionEquals         AlertCondition = "equals"
	AlertConditionNotEquals      AlertCondition = "not_equals"
	AlertConditionIncreasing     AlertCondition = "increasing"
	AlertConditionDecreasing     AlertCondition = "decreasing"
	AlertConditionMissing        AlertCondition = "missing"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents an active or historical alert
type Alert struct {
	ID           string            `json:"id"`
	RuleID       string            `json:"rule_id"`
	RuleName     string            `json:"rule_name"`
	MetricName   string            `json:"metric_name"`
	Condition    AlertCondition    `json:"condition"`
	Threshold    float64           `json:"threshold"`
	CurrentValue float64           `json:"current_value"`
	Severity     AlertSeverity     `json:"severity"`
	Status       AlertStatus       `json:"status"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Tags         map[string]string `json:"tags"`
	Message      string            `json:"message"`
	Acknowledged bool              `json:"acknowledged"`
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// NotificationChannel interface for sending alert notifications
type NotificationChannel interface {
	Send(alert Alert) error
	Name() string
	IsEnabled() bool
}

// DashboardManager manages performance dashboards
type DashboardManager struct {
	mu          sync.RWMutex
	enabled     bool
	dashboards  map[string]*Dashboard
	widgets     map[string]*Widget
	templates   map[string]*DashboardTemplate
	stats       DashboardManagerStats
}

// Dashboard represents a performance dashboard
type Dashboard struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Widgets     []string          `json:"widgets"`
	Layout      DashboardLayout   `json:"layout"`
	RefreshRate time.Duration     `json:"refresh_rate"`
	Tags        map[string]string `json:"tags"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Widget represents a dashboard widget
type Widget struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        WidgetType        `json:"type"`
	MetricNames []string          `json:"metric_names"`
	TimeRange   time.Duration     `json:"time_range"`
	Aggregation AggregationType   `json:"aggregation"`
	Position    WidgetPosition    `json:"position"`
	Size        WidgetSize        `json:"size"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool              `json:"enabled"`
}

// WidgetType represents the type of widget
type WidgetType string

const (
	WidgetTypeChart     WidgetType = "chart"
	WidgetTypeGauge     WidgetType = "gauge"
	WidgetTypeTable     WidgetType = "table"
	WidgetTypeText      WidgetType = "text"
	WidgetTypeAlert     WidgetType = "alert"
	WidgetTypeMetric    WidgetType = "metric"
)

// DashboardLayout represents dashboard layout configuration
type DashboardLayout struct {
	Columns int `json:"columns"`
	Rows    int `json:"rows"`
}

// WidgetPosition represents widget position
type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WidgetSize represents widget size
type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardTemplate represents a dashboard template
type DashboardTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Template    Dashboard         `json:"template"`
	Variables   map[string]string `json:"variables"`
	Tags        []string          `json:"tags"`
}

// Statistics structures
type MetricsCollectorStats struct {
	TotalMetrics        int64     `json:"total_metrics"`
	ActiveMetrics       int64     `json:"active_metrics"`
	MetricsPerSecond    float64   `json:"metrics_per_second"`
	TotalValues         int64     `json:"total_values"`
	AverageLatency      time.Duration `json:"average_latency"`
	ErrorCount          int64     `json:"error_count"`
	ExportCount         int64     `json:"export_count"`
	LastCollection      time.Time `json:"last_collection"`
	MemoryUsage         int64     `json:"memory_usage"`
}

type MetricAggregatorStats struct {
	TotalWindows       int64     `json:"total_windows"`
	ActiveWindows      int64     `json:"active_windows"`
	AggregationsPerSecond float64 `json:"aggregations_per_second"`
	AverageWindowSize  time.Duration `json:"average_window_size"`
	LastAggregation    time.Time `json:"last_aggregation"`
}

type AlertManagerStats struct {
	TotalRules        int64     `json:"total_rules"`
	ActiveRules       int64     `json:"active_rules"`
	TotalAlerts       int64     `json:"total_alerts"`
	ActiveAlerts      int64     `json:"active_alerts"`
	ResolvedAlerts    int64     `json:"resolved_alerts"`
	AlertsPerHour     float64   `json:"alerts_per_hour"`
	AverageResolution time.Duration `json:"average_resolution"`
	LastAlert         time.Time `json:"last_alert"`
}

type DashboardManagerStats struct {
	TotalDashboards   int64     `json:"total_dashboards"`
	ActiveDashboards  int64     `json:"active_dashboards"`
	TotalWidgets      int64     `json:"total_widgets"`
	ActiveWidgets     int64     `json:"active_widgets"`
	ViewsPerHour      float64   `json:"views_per_hour"`
	LastUpdate        time.Time `json:"last_update"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MetricsCollector{
		enabled:           true,
		metricRegistry:    make(map[string]*Metric),
		aggregators:       make(map[string]*MetricAggregator),
		exporters:         []MetricExporter{},
		alertManager:      NewAlertManager(),
		dashboardManager:  NewDashboardManager(),
		ctx:               ctx,
		cancel:            cancel,
		collectInterval:   10 * time.Second,
		retentionPeriod:   24 * time.Hour,
		stats:             MetricsCollectorStats{},
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		enabled:         true,
		alertRules:      make(map[string]*AlertRule),
		activeAlerts:    make(map[string]*Alert),
		alertHistory:    []Alert{},
		maxHistorySize:  1000,
		checkInterval:   30 * time.Second,
		notifications:   []NotificationChannel{},
		stats:           AlertManagerStats{},
	}
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager() *DashboardManager {
	return &DashboardManager{
		enabled:    true,
		dashboards: make(map[string]*Dashboard),
		widgets:    make(map[string]*Widget),
		templates:  make(map[string]*DashboardTemplate),
		stats:      DashboardManagerStats{},
	}
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if !mc.enabled {
		return fmt.Errorf("metrics collector is disabled")
	}
	
	// Start alert manager
	if err := mc.alertManager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}
	
	// Start dashboard manager
	if err := mc.dashboardManager.Start(); err != nil {
		return fmt.Errorf("failed to start dashboard manager: %w", err)
	}
	
	// Start collection loop
	go mc.collectionLoop()
	
	// Start export loop
	go mc.exportLoop()
	
	// Start cleanup loop
	go mc.cleanupLoop()
	
	return nil
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cancel()
	
	// Stop components
	mc.alertManager.Stop()
	mc.dashboardManager.Stop()
	
	return nil
}

// RegisterMetric registers a new metric
func (mc *MetricsCollector) RegisterMetric(name string, metricType MetricType, description, unit string, tags map[string]string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if _, exists := mc.metricRegistry[name]; exists {
		return fmt.Errorf("metric %s already exists", name)
	}
	
	metric := &Metric{
		Name:        name,
		Type:        metricType,
		Description: description,
		Unit:        unit,
		Tags:        tags,
		Values:      []MetricValue{},
		LastUpdated: time.Now(),
		Aggregation: AggregationAverage,
		Retention:   mc.retentionPeriod,
	}
	
	mc.metricRegistry[name] = metric
	mc.stats.TotalMetrics++
	
	return nil
}

// RecordValue records a value for a metric
func (mc *MetricsCollector) RecordValue(name string, value float64, tags map[string]string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	metric, exists := mc.metricRegistry[name]
	if !exists {
		return fmt.Errorf("metric %s not found", name)
	}
	
	metric.mu.Lock()
	defer metric.mu.Unlock()
	
	metricValue := MetricValue{
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
	}
	
	metric.Values = append(metric.Values, metricValue)
	metric.CurrentValue = value
	metric.LastUpdated = time.Now()
	
	// Update aggregators
	if aggregator, exists := mc.aggregators[name]; exists {
		aggregator.AddValue(value, time.Now())
	}
	
	mc.stats.TotalValues++
	
	return nil
}

// GetMetric returns a metric by name
func (mc *MetricsCollector) GetMetric(name string) (*Metric, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	metric, exists := mc.metricRegistry[name]
	if !exists {
		return nil, fmt.Errorf("metric %s not found", name)
	}
	
	return metric, nil
}

// GetAllMetrics returns all metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	metrics := make(map[string]*Metric)
	for name, metric := range mc.metricRegistry {
		metrics[name] = metric
	}
	
	return metrics
}

// CreateAggregator creates a metric aggregator
func (mc *MetricsCollector) CreateAggregator(metricName string, windowSize time.Duration, aggregationType AggregationType) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	aggregator := &MetricAggregator{
		MetricName:      metricName,
		WindowSize:      windowSize,
		AggregationType: aggregationType,
		Windows:         make(map[string]*AggregationWindow),
		MaxWindows:      100,
		Stats:           MetricAggregatorStats{},
	}
	
	mc.aggregators[metricName] = aggregator
	return nil
}

// GetStats returns metrics collector statistics
func (mc *MetricsCollector) GetStats() MetricsCollectorStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	mc.stats.ActiveMetrics = int64(len(mc.metricRegistry))
	mc.stats.LastCollection = time.Now()
	
	return mc.stats
}

// Collection loop
func (mc *MetricsCollector) collectionLoop() {
	ticker := time.NewTicker(mc.collectInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.collectSystemMetrics()
		}
	}
}

// Export loop
func (mc *MetricsCollector) exportLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.exportMetrics()
		}
	}
}

// Cleanup loop
func (mc *MetricsCollector) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.cleanupOldMetrics()
		}
	}
}

// collectSystemMetrics collects system metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	// Record basic system metrics
	mc.RecordValue("wgo.watchers.concurrent.active", float64(1), map[string]string{"type": "system"})
	mc.RecordValue("wgo.watchers.memory.usage", float64(1024*1024), map[string]string{"type": "system"})
	mc.RecordValue("wgo.watchers.events.processed", float64(100), map[string]string{"type": "system"})
}

// exportMetrics exports metrics to registered exporters
func (mc *MetricsCollector) exportMetrics() {
	metrics := mc.GetAllMetrics()
	
	for _, exporter := range mc.exporters {
		if exporter.IsEnabled() {
			if err := exporter.Export(metrics); err != nil {
				mc.stats.ErrorCount++
			} else {
				mc.stats.ExportCount++
			}
		}
	}
}

// cleanupOldMetrics removes old metric values
func (mc *MetricsCollector) cleanupOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	cutoff := time.Now().Add(-mc.retentionPeriod)
	
	for _, metric := range mc.metricRegistry {
		metric.mu.Lock()
		
		var newValues []MetricValue
		for _, value := range metric.Values {
			if value.Timestamp.After(cutoff) {
				newValues = append(newValues, value)
			}
		}
		
		metric.Values = newValues
		metric.mu.Unlock()
	}
}

// MetricAggregator methods
func (ma *MetricAggregator) AddValue(value float64, timestamp time.Time) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	// Determine which window this value belongs to
	windowKey := ma.getWindowKey(timestamp)
	
	window, exists := ma.Windows[windowKey]
	if !exists {
		window = &AggregationWindow{
			StartTime:   timestamp.Truncate(ma.WindowSize),
			EndTime:     timestamp.Truncate(ma.WindowSize).Add(ma.WindowSize),
			Values:      []float64{},
			Count:       0,
			Sum:         0,
			Min:         value,
			Max:         value,
			Percentiles: make(map[string]float64),
		}
		ma.Windows[windowKey] = window
	}
	
	// Add value to window
	window.Values = append(window.Values, value)
	window.Count++
	window.Sum += value
	
	if value < window.Min {
		window.Min = value
	}
	if value > window.Max {
		window.Max = value
	}
	
	// Calculate average
	window.Average = window.Sum / float64(window.Count)
	
	// Calculate aggregated value based on type
	switch ma.AggregationType {
	case AggregationSum:
		window.AggregatedValue = window.Sum
	case AggregationAverage:
		window.AggregatedValue = window.Average
	case AggregationMin:
		window.AggregatedValue = window.Min
	case AggregationMax:
		window.AggregatedValue = window.Max
	}
	
	// Cleanup old windows
	if len(ma.Windows) > ma.MaxWindows {
		ma.cleanupOldWindows()
	}
	
	ma.Stats.LastAggregation = time.Now()
}

func (ma *MetricAggregator) getWindowKey(timestamp time.Time) string {
	return timestamp.Truncate(ma.WindowSize).Format("2006-01-02T15:04:05")
}

func (ma *MetricAggregator) cleanupOldWindows() {
	// Keep only the most recent windows
	windowKeys := make([]string, 0, len(ma.Windows))
	for key := range ma.Windows {
		windowKeys = append(windowKeys, key)
	}
	
	// Sort by key (timestamp)
	for i := 0; i < len(windowKeys)-1; i++ {
		for j := i + 1; j < len(windowKeys); j++ {
			if windowKeys[i] > windowKeys[j] {
				windowKeys[i], windowKeys[j] = windowKeys[j], windowKeys[i]
			}
		}
	}
	
	// Remove oldest windows
	if len(windowKeys) > ma.MaxWindows {
		toRemove := windowKeys[:len(windowKeys)-ma.MaxWindows]
		for _, key := range toRemove {
			delete(ma.Windows, key)
		}
	}
}

// AlertManager methods
func (am *AlertManager) Start() error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if !am.enabled {
		return fmt.Errorf("alert manager is disabled")
	}
	
	go am.alertLoop()
	return nil
}

func (am *AlertManager) Stop() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.enabled = false
}

func (am *AlertManager) alertLoop() {
	ticker := time.NewTicker(am.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			am.checkAlerts()
		}
	}
}

func (am *AlertManager) checkAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if !am.enabled {
		return
	}
	
	// Check all alert rules
	for _, rule := range am.alertRules {
		if rule.Enabled {
			am.evaluateRule(rule)
		}
	}
}

func (am *AlertManager) evaluateRule(rule *AlertRule) {
	// This is a simplified implementation
	// In a real implementation, you'd fetch the current metric value
	// and compare it against the rule conditions
	
	// For now, just create a mock alert
	if time.Since(rule.LastTriggered) > 5*time.Minute {
		alert := &Alert{
			ID:           fmt.Sprintf("alert-%s-%d", rule.ID, time.Now().UnixNano()),
			RuleID:       rule.ID,
			RuleName:     rule.Name,
			MetricName:   rule.MetricName,
			Condition:    rule.Condition,
			Threshold:    rule.Threshold,
			CurrentValue: rule.Threshold + 10, // Mock value
			Severity:     rule.Severity,
			Status:       AlertStatusActive,
			StartTime:    time.Now(),
			Tags:         rule.Tags,
			Message:      fmt.Sprintf("Alert triggered for metric %s", rule.MetricName),
		}
		
		am.activeAlerts[alert.ID] = alert
		am.alertHistory = append(am.alertHistory, *alert)
		
		rule.LastTriggered = time.Now()
		rule.TriggerCount++
		
		// Send notifications
		for _, channel := range am.notifications {
			if channel.IsEnabled() {
				channel.Send(*alert)
			}
		}
		
		am.stats.TotalAlerts++
		am.stats.ActiveAlerts++
		am.stats.LastAlert = time.Now()
	}
}

// DashboardManager methods
func (dm *DashboardManager) Start() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if !dm.enabled {
		return fmt.Errorf("dashboard manager is disabled")
	}
	
	return nil
}

func (dm *DashboardManager) Stop() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	dm.enabled = false
}

func (dm *DashboardManager) CreateDashboard(dashboard *Dashboard) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if _, exists := dm.dashboards[dashboard.ID]; exists {
		return fmt.Errorf("dashboard %s already exists", dashboard.ID)
	}
	
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()
	
	dm.dashboards[dashboard.ID] = dashboard
	dm.stats.TotalDashboards++
	
	return nil
}

func (dm *DashboardManager) GetDashboard(id string) (*Dashboard, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	dashboard, exists := dm.dashboards[id]
	if !exists {
		return nil, fmt.Errorf("dashboard %s not found", id)
	}
	
	return dashboard, nil
}

// Helper methods to create default metrics
func (mc *MetricsCollector) RegisterDefaultMetrics() {
	// Watch system metrics
	mc.RegisterMetric("wgo.watchers.concurrent.active", MetricTypeGauge, "Number of active concurrent watchers", "count", map[string]string{"component": "watchers"})
	mc.RegisterMetric("wgo.watchers.events.processed", MetricTypeCounter, "Total events processed", "count", map[string]string{"component": "watchers"})
	mc.RegisterMetric("wgo.watchers.events.rate", MetricTypeGauge, "Events processed per second", "rate", map[string]string{"component": "watchers"})
	mc.RegisterMetric("wgo.watchers.memory.usage", MetricTypeGauge, "Memory usage", "bytes", map[string]string{"component": "watchers"})
	mc.RegisterMetric("wgo.watchers.correlation.rate", MetricTypeGauge, "Event correlation rate", "rate", map[string]string{"component": "correlator"})
	mc.RegisterMetric("wgo.watchers.scan.duration", MetricTypeTimer, "Scan duration", "seconds", map[string]string{"component": "scanner"})
	mc.RegisterMetric("wgo.watchers.pipeline.latency", MetricTypeTimer, "Pipeline processing latency", "seconds", map[string]string{"component": "pipeline"})
	mc.RegisterMetric("wgo.watchers.cache.hit_rate", MetricTypeGauge, "Cache hit rate", "percentage", map[string]string{"component": "cache"})
	mc.RegisterMetric("wgo.watchers.errors.count", MetricTypeCounter, "Total errors", "count", map[string]string{"component": "watchers"})
	mc.RegisterMetric("wgo.watchers.gc.duration", MetricTypeTimer, "Garbage collection duration", "seconds", map[string]string{"component": "gc"})
	
	// Create default aggregators
	mc.CreateAggregator("wgo.watchers.events.rate", 1*time.Minute, AggregationAverage)
	mc.CreateAggregator("wgo.watchers.scan.duration", 5*time.Minute, AggregationP95)
	mc.CreateAggregator("wgo.watchers.pipeline.latency", 1*time.Minute, AggregationP99)
}

// IsEnabled returns whether the metrics collector is enabled
func (mc *MetricsCollector) IsEnabled() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.enabled
}

// SetEnabled enables or disables the metrics collector
func (mc *MetricsCollector) SetEnabled(enabled bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = enabled
}