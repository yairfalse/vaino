package plugins

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/exports"
	"github.com/yairfalse/vaino/pkg/types"
)

// PrometheusExportPlugin implements ExportPlugin for Prometheus metrics export
type PrometheusExportPlugin struct {
	mu          sync.RWMutex
	config      exports.PluginConfig
	running     bool
	startTime   time.Time
	metrics     exports.PluginMetrics
	endpoint    string
	pushGateway string
	jobName     string
	instance    string
	namespace   string

	// Metric storage
	gauges     map[string]*PrometheusGauge
	counters   map[string]*PrometheusCounter
	histograms map[string]*PrometheusHistogram

	// HTTP client for push gateway
	httpTimeout time.Duration
	userAgent   string
}

// PrometheusGauge represents a Prometheus gauge metric
type PrometheusGauge struct {
	Name      string            `json:"name"`
	Help      string            `json:"help"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

// PrometheusCounter represents a Prometheus counter metric
type PrometheusCounter struct {
	Name      string            `json:"name"`
	Help      string            `json:"help"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

// PrometheusHistogram represents a Prometheus histogram metric
type PrometheusHistogram struct {
	Name      string             `json:"name"`
	Help      string             `json:"help"`
	Buckets   map[string]float64 `json:"buckets"` // bucket le -> count
	Count     float64            `json:"count"`
	Sum       float64            `json:"sum"`
	Labels    map[string]string  `json:"labels"`
	Timestamp time.Time          `json:"timestamp"`
}

// NewPrometheusExportPlugin creates a new Prometheus export plugin
func NewPrometheusExportPlugin() *PrometheusExportPlugin {
	return &PrometheusExportPlugin{
		config: exports.PluginConfig{
			Name:    "prometheus",
			Version: "1.0.0",
			Enabled: true,
			Settings: map[string]interface{}{
				"push_gateway":  "http://localhost:9091",
				"job_name":      "vaino",
				"instance":      "localhost:8080",
				"namespace":     "vaino",
				"timeout":       "30s",
				"enable_push":   true,
				"metric_prefix": "vaino_",
			},
		},
		pushGateway: "http://localhost:9091",
		jobName:     "vaino",
		instance:    "localhost:8080",
		namespace:   "vaino",
		httpTimeout: 30 * time.Second,
		userAgent:   "vaino-prometheus-exporter/1.0.0",
		gauges:      make(map[string]*PrometheusGauge),
		counters:    make(map[string]*PrometheusCounter),
		histograms:  make(map[string]*PrometheusHistogram),
		startTime:   time.Now(),
	}
}

// Plugin metadata methods

func (p *PrometheusExportPlugin) Name() string {
	return "prometheus"
}

func (p *PrometheusExportPlugin) Version() string {
	return "1.0.0"
}

func (p *PrometheusExportPlugin) Description() string {
	return "Prometheus metrics export plugin for infrastructure drift monitoring"
}

func (p *PrometheusExportPlugin) SupportedFormats() []exports.ExportFormat {
	return []exports.ExportFormat{
		exports.FormatPrometheus,
	}
}

// Plugin lifecycle methods

func (p *PrometheusExportPlugin) Initialize(ctx context.Context, config exports.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if config.Name != "" {
		p.config = config
	}

	// Extract settings
	if pushGateway, ok := p.config.Settings["push_gateway"].(string); ok {
		p.pushGateway = pushGateway
	}

	if jobName, ok := p.config.Settings["job_name"].(string); ok {
		p.jobName = jobName
	}

	if instance, ok := p.config.Settings["instance"].(string); ok {
		p.instance = instance
	}

	if namespace, ok := p.config.Settings["namespace"].(string); ok {
		p.namespace = namespace
	}

	if timeoutStr, ok := p.config.Settings["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			p.httpTimeout = timeout
		}
	}

	p.metrics.MetricsUpdatedAt = time.Now()
	return nil
}

func (p *PrometheusExportPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("Prometheus plugin already running")
	}

	p.running = true
	p.startTime = time.Now()

	return nil
}

func (p *PrometheusExportPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = false
	return nil
}

