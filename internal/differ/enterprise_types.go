package differ

import (
	"sync"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// Type aliases for compatibility
type DriftSeverity = RiskLevel

// Additional severity constants for enterprise
const (
	SeverityLow      = RiskLevelLow
	SeverityMedium   = RiskLevelMedium
	SeverityHigh     = RiskLevelHigh
	SeverityCritical = RiskLevelCritical
)

// Additional category constants for enterprise
const (
	CategoryConfig   = DriftCategoryConfig
	CategorySecurity = DriftCategorySecurity
	CategoryNetwork  = DriftCategoryNetwork
	CategoryStorage  = DriftCategoryStorage
	CategoryCompute  = DriftCategoryCompute
)

// ChangeCorrelator identifies patterns and relationships between changes
type ChangeCorrelator interface {
	Correlate(changes []ResourceDiff) []ChangeCorrelation
}

// RiskAssessor provides advanced risk scoring for changes
type RiskAssessor interface {
	AssessResourceRisk(diff ResourceDiff, baseline, current types.Resource) float64
	AssessNewResourceRisk(resource types.Resource) float64
	AssessRemovedResourceRisk(resource types.Resource) float64
}

// ChangeCorrelation represents a correlation between multiple changes
type ChangeCorrelation struct {
	ID          string                 `json:"id"`
	Pattern     string                 `json:"pattern"`
	Description string                 `json:"description"`
	ResourceIDs []string               `json:"resource_ids"`
	Confidence  float64                `json:"confidence"`
	Risk        float64                `json:"risk"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ComplianceRule defines a compliance rule for drift analysis
type ComplianceRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    DriftSeverity          `json:"severity"`
	Category    DriftCategory          `json:"category"`
	Condition   string                 `json:"condition"` // Expression to evaluate
	Remediation string                 `json:"remediation"`
	Framework   string                 `json:"framework"` // SOC2, PCI-DSS, etc.
	Metadata    map[string]interface{} `json:"metadata"`
}

// ComplianceReport contains compliance analysis results
type ComplianceReport struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Framework  string                 `json:"framework"`
	Status     string                 `json:"status"` // Compliant, NonCompliant, Unknown
	Score      float64                `json:"score"`  // 0-100 compliance score
	Violations []ComplianceViolation  `json:"violations"`
	Summary    ComplianceSummary      `json:"summary"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ComplianceViolation represents a single compliance violation
type ComplianceViolation struct {
	RuleID      string                 `json:"rule_id"`
	ResourceID  string                 `json:"resource_id"`
	Severity    DriftSeverity          `json:"severity"`
	Description string                 `json:"description"`
	Remediation string                 `json:"remediation"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ComplianceSummary provides a high-level compliance overview
type ComplianceSummary struct {
	TotalRules     int `json:"total_rules"`
	PassedRules    int `json:"passed_rules"`
	FailedRules    int `json:"failed_rules"`
	CriticalIssues int `json:"critical_issues"`
	HighIssues     int `json:"high_issues"`
	MediumIssues   int `json:"medium_issues"`
	LowIssues      int `json:"low_issues"`
}

// ExecutiveSummary provides C-level reporting information
type ExecutiveSummary struct {
	OverallRisk      string                 `json:"overall_risk"`
	ComplianceStatus string                 `json:"compliance_status"`
	KeyFindings      []string               `json:"key_findings"`
	Recommendations  []string               `json:"recommendations"`
	Metrics          map[string]interface{} `json:"metrics"`
	Timestamp        time.Time              `json:"timestamp"`
}

// WorkerPool manages concurrent diff processing
type WorkerPool struct {
	workers  int
	jobs     chan func()
	wg       sync.WaitGroup
	shutdown chan struct{}
	active   int32
}

// DiffCache provides caching for diff results
type DiffCache struct {
	data    map[string]interface{}
	mu      sync.RWMutex
	maxSize int
	hits    int64
	misses  int64
}

// ResourceIndex provides fast resource lookups
type ResourceIndex struct {
	byID       map[string]types.Resource
	byType     map[string][]types.Resource
	byProvider map[string][]types.Resource
	byRegion   map[string][]types.Resource
	byTag      map[string][]types.Resource
	mu         sync.RWMutex
}

// NewChangeCorrelator creates a new change correlator
func NewChangeCorrelator() ChangeCorrelator {
	return &DefaultChangeCorrelator{
		timeWindow: 30 * time.Second,
	}
}

// NewRiskAssessor creates a new risk assessor
func NewRiskAssessor() RiskAssessor {
	return &DefaultRiskAssessor{
		riskMatrix: buildDefaultRiskMatrix(),
	}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	pool := &WorkerPool{
		workers:  workers,
		jobs:     make(chan func(), workers*2),
		shutdown: make(chan struct{}),
	}
	pool.start()
	return pool
}

// NewDiffCache creates a new diff cache
func NewDiffCache(maxSize int) *DiffCache {
	return &DiffCache{
		data:    make(map[string]interface{}),
		maxSize: maxSize,
	}
}

// NewResourceIndex creates a new resource index
func NewResourceIndex() *ResourceIndex {
	return &ResourceIndex{
		byID:       make(map[string]types.Resource),
		byType:     make(map[string][]types.Resource),
		byProvider: make(map[string][]types.Resource),
		byRegion:   make(map[string][]types.Resource),
		byTag:      make(map[string][]types.Resource),
	}
}

// WorkerPool methods

func (wp *WorkerPool) start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for {
		select {
		case job := <-wp.jobs:
			job()
		case <-wp.shutdown:
			return
		}
	}
}

func (wp *WorkerPool) Submit(job func()) {
	select {
	case wp.jobs <- job:
	case <-wp.shutdown:
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.shutdown)
	wp.wg.Wait()
}

// DiffCache methods

func (dc *DiffCache) Get(key string) (interface{}, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	value, exists := dc.data[key]
	if exists {
		dc.hits++
		return value, true
	}
	dc.misses++
	return nil, false
}

func (dc *DiffCache) Set(key string, value interface{}) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Simple LRU eviction (could be enhanced)
	if len(dc.data) >= dc.maxSize {
		// Remove oldest entry (simplified)
		for k := range dc.data {
			delete(dc.data, k)
			break
		}
	}

	dc.data[key] = value
}

func (dc *DiffCache) Stats() (hits, misses int64) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.hits, dc.misses
}

// ResourceIndex methods

func (ri *ResourceIndex) AddBaseline(resource types.Resource) {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	ri.byID[resource.ID] = resource
	ri.byType[resource.Type] = append(ri.byType[resource.Type], resource)
	ri.byProvider[resource.Provider] = append(ri.byProvider[resource.Provider], resource)
	ri.byRegion[resource.Region] = append(ri.byRegion[resource.Region], resource)

	for key := range resource.Tags {
		ri.byTag[key] = append(ri.byTag[key], resource)
	}
}

func (ri *ResourceIndex) AddCurrent(resource types.Resource) {
	// For now, use same logic as baseline
	// In practice, you might want separate indexes
	ri.AddBaseline(resource)
}

func (ri *ResourceIndex) BuildSecondaryIndexes() {
	// Placeholder for building optimized secondary indexes
	// Could include spatial indexes for regions, full-text for names, etc.
}

func (ri *ResourceIndex) FindByID(id string) (types.Resource, bool) {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	resource, exists := ri.byID[id]
	return resource, exists
}

func (ri *ResourceIndex) FindByType(resourceType string) []types.Resource {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	return ri.byType[resourceType]
}

// Default implementations

// DefaultChangeCorrelator implements basic change correlation
type DefaultChangeCorrelator struct {
	timeWindow time.Duration
}

func (dcc *DefaultChangeCorrelator) Correlate(changes []ResourceDiff) []ChangeCorrelation {
	var correlations []ChangeCorrelation

	// Group by time windows
	timeGroups := dcc.groupByTimeWindow(changes)

	for _, group := range timeGroups {
		// Look for patterns within each time group
		patterns := dcc.detectPatterns(group)
		correlations = append(correlations, patterns...)
	}

	return correlations
}

func (dcc *DefaultChangeCorrelator) groupByTimeWindow(changes []ResourceDiff) [][]ResourceDiff {
	// Simple time-based grouping implementation
	// In practice, this would be more sophisticated
	return [][]ResourceDiff{changes}
}

func (dcc *DefaultChangeCorrelator) detectPatterns(changes []ResourceDiff) []ChangeCorrelation {
	var correlations []ChangeCorrelation

	// Detect common patterns
	if scaling := dcc.detectScalingPattern(changes); scaling != nil {
		correlations = append(correlations, *scaling)
	}

	if deployment := dcc.detectDeploymentPattern(changes); deployment != nil {
		correlations = append(correlations, *deployment)
	}

	if security := dcc.detectSecurityPattern(changes); security != nil {
		correlations = append(correlations, *security)
	}

	return correlations
}

func (dcc *DefaultChangeCorrelator) detectScalingPattern(changes []ResourceDiff) *ChangeCorrelation {
	// Look for scaling-related changes (instances, autoscaling groups, etc.)
	scalingResources := 0
	var resourceIDs []string

	for _, change := range changes {
		if isScalingResource(change.ResourceType) {
			scalingResources++
			resourceIDs = append(resourceIDs, change.ResourceID)
		}
	}

	if scalingResources >= 2 {
		return &ChangeCorrelation{
			ID:          "scaling-" + time.Now().Format("20060102-150405"),
			Pattern:     "scaling_operation",
			Description: "Multiple scaling-related changes detected",
			ResourceIDs: resourceIDs,
			Confidence:  0.8,
			Risk:        0.3, // Scaling is usually low risk
			Timestamp:   time.Now(),
		}
	}

	return nil
}

func (dcc *DefaultChangeCorrelator) detectDeploymentPattern(changes []ResourceDiff) *ChangeCorrelation {
	// Look for deployment-related changes
	deploymentResources := 0
	var resourceIDs []string

	for _, change := range changes {
		if isDeploymentResource(change.ResourceType) {
			deploymentResources++
			resourceIDs = append(resourceIDs, change.ResourceID)
		}
	}

	if deploymentResources >= 2 {
		return &ChangeCorrelation{
			ID:          "deployment-" + time.Now().Format("20060102-150405"),
			Pattern:     "deployment_operation",
			Description: "Application deployment changes detected",
			ResourceIDs: resourceIDs,
			Confidence:  0.9,
			Risk:        0.5, // Deployments have medium risk
			Timestamp:   time.Now(),
		}
	}

	return nil
}

func (dcc *DefaultChangeCorrelator) detectSecurityPattern(changes []ResourceDiff) *ChangeCorrelation {
	// Look for security-related changes
	securityResources := 0
	var resourceIDs []string

	for _, change := range changes {
		if isSecurityResource(change.ResourceType) || hasSecurityChanges(change) {
			securityResources++
			resourceIDs = append(resourceIDs, change.ResourceID)
		}
	}

	if securityResources >= 1 {
		return &ChangeCorrelation{
			ID:          "security-" + time.Now().Format("20060102-150405"),
			Pattern:     "security_modification",
			Description: "Security configuration changes detected",
			ResourceIDs: resourceIDs,
			Confidence:  0.95,
			Risk:        0.8, // Security changes are high risk
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// DefaultRiskAssessor implements basic risk assessment
type DefaultRiskAssessor struct {
	riskMatrix map[string]float64
}

func (dra *DefaultRiskAssessor) AssessResourceRisk(diff ResourceDiff, baseline, current types.Resource) float64 {
	baseRisk := diff.RiskScore

	// Adjust based on resource type
	if typeRisk, exists := dra.riskMatrix[current.Type]; exists {
		baseRisk *= typeRisk
	}

	// Adjust based on change severity
	switch diff.Severity {
	case SeverityCritical:
		baseRisk *= 1.5
	case SeverityHigh:
		baseRisk *= 1.2
	case SeverityMedium:
		baseRisk *= 1.0
	case SeverityLow:
		baseRisk *= 0.8
	}

	// Cap at 1.0
	if baseRisk > 1.0 {
		baseRisk = 1.0
	}

	return baseRisk
}

func (dra *DefaultRiskAssessor) AssessNewResourceRisk(resource types.Resource) float64 {
	baseRisk := 0.3 // New resources are generally medium-low risk

	// Adjust based on resource type
	if typeRisk, exists := dra.riskMatrix[resource.Type]; exists {
		baseRisk = typeRisk * 0.7 // New resources are lower risk than modifications
	}

	// Increase risk for critical resource types
	if isCriticalResource(resource) {
		baseRisk *= 1.5
	}

	if baseRisk > 1.0 {
		baseRisk = 1.0
	}

	return baseRisk
}

func (dra *DefaultRiskAssessor) AssessRemovedResourceRisk(resource types.Resource) float64 {
	baseRisk := 0.7 // Removed resources are generally higher risk

	// Adjust based on resource type
	if typeRisk, exists := dra.riskMatrix[resource.Type]; exists {
		baseRisk = typeRisk * 1.2 // Removals are higher risk than modifications
	}

	// Critical resources being removed are very high risk
	if isCriticalResource(resource) {
		baseRisk = 0.95
	}

	if baseRisk > 1.0 {
		baseRisk = 1.0
	}

	return baseRisk
}

// Helper functions

func buildDefaultRiskMatrix() map[string]float64 {
	return map[string]float64{
		// AWS Critical Resources
		"aws_iam_role":            0.9,
		"aws_iam_policy":          0.9,
		"aws_security_group":      0.8,
		"aws_kms_key":             0.9,
		"aws_rds_cluster":         0.7,
		"aws_elasticache_cluster": 0.6,

		// AWS Standard Resources
		"aws_instance":        0.5,
		"aws_s3_bucket":       0.6,
		"aws_lambda_function": 0.4,
		"aws_vpc":             0.7,
		"aws_subnet":          0.6,

		// Kubernetes Resources
		"kubernetes_secret":          0.9,
		"kubernetes_service_account": 0.8,
		"kubernetes_deployment":      0.5,
		"kubernetes_service":         0.4,
		"kubernetes_configmap":       0.3,

		// Default for unknown types
		"default": 0.5,
	}
}

func isScalingResource(resourceType string) bool {
	scalingTypes := map[string]bool{
		"aws_autoscaling_group":              true,
		"aws_instance":                       true,
		"kubernetes_deployment":              true,
		"kubernetes_replicaset":              true,
		"kubernetes_horizontalpodautoscaler": true,
	}
	return scalingTypes[resourceType]
}

func isDeploymentResource(resourceType string) bool {
	deploymentTypes := map[string]bool{
		"kubernetes_deployment":  true,
		"kubernetes_statefulset": true,
		"kubernetes_daemonset":   true,
		"aws_ecs_service":        true,
		"aws_lambda_function":    true,
	}
	return deploymentTypes[resourceType]
}

func isSecurityResource(resourceType string) bool {
	securityTypes := map[string]bool{
		"aws_security_group":         true,
		"aws_iam_role":               true,
		"aws_iam_policy":             true,
		"aws_kms_key":                true,
		"kubernetes_secret":          true,
		"kubernetes_service_account": true,
		"kubernetes_role":            true,
		"kubernetes_rolebinding":     true,
	}
	return securityTypes[resourceType]
}

func hasSecurityChanges(diff ResourceDiff) bool {
	for _, category := range diff.Categories {
		if category == CategorySecurity {
			return true
		}
	}
	return false
}

// Note: isCriticalResource function is defined in enterprise_engine_impl.go
