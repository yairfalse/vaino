package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/exports"
	"github.com/yairfalse/vaino/pkg/types"
)

// OTELExportPlugin implements ExportPlugin for OpenTelemetry export
type OTELExportPlugin struct {
	mu              sync.RWMutex
	config          exports.PluginConfig
	running         bool
	startTime       time.Time
	metrics         exports.PluginMetrics
	endpoint        string
	insecure        bool
	timeout         time.Duration
	batchSize       int
	batchTimeout    time.Duration
	headers         map[string]string
	resourceAttribs map[string]interface{}

	// Internal components
	traceExporter   *TraceExporter
	metricsExporter *MetricsExporter
	logExporter     *LogExporter
	batchProcessor  *BatchProcessor

	// Channels for data processing
	traceChan   chan *TraceData
	metricsChan chan *MetricsData
	logChan     chan *LogData
	shutdown    chan struct{}
}

// TraceExporter handles OTEL trace export
type TraceExporter struct {
	endpoint  string
	headers   map[string]string
	client    *OTELClient
	batchSize int
	timeout   time.Duration
}

// MetricsExporter handles OTEL metrics export
type MetricsExporter struct {
	endpoint  string
	headers   map[string]string
	client    *OTELClient
	batchSize int
	timeout   time.Duration
}

// LogExporter handles OTEL log export
type LogExporter struct {
	endpoint  string
	headers   map[string]string
	client    *OTELClient
	batchSize int
	timeout   time.Duration
}

// BatchProcessor batches data for efficient export
type BatchProcessor struct {
	traces    []*TraceData
	metrics   []*MetricsData
	logs      []*LogData
	batchSize int
	timeout   time.Duration
	lastFlush time.Time
	mu        sync.Mutex
}

// OTELClient handles HTTP communication with OTEL collector
type OTELClient struct {
	endpoint string
	headers  map[string]string
	timeout  time.Duration
	insecure bool
}

// TraceData represents trace data for OTEL export
type TraceData struct {
	TraceID      string                 `json:"traceId"`
	SpanID       string                 `json:"spanId"`
	ParentSpanID string                 `json:"parentSpanId,omitempty"`
	Name         string                 `json:"name"`
	Kind         string                 `json:"kind"`
	StartTime    time.Time              `json:"startTime"`
	EndTime      time.Time              `json:"endTime"`
	Status       TraceStatus            `json:"status"`
	Attributes   map[string]interface{} `json:"attributes"`
	Events       []TraceEvent           `json:"events,omitempty"`
	Links        []TraceLink            `json:"links,omitempty"`
	Resource     Resource               `json:"resource"`
}

// MetricsData represents metrics data for OTEL export
type MetricsData struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Unit        string                 `json:"unit"`
	Type        string                 `json:"type"` // counter, gauge, histogram
	Value       interface{}            `json:"value"`
	Timestamp   time.Time              `json:"timestamp"`
	Attributes  map[string]interface{} `json:"attributes"`
	Resource    Resource               `json:"resource"`
}

// LogData represents log data for OTEL export
type LogData struct {
	Timestamp         time.Time              `json:"timestamp"`
	SeverityText      string                 `json:"severityText"`
	SeverityNumber    int                    `json:"severityNumber"`
	Body              string                 `json:"body"`
	Attributes        map[string]interface{} `json:"attributes"`
	Resource          Resource               `json:"resource"`
	TraceID           string                 `json:"traceId,omitempty"`
	SpanID            string                 `json:"spanId,omitempty"`
	ObservedTimestamp time.Time              `json:"observedTimestamp"`
}

// TraceStatus represents the status of a trace span
type TraceStatus struct {
	Code    string `json:"code"` // OK, ERROR, TIMEOUT
	Message string `json:"message,omitempty"`
}