func (p *PrometheusExportPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Export methods

func (p *PrometheusExportPlugin) Export(ctx context.Context, request *exports.ExportRequest) (*exports.ExportResponse, error) {
	startTime := time.Now()
	defer func() {
		p.updateMetrics(time.Since(startTime))
	}()

	switch request.DataType {
	case exports.DataTypeDriftReport:
		if report, ok := request.Data.(*differ.DriftReport); ok {
			return p.exportDriftReport(ctx, report, request.Options)
		}
		return nil, fmt.Errorf("invalid data type for drift report export")

	case exports.DataTypeSnapshot:
		if snapshot, ok := request.Data.(*types.Snapshot); ok {
			return p.exportSnapshot(ctx, snapshot, request.Options)
		}
		return nil, fmt.Errorf("invalid data type for snapshot export")

	case exports.DataTypeCorrelation:
		if correlation, ok := request.Data.(*exports.CorrelationData); ok {
			return p.exportCorrelation(ctx, correlation, request.Options)
		}
		return nil, fmt.Errorf("invalid data type for correlation export")

	case exports.DataTypeMetrics:
		return p.exportMetrics(ctx, request.Data, request.Options)

	default:
		return nil, fmt.Errorf("unsupported data type: %s", request.DataType)
	}
}

func (p *PrometheusExportPlugin) ExportDriftReport(ctx context.Context, report *differ.DriftReport, options exports.ExportOptions) error {
	_, err := p.exportDriftReport(ctx, report, options)
	return err
}

func (p *PrometheusExportPlugin) ExportSnapshot(ctx context.Context, snapshot *types.Snapshot, options exports.ExportOptions) error {
	_, err := p.exportSnapshot(ctx, snapshot, options)
	return err
}

func (p *PrometheusExportPlugin) ExportCorrelation(ctx context.Context, correlation *exports.CorrelationData, options exports.ExportOptions) error {
	_, err := p.exportCorrelation(ctx, correlation, options)
	return err
}

// Internal export methods

func (p *PrometheusExportPlugin) exportDriftReport(ctx context.Context, report *differ.DriftReport, options exports.ExportOptions) (*exports.ExportResponse, error) {
	timestamp := time.Now()
	baseLabels := map[string]string{
		"baseline_id": report.BaselineID,
		"current_id":  report.CurrentID,
		"report_id":   report.ID,
	}

	// Drift summary metrics
	p.setGauge("drift_report_risk_score", report.Summary.RiskScore, baseLabels, "Risk score of the drift report", timestamp)
	p.setGauge("drift_report_total_resources", float64(report.Summary.TotalResources), baseLabels, "Total number of resources in the report", timestamp)
	p.setGauge("drift_report_changed_resources", float64(report.Summary.ChangedResources), baseLabels, "Number of changed resources", timestamp)
	p.setGauge("drift_report_added_resources", float64(report.Summary.AddedResources), baseLabels, "Number of added resources", timestamp)
	p.setGauge("drift_report_removed_resources", float64(report.Summary.RemovedResources), baseLabels, "Number of removed resources", timestamp)
	p.setGauge("drift_report_modified_resources", float64(report.Summary.ModifiedResources), baseLabels, "Number of modified resources", timestamp)

	// Changes by severity
	severityCounts := map[string]float64{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	for severity, count := range report.Summary.ChangesBySeverity {
		severityCounts[string(severity)] = float64(count)
	}

	for severity, count := range severityCounts {
		labels := copyLabels(baseLabels)
		labels["severity"] = severity
		p.setGauge("drift_report_changes_by_severity", count, labels, "Number of changes by severity level", timestamp)
	}

	// Individual resource changes
	for _, change := range report.ResourceChanges {
		changeLabels := copyLabels(baseLabels)
		changeLabels["resource_id"] = change.ResourceID
		changeLabels["resource_type"] = change.ResourceType
		changeLabels["change_type"] = string(change.DriftType)
		changeLabels["severity"] = string(change.Severity)
		changeLabels["category"] = string(change.Category)

		p.incrementCounter("drift_resource_changes_total", changeLabels, "Total number of resource changes", timestamp)
		p.setGauge("drift_resource_change_risk_score", change.RiskScore, changeLabels, "Risk score of individual resource change", timestamp)
	}

	// Generate Prometheus exposition format
	data, err := p.generatePrometheusExposition()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Prometheus exposition: %w", err)
	}

	// Optionally push to push gateway
	if enablePush, ok := p.config.Settings["enable_push"].(bool); ok && enablePush {
		if err := p.pushToPushGateway(ctx, data); err != nil {
			// Log error but don't fail the export
			fmt.Printf("Warning: Failed to push to Prometheus push gateway: %v\n", err)
		}
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        data,
		ContentType: "text/plain; version=0.0.4",
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"metrics_count":   len(p.gauges) + len(p.counters) + len(p.histograms),
			"report_id":       report.ID,
			"push_gateway":    p.pushGateway,
			"exposition_size": len(data),
		},
	}, nil
}

