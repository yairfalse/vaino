package exports

import (
	"context"
	"io"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// ExportPlugin defines the interface for server-side export plugins
type ExportPlugin interface {
	// Plugin metadata
	Name() string
	Version() string
	Description() string
	SupportedFormats() []ExportFormat

	// Plugin lifecycle
	Initialize(ctx context.Context, config PluginConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool

	// Export operations
	Export(ctx context.Context, request *ExportRequest) (*ExportResponse, error)
	ExportDriftReport(ctx context.Context, report *differ.DriftReport, options ExportOptions) error
	ExportSnapshot(ctx context.Context, snapshot *types.Snapshot, options ExportOptions) error
	ExportCorrelation(ctx context.Context, correlation *CorrelationData, options ExportOptions) error

	// Plugin validation and health
	Validate(config PluginConfig) error
	HealthCheck(ctx context.Context) HealthStatus
	GetMetrics() PluginMetrics

	// Configuration and schema
	Schema() PluginSchema
	GetConfig() PluginConfig
	UpdateConfig(ctx context.Context, config PluginConfig) error
}

// ExportFormat represents supported export formats
type ExportFormat string

const (
	FormatJSON       ExportFormat = "json"
	FormatYAML       ExportFormat = "yaml"
	FormatMarkdown   ExportFormat = "markdown"
	FormatHTML       ExportFormat = "html"
	FormatCSV        ExportFormat = "csv"
	FormatXML        ExportFormat = "xml"
	FormatPrometheus ExportFormat = "prometheus"
	FormatOTEL       ExportFormat = "otel"
	FormatWebhook    ExportFormat = "webhook"
	FormatCustom     ExportFormat = "custom"
)

// ExportRequest represents a request to export data
type ExportRequest struct {
	ID          string                 `json:"id"`
	PluginName  string                 `json:"plugin_name"`
	Format      ExportFormat           `json:"format"`
	DataType    ExportDataType         `json:"data_type"`
	Data        interface{}            `json:"data"`
	Options     ExportOptions          `json:"options"`
	Metadata    map[string]interface{} `json:"metadata"`
	RequestedAt time.Time              `json:"requested_at"`
	Priority    ExportPriority         `json:"priority"`
	Async       bool                   `json:"async"`
}

// ExportResponse represents the result of an export operation
type ExportResponse struct {
	ID          string                 `json:"id"`
	PluginName  string                 `json:"plugin_name"`
	Status      ExportStatus           `json:"status"`
	Data        []byte                 `json:"data,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	OutputPath  string                 `json:"output_path,omitempty"`
	ExternalURL string                 `json:"external_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	ProcessedAt time.Time              `json:"processed_at"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
	Metrics     ExportMetrics          `json:"metrics"`
}

// ExportDataType represents the type of data being exported
type ExportDataType string

const (
	DataTypeDriftReport ExportDataType = "drift_report"
	DataTypeSnapshot    ExportDataType = "snapshot"
	DataTypeCorrelation ExportDataType = "correlation"
	DataTypeBaseline    ExportDataType = "baseline"
	DataTypeTimeline    ExportDataType = "timeline"
	DataTypeMetrics     ExportDataType = "metrics"
	DataTypeEvents      ExportDataType = "events"
	DataTypeAlert       ExportDataType = "alert"
)

// ExportStatus represents the status of an export operation
type ExportStatus string

const (
	StatusPending    ExportStatus = "pending"
	StatusProcessing ExportStatus = "processing"
	StatusCompleted  ExportStatus = "completed"
	StatusFailed     ExportStatus = "failed"
	StatusCancelled  ExportStatus = "cancelled"
)

// ExportPriority represents the priority of an export request
type ExportPriority int

const (
	PriorityLow      ExportPriority = 1
	PriorityNormal   ExportPriority = 5
	PriorityHigh     ExportPriority = 10
	PriorityCritical ExportPriority = 15
)

// ExportOptions configures export behavior
type ExportOptions struct {
	// Output configuration
	Format      ExportFormat `json:"format"`
	OutputPath  string       `json:"output_path,omitempty"`
	Destination string       `json:"destination,omitempty"`
	Writer      io.Writer    `json:"-"`

	// Content options
	Compress     bool     `json:"compress"`
	Pretty       bool     `json:"pretty"`
	Template     string   `json:"template,omitempty"`
	CustomFields []string `json:"custom_fields,omitempty"`

	// Filtering and transformation
	FilterLevel   string            `json:"filter_level,omitempty"`
	FilterTags    map[string]string `json:"filter_tags,omitempty"`
	IncludeFields []string          `json:"include_fields,omitempty"`
	ExcludeFields []string          `json:"exclude_fields,omitempty"`

	// Delivery options
	Async   bool          `json:"async"`
	Retry   RetryPolicy   `json:"retry"`
	Timeout time.Duration `json:"timeout"`

	// Security and authentication
	AuthToken string            `json:"auth_token,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`

	// Plugin-specific options
	PluginOptions map[string]interface{} `json:"plugin_options,omitempty"`
}

// RetryPolicy defines retry behavior for failed exports
type RetryPolicy struct {
	Enabled      bool          `json:"enabled"`
	MaxRetries   int           `json:"max_retries"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Backoff      BackoffType   `json:"backoff"`
}

// BackoffType defines the backoff strategy for retries
type BackoffType string

const (
	BackoffLinear      BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
	BackoffFixed       BackoffType = "fixed"
)

// PluginConfig holds plugin-specific configuration
type PluginConfig struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Enabled     bool                   `json:"enabled"`
	Settings    map[string]interface{} `json:"settings"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	Endpoints   map[string]string      `json:"endpoints,omitempty"`
	RateLimit   RateLimitConfig        `json:"rate_limit"`
	Security    SecurityConfig         `json:"security"`
	Monitoring  MonitoringConfig       `json:"monitoring"`
}

// RateLimitConfig defines rate limiting for export operations
type RateLimitConfig struct {
	Enabled        bool          `json:"enabled"`
	RequestsPerSec int           `json:"requests_per_sec"`
	BurstLimit     int           `json:"burst_limit"`
	QueueSize      int           `json:"queue_size"`
	Timeout        time.Duration `json:"timeout"`
}

// SecurityConfig defines security settings for plugins
type SecurityConfig struct {
	TLSEnabled     bool                   `json:"tls_enabled"`
	TLSConfig      map[string]interface{} `json:"tls_config,omitempty"`
	Authentication AuthConfig             `json:"authentication"`
	Encryption     EncryptionConfig       `json:"encryption"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	Type        string            `json:"type"` // bearer, basic, oauth, apikey, mtls
	Credentials map[string]string `json:"credentials,omitempty"`
	TokenURL    string            `json:"token_url,omitempty"`
	Scopes      []string          `json:"scopes,omitempty"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	Enabled   bool   `json:"enabled"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id,omitempty"`
}

// MonitoringConfig defines monitoring and observability settings
type MonitoringConfig struct {
	Enabled         bool              `json:"enabled"`
	MetricsEnabled  bool              `json:"metrics_enabled"`
	TracingEnabled  bool              `json:"tracing_enabled"`
	LogLevel        string            `json:"log_level"`
	HealthCheckPath string            `json:"health_check_path"`
	SampleRate      float64           `json:"sample_rate"`
	Labels          map[string]string `json:"labels,omitempty"`
}

// HealthStatus represents the health status of a plugin
type HealthStatus struct {
	Status    string                 `json:"status"` // healthy, degraded, unhealthy
	LastCheck time.Time              `json:"last_check"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Checks    []HealthCheck          `json:"checks,omitempty"`
	Uptime    time.Duration          `json:"uptime"`
	Version   string                 `json:"version"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Name     string                 `json:"name"`
	Status   string                 `json:"status"`
	Duration time.Duration          `json:"duration"`
	Message  string                 `json:"message"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// PluginMetrics contains performance and usage metrics for a plugin
type PluginMetrics struct {
	// Request metrics
	TotalRequests     int64 `json:"total_requests"`
	SuccessfulExports int64 `json:"successful_exports"`
	FailedExports     int64 `json:"failed_exports"`
	QueuedRequests    int64 `json:"queued_requests"`

	// Performance metrics
	AverageLatency time.Duration `json:"average_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinLatency     time.Duration `json:"min_latency"`

	// Resource utilization
	MemoryUsage       int64   `json:"memory_usage_bytes"`
	CPUUsage          float64 `json:"cpu_usage_percent"`
	ActiveConnections int     `json:"active_connections"`

	// Error metrics
	ErrorRate   float64 `json:"error_rate"`
	TimeoutRate float64 `json:"timeout_rate"`
	RetryRate   float64 `json:"retry_rate"`

	// Data metrics
	BytesExported    int64   `json:"bytes_exported"`
	RecordsExported  int64   `json:"records_exported"`
	CompressionRatio float64 `json:"compression_ratio"`

	// Timestamps
	LastExport       time.Time `json:"last_export"`
	LastError        time.Time `json:"last_error"`
	MetricsUpdatedAt time.Time `json:"metrics_updated_at"`
}

// ExportMetrics contains metrics specific to an export operation
type ExportMetrics struct {
	ProcessingTime time.Duration `json:"processing_time"`
	QueueTime      time.Duration `json:"queue_time"`
	RetryCount     int           `json:"retry_count"`
	DataSize       int64         `json:"data_size_bytes"`
	CompressedSize int64         `json:"compressed_size_bytes,omitempty"`
	RecordCount    int64         `json:"record_count"`
	TransformTime  time.Duration `json:"transform_time"`
	DeliveryTime   time.Duration `json:"delivery_time"`
}

// PluginSchema defines the configuration schema for a plugin
type PluginSchema struct {
	Name        string                   `json:"name"`
	Version     string                   `json:"version"`
	Description string                   `json:"description"`
	Schema      map[string]SchemaField   `json:"schema"`
	Required    []string                 `json:"required"`
	Examples    []map[string]interface{} `json:"examples,omitempty"`
}

// SchemaField defines a configuration field in the plugin schema
type SchemaField struct {
	Type        string                 `json:"type"` // string, int, bool, array, object
	Description string                 `json:"description"`
	Default     interface{}            `json:"default,omitempty"`
	Required    bool                   `json:"required"`
	Enum        []string               `json:"enum,omitempty"`
	Format      string                 `json:"format,omitempty"` // email, url, regex, etc.
	MinLength   *int                   `json:"min_length,omitempty"`
	MaxLength   *int                   `json:"max_length,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Items       *SchemaField           `json:"items,omitempty"`      // For arrays
	Properties  map[string]SchemaField `json:"properties,omitempty"` // For objects
	Examples    []interface{}          `json:"examples,omitempty"`
	Sensitive   bool                   `json:"sensitive,omitempty"` // Mark sensitive fields
}

// CorrelationData represents correlation analysis data for export
type CorrelationData struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Correlations []Correlation          `json:"correlations"`
	Timeline     []TimelineEvent        `json:"timeline,omitempty"`
	Summary      CorrelationSummary     `json:"summary"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Correlation represents a single correlation finding
type Correlation struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Source          string                 `json:"source"`
	Target          string                 `json:"target"`
	Confidence      float64                `json:"confidence"`
	Strength        float64                `json:"strength"`
	Description     string                 `json:"description"`
	Evidence        []Evidence             `json:"evidence"`
	Recommendations []string               `json:"recommendations"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Evidence represents supporting evidence for a correlation
type Evidence struct {
	Type        string      `json:"type"`
	Source      string      `json:"source"`
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
	Confidence  float64     `json:"confidence"`
	Description string      `json:"description"`
}

// TimelineEvent represents an event in a correlation timeline
type TimelineEvent struct {
	ID          string      `json:"id"`
	Timestamp   time.Time   `json:"timestamp"`
	Type        string      `json:"type"`
	Source      string      `json:"source"`
	Description string      `json:"description"`
	Data        interface{} `json:"data"`
	Severity    string      `json:"severity"`
	Tags        []string    `json:"tags"`
}

// CorrelationSummary provides a summary of correlation analysis
type CorrelationSummary struct {
	TotalCorrelations  int                    `json:"total_correlations"`
	HighConfidence     int                    `json:"high_confidence"`
	MediumConfidence   int                    `json:"medium_confidence"`
	LowConfidence      int                    `json:"low_confidence"`
	CriticalFindings   int                    `json:"critical_findings"`
	TopCorrelations    []string               `json:"top_correlations"`
	RecommendedActions []string               `json:"recommended_actions"`
	RiskScore          float64                `json:"risk_score"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// ExportManager defines the interface for managing export plugins
type ExportManager interface {
	// Plugin management
	RegisterPlugin(plugin ExportPlugin) error
	UnregisterPlugin(name string) error
	GetPlugin(name string) (ExportPlugin, error)
	ListPlugins() []PluginInfo

	// Export operations
	Export(ctx context.Context, request *ExportRequest) (*ExportResponse, error)
	ExportAsync(ctx context.Context, request *ExportRequest) (string, error)
	GetExportStatus(exportID string) (*ExportResponse, error)
	CancelExport(exportID string) error

	// Configuration and health
	UpdatePluginConfig(pluginName string, config PluginConfig) error
	GetPluginHealth(pluginName string) (HealthStatus, error)
	GetSystemHealth() (map[string]HealthStatus, error)

	// Monitoring and metrics
	GetPluginMetrics(pluginName string) (PluginMetrics, error)
	GetSystemMetrics() (map[string]PluginMetrics, error)
}

// PluginInfo contains information about a registered plugin
type PluginInfo struct {
	Name             string         `json:"name"`
	Version          string         `json:"version"`
	Description      string         `json:"description"`
	Author           string         `json:"author,omitempty"`
	SupportedFormats []ExportFormat `json:"supported_formats"`
	Status           string         `json:"status"`
	Enabled          bool           `json:"enabled"`
	LoadedAt         time.Time      `json:"loaded_at"`
	ConfigSchema     PluginSchema   `json:"config_schema"`
	Capabilities     []string       `json:"capabilities"`
}

// ExportRouter defines the interface for routing export requests
type ExportRouter interface {
	Route(ctx context.Context, request *ExportRequest) (ExportPlugin, error)
	RegisterRoute(pattern RoutePattern, plugin ExportPlugin) error
	UnregisterRoute(pattern RoutePattern) error
	ListRoutes() []RouteInfo
}

// RoutePattern defines routing patterns for export requests
type RoutePattern struct {
	DataType    ExportDataType    `json:"data_type,omitempty"`
	Format      ExportFormat      `json:"format,omitempty"`
	Destination string            `json:"destination,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Priority    ExportPriority    `json:"priority,omitempty"`
	Conditions  []RouteCondition  `json:"conditions,omitempty"`
}

// RouteCondition defines a routing condition
type RouteCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, contains, regex
	Value    interface{} `json:"value"`
}

// RouteInfo contains information about a routing rule
type RouteInfo struct {
	ID          string       `json:"id"`
	Pattern     RoutePattern `json:"pattern"`
	PluginName  string       `json:"plugin_name"`
	Priority    int          `json:"priority"`
	Enabled     bool         `json:"enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	MatchCount  int64        `json:"match_count"`
	LastMatched time.Time    `json:"last_matched,omitempty"`
}