// TraceEvent represents an event within a trace span
type TraceEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// TraceLink represents a link to another trace
type TraceLink struct {
	TraceID    string                 `json:"traceId"`
	SpanID     string                 `json:"spanId"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Resource represents OTEL resource information
type Resource struct {
	Attributes map[string]interface{} `json:"attributes"`
}

// NewOTELExportPlugin creates a new OTEL export plugin
func NewOTELExportPlugin() *OTELExportPlugin {
	return &OTELExportPlugin{
		config: exports.PluginConfig{
			Name:    "otel",
			Version: "1.0.0",
			Enabled: true,
			Settings: map[string]interface{}{
				"endpoint":       "http://localhost:4317",
				"insecure":       true,
				"timeout":        "30s",
				"batch_size":     100,
				"batch_timeout":  "10s",
				"enable_traces":  true,
				"enable_metrics": true,
				"enable_logs":    true,
			},
			Endpoints: map[string]string{
				"traces":  "/v1/traces",
				"metrics": "/v1/metrics",
				"logs":    "/v1/logs",
			},
		},
		endpoint:        "http://localhost:4317",
		insecure:        true,
		timeout:         30 * time.Second,
		batchSize:       100,
		batchTimeout:    10 * time.Second,
		headers:         make(map[string]string),
		resourceAttribs: make(map[string]interface{}),
		traceChan:       make(chan *TraceData, 1000),
		metricsChan:     make(chan *MetricsData, 1000),
		logChan:         make(chan *LogData, 1000),
		shutdown:        make(chan struct{}),
		startTime:       time.Now(),
	}
}

// Plugin metadata methods

func (p *OTELExportPlugin) Name() string {
	return "otel"
}

func (p *OTELExportPlugin) Version() string {
	return "1.0.0"
}

func (p *OTELExportPlugin) Description() string {
	return "OpenTelemetry export plugin for traces, metrics, and logs"
}

func (p *OTELExportPlugin) SupportedFormats() []exports.ExportFormat {
	return []exports.ExportFormat{
		exports.FormatOTEL,
	}
}

// Plugin lifecycle methods

func (p *OTELExportPlugin) Initialize(ctx context.Context, config exports.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if config.Name != "" {
		p.config = config
	}

	// Extract settings
	if endpoint, ok := p.config.Settings["endpoint"].(string); ok {
		p.endpoint = endpoint
	}

	if insecure, ok := p.config.Settings["insecure"].(bool); ok {
		p.insecure = insecure
	}

	if timeoutStr, ok := p.config.Settings["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			p.timeout = timeout
		}
	}

	if batchSize, ok := p.config.Settings["batch_size"].(int); ok {
		p.batchSize = batchSize
	}

	if batchTimeoutStr, ok := p.config.Settings["batch_timeout"].(string); ok {
		if batchTimeout, err := time.ParseDuration(batchTimeoutStr); err == nil {
			p.batchTimeout = batchTimeout
		}
	}

	// Initialize headers
	p.headers["Content-Type"] = "application/x-protobuf"
	p.headers["User-Agent"] = "vaino-otel-exporter/1.0.0"

	// Add any custom headers from settings
	if headers, ok := p.config.Settings["headers"].(map[string]interface{}); ok {
		for hKey, hValue := range headers {
			if headerStr, ok := hValue.(string); ok {
				p.headers[hKey] = headerStr
			}
		}
	}

	// Initialize resource attributes
	p.resourceAttribs["service.name"] = "vaino"
	p.resourceAttribs["service.version"] = "1.0.0"
	p.resourceAttribs["deployment.environment"] = "production"

	// Initialize exporters
	p.traceExporter = &TraceExporter{
		endpoint:  p.endpoint + p.config.Endpoints["traces"],
		headers:   p.headers,
		batchSize: p.batchSize,
		timeout:   p.timeout,
	}

	p.metricsExporter = &MetricsExporter{
		endpoint:  p.endpoint + p.config.Endpoints["metrics"],
		headers:   p.headers,
		batchSize: p.batchSize,
		timeout:   p.timeout,
	}

	p.logExporter = &LogExporter{
		endpoint:  p.endpoint + p.config.Endpoints["logs"],
		headers:   p.headers,
		batchSize: p.batchSize,
		timeout:   p.timeout,
	}

	// Initialize batch processor
	p.batchProcessor = &BatchProcessor{
		traces:    make([]*TraceData, 0, p.batchSize),
		metrics:   make([]*MetricsData, 0, p.batchSize),
		logs:      make([]*LogData, 0, p.batchSize),
		batchSize: p.batchSize,
		timeout:   p.batchTimeout,
		lastFlush: time.Now(),
	}

	// Initialize OTEL client
	client := &OTELClient{
		endpoint: p.endpoint,
		headers:  p.headers,
		timeout:  p.timeout,
		insecure: p.insecure,
	}

	p.traceExporter.client = client
	p.metricsExporter.client = client
	p.logExporter.client = client

	return nil
}

func (p *OTELExportPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("OTEL plugin already running")
	}

	p.running = true
	p.startTime = time.Now()

	// Start background processors
	go p.processTraces(ctx)
	go p.processMetrics(ctx)
	go p.processLogs(ctx)
	go p.batchFlushLoop(ctx)

	return nil
}

func (p *OTELExportPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	// Signal shutdown
	close(p.shutdown)

	// Flush remaining data
	p.batchProcessor.flush(ctx, p)

	p.running = false
	return nil
}

func (p *OTELExportPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Export methods

func (p *OTELExportPlugin) Export(ctx context.Context, request *exports.ExportRequest) (*exports.ExportResponse, error) {
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

func (p *OTELExportPlugin) ExportDriftReport(ctx context.Context, report *differ.DriftReport, options exports.ExportOptions) error {
	_, err := p.exportDriftReport(ctx, report, options)
	return err
}

func (p *OTELExportPlugin) ExportSnapshot(ctx context.Context, snapshot *types.Snapshot, options exports.ExportOptions) error {
	_, err := p.exportSnapshot(ctx, snapshot, options)
	return err
}

func (p *OTELExportPlugin) ExportCorrelation(ctx context.Context, correlation *exports.CorrelationData, options exports.ExportOptions) error {
	_, err := p.exportCorrelation(ctx, correlation, options)
	return err
}

// Internal export methods

func (p *OTELExportPlugin) exportDriftReport(ctx context.Context, report *differ.DriftReport, options exports.ExportOptions) (*exports.ExportResponse, error) {
	// Create trace for drift detection process
	traceData := &TraceData{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Name:      "drift_detection",
		Kind:      "internal",
		StartTime: report.Timestamp,
		EndTime:   time.Now(),
		Status: TraceStatus{
			Code: "OK",
		},
		Attributes: map[string]interface{}{
			"vaino.drift.report_id":         report.ID,
			"vaino.drift.baseline_id":       report.BaselineID,
			"vaino.drift.current_id":        report.CurrentID,
			"vaino.drift.changes_count":     len(report.ResourceChanges),
			"vaino.drift.overall_risk":      string(report.Summary.OverallRisk),
			"vaino.drift.risk_score":        report.Summary.RiskScore,
			"vaino.drift.total_resources":   report.Summary.TotalResources,
			"vaino.drift.changed_resources": report.Summary.ChangedResources,
		},
		Resource: Resource{
			Attributes: p.resourceAttribs,
		},
	}

	// Add events for each significant change
	for _, change := range report.ResourceChanges {
		if change.Severity == differ.RiskLevelCritical || change.Severity == differ.RiskLevelHigh {
			event := TraceEvent{
				Name:      "resource_change_detected",
				Timestamp: time.Now(),
				Attributes: map[string]interface{}{
					"vaino.resource.id":       change.ResourceID,
					"vaino.resource.type":     change.ResourceType,
					"vaino.change.type":       string(change.DriftType),
					"vaino.change.severity":   string(change.Severity),
					"vaino.change.category":   string(change.Category),
					"vaino.change.risk_score": change.RiskScore,
				},
			}
			traceData.Events = append(traceData.Events, event)
		}
	}

	// Send trace data
	select {
	case p.traceChan <- traceData:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create metrics for drift report
	timestamp := time.Now()
	metricsData := []*MetricsData{
		{
			Name:        "vaino_drift_changes_total",
			Description: "Total number of drift changes detected",
			Unit:        "1",
			Type:        "counter",
			Value:       int64(len(report.ResourceChanges)),
			Timestamp:   timestamp,
			Attributes: map[string]interface{}{
				"baseline_id": report.BaselineID,
				"report_id":   report.ID,
			},
			Resource: Resource{Attributes: p.resourceAttribs},
		},
		{
			Name:        "vaino_drift_risk_score",
			Description: "Risk score of the drift report",
			Unit:        "1",
			Type:        "gauge",
			Value:       report.Summary.RiskScore,
			Timestamp:   timestamp,
			Attributes: map[string]interface{}{
				"baseline_id": report.BaselineID,
				"report_id":   report.ID,
				"risk_level":  string(report.Summary.OverallRisk),
			},
			Resource: Resource{Attributes: p.resourceAttribs},
		},
	}

	// Send metrics data
	for _, metric := range metricsData {
		select {
		case p.metricsChan <- metric:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Create log entries for critical changes
	for _, change := range report.ResourceChanges {
		if change.Severity == differ.RiskLevelCritical {
			logData := &LogData{
				Timestamp:      time.Now(),
				SeverityText:   "ERROR",
				SeverityNumber: 17, // OTEL ERROR level
				Body:           fmt.Sprintf("Critical drift detected: %s", change.Description),
				Attributes: map[string]interface{}{
					"vaino.resource.id":     change.ResourceID,
					"vaino.resource.type":   change.ResourceType,
					"vaino.change.type":     string(change.DriftType),
					"vaino.change.severity": string(change.Severity),
					"vaino.drift.report_id": report.ID,
				},
				Resource:          Resource{Attributes: p.resourceAttribs},
				TraceID:           traceData.TraceID,
				SpanID:            traceData.SpanID,
				ObservedTimestamp: time.Now(),
			}

			select {
			case p.logChan <- logData:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"trace_id":      traceData.TraceID,
			"metrics_count": len(metricsData),
			"events_count":  len(traceData.Events),
		},
	}, nil
}

func (p *OTELExportPlugin) exportSnapshot(ctx context.Context, snapshot *types.Snapshot, options exports.ExportOptions) (*exports.ExportResponse, error) {
	// Create metrics for snapshot
	timestamp := time.Now()
	metricsData := []*MetricsData{
		{
			Name:        "vaino_snapshot_resources_total",
			Description: "Total number of resources in snapshot",
			Unit:        "1",
			Type:        "gauge",
			Value:       int64(len(snapshot.Resources)),
			Timestamp:   timestamp,
			Attributes: map[string]interface{}{
				"snapshot_id": snapshot.ID,
				"provider":    snapshot.Provider,
			},
			Resource: Resource{Attributes: p.resourceAttribs},
		},
	}

	// Count resources by type
	resourceCounts := make(map[string]int)
	for _, resource := range snapshot.Resources {
		resourceCounts[resource.Type]++
	}

	for resourceType, count := range resourceCounts {
		metricsData = append(metricsData, &MetricsData{
			Name:        "vaino_snapshot_resources_by_type",
			Description: "Number of resources by type in snapshot",
			Unit:        "1",
			Type:        "gauge",
			Value:       int64(count),
			Timestamp:   timestamp,
			Attributes: map[string]interface{}{
				"snapshot_id":   snapshot.ID,
				"provider":      snapshot.Provider,
				"resource_type": resourceType,
			},
			Resource: Resource{Attributes: p.resourceAttribs},
		})
	}

	// Send metrics
	for _, metric := range metricsData {
		select {
		case p.metricsChan <- metric:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"metrics_count": len(metricsData),
			"snapshot_id":   snapshot.ID,
		},
	}, nil
}

func (p *OTELExportPlugin) exportCorrelation(ctx context.Context, correlation *exports.CorrelationData, options exports.ExportOptions) (*exports.ExportResponse, error) {
	// Create trace for correlation analysis
	traceData := &TraceData{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Name:      "correlation_analysis",
		Kind:      "internal",
		StartTime: correlation.Timestamp.Add(-1 * time.Minute), // Estimate start time
		EndTime:   correlation.Timestamp,
		Status: TraceStatus{
			Code: "OK",
		},
		Attributes: map[string]interface{}{
			"vaino.correlation.id":                correlation.ID,
			"vaino.correlation.total":             len(correlation.Correlations),
			"vaino.correlation.high_confidence":   correlation.Summary.HighConfidence,
			"vaino.correlation.critical_findings": correlation.Summary.CriticalFindings,
			"vaino.correlation.risk_score":        correlation.Summary.RiskScore,
		},
		Resource: Resource{Attributes: p.resourceAttribs},
	}

	// Add events for high-confidence correlations
	for _, corr := range correlation.Correlations {
		if corr.Confidence > 0.8 {
			event := TraceEvent{
				Name:      "high_confidence_correlation",
				Timestamp: time.Now(),
				Attributes: map[string]interface{}{
					"vaino.correlation.type":       corr.Type,
					"vaino.correlation.source":     corr.Source,
					"vaino.correlation.target":     corr.Target,
					"vaino.correlation.confidence": corr.Confidence,
					"vaino.correlation.strength":   corr.Strength,
				},
			}
			traceData.Events = append(traceData.Events, event)
		}
	}

	// Send trace
	select {
	case p.traceChan <- traceData:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"trace_id":       traceData.TraceID,
			"correlation_id": correlation.ID,
			"events_count":   len(traceData.Events),
		},
	}, nil
}

func (p *OTELExportPlugin) exportMetrics(ctx context.Context, data interface{}, options exports.ExportOptions) (*exports.ExportResponse, error) {
	// Convert arbitrary metrics data to OTEL format
	metricsJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics data: %w", err)
	}

	var metricsMap map[string]interface{}
	if err := json.Unmarshal(metricsJSON, &metricsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics data: %w", err)
	}

	timestamp := time.Now()
	var sentCount int

	// Convert map entries to OTEL metrics
	for key, value := range metricsMap {
		metric := &MetricsData{
			Name:        fmt.Sprintf("vaino_custom_%s", key),
			Description: fmt.Sprintf("Custom metric: %s", key),
			Unit:        "1",
			Type:        "gauge",
			Value:       value,
			Timestamp:   timestamp,
			Attributes: map[string]interface{}{
				"metric_source": "custom",
			},
			Resource: Resource{Attributes: p.resourceAttribs},
		}

		select {
		case p.metricsChan <- metric:
			sentCount++
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		PluginName:  p.Name(),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"metrics_sent": sentCount,
		},
	}, nil
}

// Background processing methods

func (p *OTELExportPlugin) processTraces(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdown:
			return
		case trace := <-p.traceChan:
			p.batchProcessor.addTrace(trace)
		}
	}
}

func (p *OTELExportPlugin) processMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdown:
			return
		case metric := <-p.metricsChan:
			p.batchProcessor.addMetric(metric)
		}
	}
}

func (p *OTELExportPlugin) processLogs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdown:
			return
		case log := <-p.logChan:
			p.batchProcessor.addLog(log)
		}
	}
}

func (p *OTELExportPlugin) batchFlushLoop(ctx context.Context) {
	ticker := time.NewTicker(p.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdown:
			return
		case <-ticker.C:
			p.batchProcessor.flush(ctx, p)
		}
	}
}

// Batch processor methods

func (bp *BatchProcessor) addTrace(trace *TraceData) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.traces = append(bp.traces, trace)
	if len(bp.traces) >= bp.batchSize {
		// Trigger immediate flush for full batch
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), bp.timeout)
			defer cancel()
			bp.flushTraces(ctx, nil) // Would pass actual plugin in production
		}()
	}
}

func (bp *BatchProcessor) addMetric(metric *MetricsData) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.metrics = append(bp.metrics, metric)
	if len(bp.metrics) >= bp.batchSize {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), bp.timeout)
			defer cancel()
			bp.flushMetrics(ctx, nil) // Would pass actual plugin in production
		}()
	}
}

func (bp *BatchProcessor) addLog(log *LogData) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.logs = append(bp.logs, log)
	if len(bp.logs) >= bp.batchSize {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), bp.timeout)
			defer cancel()
			bp.flushLogs(ctx, nil) // Would pass actual plugin in production
		}()
	}
}

func (bp *BatchProcessor) flush(ctx context.Context, plugin *OTELExportPlugin) {
	bp.flushTraces(ctx, plugin)
	bp.flushMetrics(ctx, plugin)
	bp.flushLogs(ctx, plugin)
}

func (bp *BatchProcessor) flushTraces(ctx context.Context, plugin *OTELExportPlugin) {
	bp.mu.Lock()
	if len(bp.traces) == 0 {
		bp.mu.Unlock()
		return
	}
	traces := make([]*TraceData, len(bp.traces))
	copy(traces, bp.traces)
	bp.traces = bp.traces[:0]
	bp.mu.Unlock()

	// In a real implementation, this would send to OTEL collector
	// For now, just log the export
	fmt.Printf("OTEL: Exporting %d traces\n", len(traces))
}

func (bp *BatchProcessor) flushMetrics(ctx context.Context, plugin *OTELExportPlugin) {
	bp.mu.Lock()
	if len(bp.metrics) == 0 {
		bp.mu.Unlock()
		return
	}
	metrics := make([]*MetricsData, len(bp.metrics))
	copy(metrics, bp.metrics)
	bp.metrics = bp.metrics[:0]
	bp.mu.Unlock()

	// In a real implementation, this would send to OTEL collector
	fmt.Printf("OTEL: Exporting %d metrics\n", len(metrics))
}

func (bp *BatchProcessor) flushLogs(ctx context.Context, plugin *OTELExportPlugin) {
	bp.mu.Lock()
	if len(bp.logs) == 0 {
		bp.mu.Unlock()
		return
	}
	logs := make([]*LogData, len(bp.logs))
	copy(logs, bp.logs)
	bp.logs = bp.logs[:0]
	bp.mu.Unlock()

	// In a real implementation, this would send to OTEL collector
	fmt.Printf("OTEL: Exporting %d logs\n", len(logs))
}

// Utility methods

func (p *OTELExportPlugin) updateMetrics(duration time.Duration) {
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

func generateTraceID() string {
	// Simplified trace ID generation
	return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), time.Now().UnixNano())
}

func generateSpanID() string {
	// Simplified span ID generation
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

// Plugin validation and health methods

func (p *OTELExportPlugin) Validate(config exports.PluginConfig) error {
	// Validate endpoint
	if endpoint, ok := config.Settings["endpoint"].(string); ok {
		if endpoint == "" {
			return fmt.Errorf("endpoint cannot be empty")
		}
	}

	// Validate timeout
	if timeoutStr, ok := config.Settings["timeout"].(string); ok {
		if _, err := time.ParseDuration(timeoutStr); err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
	}

	// Validate batch settings
	if batchSize, ok := config.Settings["batch_size"].(int); ok {
		if batchSize < 1 || batchSize > 10000 {
			return fmt.Errorf("batch_size must be between 1 and 10000")
		}
	}

	return nil
}

func (p *OTELExportPlugin) HealthCheck(ctx context.Context) exports.HealthStatus {
	status := exports.HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
		Message:   "OTEL plugin operational",
		Uptime:    time.Since(p.startTime),
		Version:   p.Version(),
		Details:   make(map[string]interface{}),
	}

	// Check channel health
	if len(p.traceChan) > cap(p.traceChan)*8/10 {
		status.Status = "degraded"
		status.Message = "Trace channel near capacity"
	}

	if len(p.metricsChan) > cap(p.metricsChan)*8/10 {
		status.Status = "degraded"
		status.Message = "Metrics channel near capacity"
	}

	if len(p.logChan) > cap(p.logChan)*8/10 {
		status.Status = "degraded"
		status.Message = "Log channel near capacity"
	}

	status.Details["endpoint"] = p.endpoint
	status.Details["running"] = p.running
	status.Details["trace_queue_size"] = len(p.traceChan)
	status.Details["metrics_queue_size"] = len(p.metricsChan)
	status.Details["log_queue_size"] = len(p.logChan)

	return status
}

func (p *OTELExportPlugin) GetMetrics() exports.PluginMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// Configuration methods

func (p *OTELExportPlugin) Schema() exports.PluginSchema {
	return exports.PluginSchema{
		Name:        "otel",
		Version:     "1.0.0",
		Description: "OpenTelemetry export plugin for traces, metrics, and logs",
		Schema: map[string]exports.SchemaField{
			"endpoint": {
				Type:        "string",
				Description: "OTEL collector endpoint URL",
				Default:     "http://localhost:4317",
				Required:    true,
				Format:      "url",
			},
			"insecure": {
				Type:        "bool",
				Description: "Use insecure connection",
				Default:     true,
				Required:    false,
			},
			"timeout": {
				Type:        "string",
				Description: "Request timeout duration",
				Default:     "30s",
				Required:    false,
				Pattern:     `^\d+[smh]$`,
			},
			"batch_size": {
				Type:        "int",
				Description: "Batch size for data export",
				Default:     100,
				Minimum:     &[]float64{1}[0],
				Maximum:     &[]float64{10000}[0],
				Required:    false,
			},
			"batch_timeout": {
				Type:        "string",
				Description: "Batch timeout duration",
				Default:     "10s",
				Required:    false,
				Pattern:     `^\d+[smh]$`,
			},
			"enable_traces": {
				Type:        "bool",
				Description: "Enable trace export",
				Default:     true,
				Required:    false,
			},
			"enable_metrics": {
				Type:        "bool",
				Description: "Enable metrics export",
				Default:     true,
				Required:    false,
			},
			"enable_logs": {
				Type:        "bool",
				Description: "Enable log export",
				Default:     true,
				Required:    false,
			},
		},
		Required: []string{"endpoint"},
		Examples: []map[string]interface{}{
			{
				"endpoint":       "https://otel-collector.example.com:4317",
				"insecure":       false,
				"timeout":        "60s",
				"batch_size":     500,
				"batch_timeout":  "30s",
				"enable_traces":  true,
				"enable_metrics": true,
				"enable_logs":    false,
			},
		},
	}
}

func (p *OTELExportPlugin) GetConfig() exports.PluginConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

func (p *OTELExportPlugin) UpdateConfig(ctx context.Context, config exports.PluginConfig) error {
	if err := p.Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config

	// Update internal settings
	if endpoint, ok := config.Settings["endpoint"].(string); ok {
		p.endpoint = endpoint
	}

	if insecure, ok := config.Settings["insecure"].(bool); ok {
		p.insecure = insecure
	}

	if timeoutStr, ok := config.Settings["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			p.timeout = timeout
		}
	}

	if batchSize, ok := config.Settings["batch_size"].(int); ok {
		p.batchSize = batchSize
	}

	return nil
}