func (p *PrometheusExportPlugin) exportSnapshot(ctx context.Context, snapshot *types.Snapshot, options exports.ExportOptions) (*exports.ExportResponse, error) {
	timestamp := time.Now()
	baseLabels := map[string]string{
		"snapshot_id": snapshot.ID,
		"provider":    snapshot.Provider,
	}

	// Snapshot metrics
	p.setGauge("snapshot_resources_total", float64(len(snapshot.Resources)), baseLabels, "Total number of resources in snapshot", timestamp)
	p.setGauge("snapshot_timestamp", float64(snapshot.Timestamp.Unix()), baseLabels, "Timestamp of the snapshot", timestamp)

	// Resources by type
	resourceCounts := make(map[string]int)
	for _, resource := range snapshot.Resources {
		resourceCounts[resource.Type]++
	}

	for resourceType, count := range resourceCounts {
		typeLabels := copyLabels(baseLabels)
		typeLabels["resource_type"] = resourceType
		p.setGauge("snapshot_resources_by_type", float64(count), typeLabels, "Number of resources by type in snapshot", timestamp)
	}

	// Generate and potentially push metrics
	data, err := p.generatePrometheusExposition()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Prometheus exposition: %w", err)
	}

	if enablePush, ok := p.config.Settings["enable_push"].(bool); ok && enablePush {
		if err := p.pushToPushGateway(ctx, data); err != nil {
			fmt.Printf("Warning: Failed to push to Prometheus push gateway: %v\n", err)
		}
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        data,
		ContentType: "text/plain; version=0.0.4",
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"metrics_count":   len(p.gauges) + len(p.counters) + len(p.histograms),
			"snapshot_id":     snapshot.ID,
			"resource_count":  len(snapshot.Resources),
			"exposition_size": len(data),
		},
	}, nil
}

func (p *PrometheusExportPlugin) exportCorrelation(ctx context.Context, correlation *exports.CorrelationData, options exports.ExportOptions) (*exports.ExportResponse, error) {
	timestamp := time.Now()
	baseLabels := map[string]string{
		"correlation_id": correlation.ID,
	}

	// Correlation summary metrics
	p.setGauge("correlation_risk_score", correlation.Summary.RiskScore, baseLabels, "Risk score of correlation analysis", timestamp)
	p.setGauge("correlation_total", float64(len(correlation.Correlations)), baseLabels, "Total number of correlations", timestamp)
	p.setGauge("correlation_high_confidence", float64(correlation.Summary.HighConfidence), baseLabels, "Number of high confidence correlations", timestamp)
	p.setGauge("correlation_medium_confidence", float64(correlation.Summary.MediumConfidence), baseLabels, "Number of medium confidence correlations", timestamp)
	p.setGauge("correlation_low_confidence", float64(correlation.Summary.LowConfidence), baseLabels, "Number of low confidence correlations", timestamp)
	p.setGauge("correlation_critical_findings", float64(correlation.Summary.CriticalFindings), baseLabels, "Number of critical findings", timestamp)

	// Individual correlations
	for _, corr := range correlation.Correlations {
		corrLabels := copyLabels(baseLabels)
		corrLabels["correlation_type"] = corr.Type
		corrLabels["source"] = corr.Source
		corrLabels["target"] = corr.Target

		p.setGauge("correlation_confidence", corr.Confidence, corrLabels, "Confidence level of correlation", timestamp)
		p.setGauge("correlation_strength", corr.Strength, corrLabels, "Strength of correlation", timestamp)
		p.incrementCounter("correlations_total", corrLabels, "Total number of correlations by type", timestamp)
	}

	data, err := p.generatePrometheusExposition()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Prometheus exposition: %w", err)
	}

	if enablePush, ok := p.config.Settings["enable_push"].(bool); ok && enablePush {
		if err := p.pushToPushGateway(ctx, data); err != nil {
			fmt.Printf("Warning: Failed to push to Prometheus push gateway: %v\n", err)
		}
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        data,
		ContentType: "text/plain; version=0.0.4",
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"metrics_count":      len(p.gauges) + len(p.counters) + len(p.histograms),
			"correlation_id":     correlation.ID,
			"correlations_count": len(correlation.Correlations),
			"exposition_size":    len(data),
		},
	}, nil
}

func (p *PrometheusExportPlugin) exportMetrics(ctx context.Context, data interface{}, options exports.ExportOptions) (*exports.ExportResponse, error) {
	// Convert arbitrary metrics data to Prometheus format
	timestamp := time.Now()
	baseLabels := map[string]string{
		"source": "custom",
	}

	// Try to convert data to map for processing
	if metricsMap, ok := data.(map[string]interface{}); ok {
		for key, value := range metricsMap {
			metricName := p.sanitizeMetricName(key)
			if floatVal, ok := p.convertToFloat64(value); ok {
				p.setGauge(fmt.Sprintf("custom_%s", metricName), floatVal, baseLabels, fmt.Sprintf("Custom metric: %s", key), timestamp)
			}
		}
	}

	exposition, err := p.generatePrometheusExposition()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Prometheus exposition: %w", err)
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        exposition,
		ContentType: "text/plain; version=0.0.4",
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"metrics_count":   len(p.gauges) + len(p.counters) + len(p.histograms),
			"exposition_size": len(exposition),
		},
	}, nil
}

// Metric management methods

func (p *PrometheusExportPlugin) setGauge(name string, value float64, labels map[string]string, help string, timestamp time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metricName := p.getFullMetricName(name)
	key := p.getMetricKey(metricName, labels)

	p.gauges[key] = &PrometheusGauge{
		Name:      metricName,
		Help:      help,
		Value:     value,
		Labels:    labels,
		Timestamp: timestamp,
	}
}

func (p *PrometheusExportPlugin) incrementCounter(name string, labels map[string]string, help string, timestamp time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metricName := p.getFullMetricName(name)
	key := p.getMetricKey(metricName, labels)

	if counter, exists := p.counters[key]; exists {
		counter.Value++
		counter.Timestamp = timestamp
	} else {
		p.counters[key] = &PrometheusCounter{
			Name:      metricName,
			Help:      help,
			Value:     1,
			Labels:    labels,
			Timestamp: timestamp,
		}
	}
}

func (p *PrometheusExportPlugin) getFullMetricName(name string) string {
	prefix := "vaino_"
	if metricPrefix, ok := p.config.Settings["metric_prefix"].(string); ok {
		prefix = metricPrefix
	}

	if p.namespace != "" {
		return fmt.Sprintf("%s%s_%s", prefix, p.namespace, name)
	}
	return fmt.Sprintf("%s%s", prefix, name)
}

func (p *PrometheusExportPlugin) getMetricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	var parts []string
	parts = append(parts, name)

	// Sort labels for consistent keys
	var labelPairs []string
	for k, v := range labels {
		labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(labelPairs)

	parts = append(parts, strings.Join(labelPairs, ","))
	return strings.Join(parts, "{")
}

func (p *PrometheusExportPlugin) generatePrometheusExposition() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result strings.Builder

	// Generate gauge metrics
	processedGauges := make(map[string]bool)
	for _, gauge := range p.gauges {
		if !processedGauges[gauge.Name] {
			result.WriteString(fmt.Sprintf("# HELP %s %s\n", gauge.Name, gauge.Help))
			result.WriteString(fmt.Sprintf("# TYPE %s gauge\n", gauge.Name))
			processedGauges[gauge.Name] = true
		}

		if len(gauge.Labels) > 0 {
			result.WriteString(fmt.Sprintf("%s%s %v %d\n", gauge.Name, p.formatLabels(gauge.Labels), gauge.Value, gauge.Timestamp.UnixMilli()))
		} else {
			result.WriteString(fmt.Sprintf("%s %v %d\n", gauge.Name, gauge.Value, gauge.Timestamp.UnixMilli()))
		}
	}

	// Generate counter metrics
	processedCounters := make(map[string]bool)
	for _, counter := range p.counters {
		if !processedCounters[counter.Name] {
			result.WriteString(fmt.Sprintf("# HELP %s %s\n", counter.Name, counter.Help))
			result.WriteString(fmt.Sprintf("# TYPE %s counter\n", counter.Name))
			processedCounters[counter.Name] = true
		}

		if len(counter.Labels) > 0 {
			result.WriteString(fmt.Sprintf("%s%s %v %d\n", counter.Name, p.formatLabels(counter.Labels), counter.Value, counter.Timestamp.UnixMilli()))
		} else {
			result.WriteString(fmt.Sprintf("%s %v %d\n", counter.Name, counter.Value, counter.Timestamp.UnixMilli()))
		}
	}

	// Generate histogram metrics (if any)
	processedHistograms := make(map[string]bool)
	for _, histogram := range p.histograms {
		if !processedHistograms[histogram.Name] {
			result.WriteString(fmt.Sprintf("# HELP %s %s\n", histogram.Name, histogram.Help))
			result.WriteString(fmt.Sprintf("# TYPE %s histogram\n", histogram.Name))
			processedHistograms[histogram.Name] = true
		}

		baseLabels := copyLabels(histogram.Labels)

		// Bucket metrics
		for le, count := range histogram.Buckets {
			bucketLabels := copyLabels(baseLabels)
			bucketLabels["le"] = le
			result.WriteString(fmt.Sprintf("%s_bucket%s %v %d\n", histogram.Name, p.formatLabels(bucketLabels), count, histogram.Timestamp.UnixMilli()))
		}

		// Count and sum
		result.WriteString(fmt.Sprintf("%s_count%s %v %d\n", histogram.Name, p.formatLabels(baseLabels), histogram.Count, histogram.Timestamp.UnixMilli()))
		result.WriteString(fmt.Sprintf("%s_sum%s %v %d\n", histogram.Name, p.formatLabels(baseLabels), histogram.Sum, histogram.Timestamp.UnixMilli()))
	}

	return []byte(result.String()), nil
}

func (p *PrometheusExportPlugin) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var pairs []string
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, k, p.escapeLabel(v)))
	}
	sort.Strings(pairs)

	return fmt.Sprintf("{%s}", strings.Join(pairs, ","))
}

func (p *PrometheusExportPlugin) escapeLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}

func (p *PrometheusExportPlugin) sanitizeMetricName(name string) string {
	// Replace invalid characters with underscores
	result := strings.ReplaceAll(name, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.ReplaceAll(result, " ", "_")
	return strings.ToLower(result)
}

func (p *PrometheusExportPlugin) convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

func (p *PrometheusExportPlugin) pushToPushGateway(ctx context.Context, data []byte) error {
	// In a real implementation, this would make an HTTP request to the Prometheus Push Gateway
	// For now, just simulate the push
	fmt.Printf("Prometheus: Pushing %d bytes to %s/metrics/job/%s/instance/%s\n",
		len(data), p.pushGateway, p.jobName, p.instance)
	return nil
}

func copyLabels(labels map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range labels {
		result[k] = v
	}
	return result
}

func (p *PrometheusExportPlugin) updateMetrics(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.TotalRequests++
	p.metrics.SuccessfulExports++
	p.metrics.LastExport = time.Now()

	if p.metrics.AverageLatency == 0 {
		p.metrics.AverageLatency = duration
	} else {
		p.metrics.AverageLatency = (p.metrics.AverageLatency + duration) / 2
	}

	p.metrics.MetricsUpdatedAt = time.Now()
}

// Plugin validation and health methods

func (p *PrometheusExportPlugin) Validate(config exports.PluginConfig) error {
	// Validate push gateway URL
	if pushGateway, ok := config.Settings["push_gateway"].(string); ok {
		if pushGateway == "" {
			return fmt.Errorf("push_gateway cannot be empty")
		}
	}

	// Validate timeout
	if timeoutStr, ok := config.Settings["timeout"].(string); ok {
		if _, err := time.ParseDuration(timeoutStr); err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
	}

	// Validate job name
	if jobName, ok := config.Settings["job_name"].(string); ok {
		if jobName == "" {
			return fmt.Errorf("job_name cannot be empty")
		}
	}

	return nil
}

func (p *PrometheusExportPlugin) HealthCheck(ctx context.Context) exports.HealthStatus {
	status := exports.HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
		Message:   "Prometheus plugin operational",
		Uptime:    time.Since(p.startTime),
		Version:   p.Version(),
		Details:   make(map[string]interface{}),
	}

	// Check metric storage health
	p.mu.RLock()
	gaugeCount := len(p.gauges)
	counterCount := len(p.counters)
	histogramCount := len(p.histograms)
	p.mu.RUnlock()

	if gaugeCount+counterCount+histogramCount > 10000 {
		status.Status = "degraded"
		status.Message = "High number of stored metrics may impact performance"
	}

	status.Details["push_gateway"] = p.pushGateway
	status.Details["job_name"] = p.jobName
	status.Details["instance"] = p.instance
	status.Details["namespace"] = p.namespace
	status.Details["running"] = p.running
	status.Details["gauge_count"] = gaugeCount
	status.Details["counter_count"] = counterCount
	status.Details["histogram_count"] = histogramCount

	return status
}

func (p *PrometheusExportPlugin) GetMetrics() exports.PluginMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// Configuration methods

func (p *PrometheusExportPlugin) Schema() exports.PluginSchema {
	return exports.PluginSchema{
		Name:        "prometheus",
		Version:     "1.0.0",
		Description: "Prometheus metrics export plugin for infrastructure monitoring",
		Schema: map[string]exports.SchemaField{
			"push_gateway": {
				Type:        "string",
				Description: "Prometheus Push Gateway URL",
				Default:     "http://localhost:9091",
				Required:    true,
				Format:      "url",
			},
			"job_name": {
				Type:        "string",
				Description: "Job name for Prometheus metrics",
				Default:     "vaino",
				Required:    true,
			},
			"instance": {
				Type:        "string",
				Description: "Instance identifier for metrics",
				Default:     "localhost:8080",
				Required:    true,
			},
			"namespace": {
				Type:        "string",
				Description: "Namespace for metric names",
				Default:     "vaino",
				Required:    false,
			},
			"timeout": {
				Type:        "string",
				Description: "HTTP timeout for push gateway requests",
				Default:     "30s",
				Required:    false,
				Pattern:     `^\d+[smh]$`,
			},
			"enable_push": {
				Type:        "bool",
				Description: "Enable pushing metrics to push gateway",
				Default:     true,
				Required:    false,
			},
			"metric_prefix": {
				Type:        "string",
				Description: "Prefix for all metric names",
				Default:     "vaino_",
				Required:    false,
			},
		},
		Required: []string{"push_gateway", "job_name", "instance"},
		Examples: []map[string]interface{}{
			{
				"push_gateway":  "https://prometheus-pushgateway.example.com:9091",
				"job_name":      "vaino-production",
				"instance":      "vaino-server-01:8080",
				"namespace":     "infrastructure",
				"timeout":       "60s",
				"enable_push":   true,
				"metric_prefix": "infra_",
			},
		},
	}
}

func (p *PrometheusExportPlugin) GetConfig() exports.PluginConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

func (p *PrometheusExportPlugin) UpdateConfig(ctx context.Context, config exports.PluginConfig) error {
	if err := p.Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config

	// Update internal settings
	if pushGateway, ok := config.Settings["push_gateway"].(string); ok {
		p.pushGateway = pushGateway
	}

	if jobName, ok := config.Settings["job_name"].(string); ok {
		p.jobName = jobName
	}

	if instance, ok := config.Settings["instance"].(string); ok {
		p.instance = instance
	}

	if namespace, ok := config.Settings["namespace"].(string); ok {
		p.namespace = namespace
	}

	if timeoutStr, ok := config.Settings["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			p.httpTimeout = timeout
		}
	}

	return nil
}
