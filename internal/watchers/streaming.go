package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// EventPipeline provides real-time event streaming with parallel processing
type EventPipeline struct {
	mu                  sync.RWMutex
	stages              []PipelineStage
	inputChannel        chan WatchEvent
	outputChannel       chan ProcessedEvent
	errorChannel        chan error
	processingGroups    map[string]*ProcessingGroup
	bufferSize          int
	maxConcurrency      int
	running             bool
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	stats               EventPipelineStats
	eventRouter         *EventRouter
	backpressureManager *BackpressureManager
	circuitBreaker      *CircuitBreaker
}

// PipelineStage represents a processing stage in the pipeline
type PipelineStage struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Processor   EventProcessor     `json:"-"`
	Concurrency int                `json:"concurrency"`
	BufferSize  int                `json:"buffer_size"`
	Timeout     time.Duration      `json:"timeout"`
	RetryPolicy RetryPolicy        `json:"retry_policy"`
	Enabled     bool               `json:"enabled"`
	Stats       PipelineStageStats `json:"stats"`
}

// EventProcessor interface for processing events
type EventProcessor interface {
	Process(ctx context.Context, event WatchEvent) (ProcessedEvent, error)
	Name() string
	CanProcess(event WatchEvent) bool
}

// ProcessedEvent represents an event that has been processed
type ProcessedEvent struct {
	ID             string                 `json:"id"`
	OriginalEvent  WatchEvent             `json:"original_event"`
	ProcessedAt    time.Time              `json:"processed_at"`
	ProcessorID    string                 `json:"processor_id"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Enrichments    map[string]interface{} `json:"enrichments"`
	Severity       types.DriftSeverity    `json:"severity"`
	Priority       int                    `json:"priority"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ProcessingGroup represents a group of concurrent workers
type ProcessingGroup struct {
	ID            string               `json:"id"`
	Workers       []*Worker            `json:"workers"`
	InputChannel  chan WatchEvent      `json:"-"`
	OutputChannel chan ProcessedEvent  `json:"-"`
	ErrorChannel  chan error           `json:"-"`
	Stats         ProcessingGroupStats `json:"stats"`
	Running       bool                 `json:"running"`
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// Worker represents a concurrent worker in the pipeline
type Worker struct {
	ID        string         `json:"id"`
	GroupID   string         `json:"group_id"`
	Processor EventProcessor `json:"-"`
	Stats     WorkerStats    `json:"stats"`
	Running   bool           `json:"running"`
	ctx       context.Context
	cancel    context.CancelFunc
}

// RetryPolicy defines retry behavior for failed processing
type RetryPolicy struct {
	MaxAttempts     int           `json:"max_attempts"`
	BackoffStrategy string        `json:"backoff_strategy"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	Multiplier      float64       `json:"multiplier"`
	Enabled         bool          `json:"enabled"`
}

// EventRouter routes events to appropriate processing groups
type EventRouter struct {
	mu           sync.RWMutex
	rules        []RoutingRule
	defaultGroup string
	stats        EventRouterStats
}

// RoutingRule defines how to route events
type RoutingRule struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Conditions []RoutingCondition `json:"conditions"`
	Target     string             `json:"target"`
	Priority   int                `json:"priority"`
	Enabled    bool               `json:"enabled"`
}

// RoutingCondition defines conditions for routing
type RoutingCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// BackpressureManager handles backpressure in the pipeline
type BackpressureManager struct {
	mu                    sync.RWMutex
	enabled               bool
	maxQueueSize          int
	dropPolicy            string
	throttleThreshold     float64
	currentLoad           float64
	dropCount             int64
	throttleCount         int64
	backpressureCallbacks []BackpressureCallback
}

// BackpressureCallback is called when backpressure is detected
type BackpressureCallback func(currentLoad float64, action string)

// CircuitBreaker provides circuit breaker functionality
type CircuitBreaker struct {
	mu               sync.RWMutex
	enabled          bool
	failureThreshold int
	resetTimeout     time.Duration
	state            CircuitBreakerState
	failureCount     int
	lastFailureTime  time.Time
	successCount     int
	totalRequests    int64
	stats            CircuitBreakerStats
}

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// Statistics structures
type EventPipelineStats struct {
	TotalProcessed      int64                           `json:"total_processed"`
	ProcessingRate      float64                         `json:"processing_rate"`
	AverageLatency      time.Duration                   `json:"average_latency"`
	ErrorRate           float64                         `json:"error_rate"`
	ThroughputPerSecond float64                         `json:"throughput_per_second"`
	BackpressureEvents  int64                           `json:"backpressure_events"`
	CircuitBreakerTrips int64                           `json:"circuit_breaker_trips"`
	StageStats          map[string]PipelineStageStats   `json:"stage_stats"`
	GroupStats          map[string]ProcessingGroupStats `json:"group_stats"`
	LastActivity        time.Time                       `json:"last_activity"`
}

type PipelineStageStats struct {
	Processed      int64         `json:"processed"`
	Errors         int64         `json:"errors"`
	AverageLatency time.Duration `json:"average_latency"`
	Throughput     float64       `json:"throughput"`
	LastActivity   time.Time     `json:"last_activity"`
}

type ProcessingGroupStats struct {
	ActiveWorkers  int           `json:"active_workers"`
	QueueSize      int           `json:"queue_size"`
	Processed      int64         `json:"processed"`
	Errors         int64         `json:"errors"`
	AverageLatency time.Duration `json:"average_latency"`
	Throughput     float64       `json:"throughput"`
	LastActivity   time.Time     `json:"last_activity"`
}

type WorkerStats struct {
	Processed      int64         `json:"processed"`
	Errors         int64         `json:"errors"`
	AverageLatency time.Duration `json:"average_latency"`
	Uptime         time.Duration `json:"uptime"`
	LastActivity   time.Time     `json:"last_activity"`
	StartTime      time.Time     `json:"start_time"`
}

type EventRouterStats struct {
	TotalRouted    int64            `json:"total_routed"`
	RoutingLatency time.Duration    `json:"routing_latency"`
	RuleStats      map[string]int64 `json:"rule_stats"`
	DefaultRouted  int64            `json:"default_routed"`
}

type CircuitBreakerStats struct {
	State           CircuitBreakerState `json:"state"`
	FailureCount    int                 `json:"failure_count"`
	SuccessCount    int                 `json:"success_count"`
	TotalRequests   int64               `json:"total_requests"`
	LastFailure     time.Time           `json:"last_failure"`
	LastStateChange time.Time           `json:"last_state_change"`
}

// NewEventPipeline creates a new event pipeline
func NewEventPipeline(bufferSize, maxConcurrency int) *EventPipeline {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventPipeline{
		stages:           []PipelineStage{},
		inputChannel:     make(chan WatchEvent, bufferSize),
		outputChannel:    make(chan ProcessedEvent, bufferSize),
		errorChannel:     make(chan error, bufferSize),
		processingGroups: make(map[string]*ProcessingGroup),
		bufferSize:       bufferSize,
		maxConcurrency:   maxConcurrency,
		running:          false,
		ctx:              ctx,
		cancel:           cancel,
		stats: EventPipelineStats{
			StageStats: make(map[string]PipelineStageStats),
			GroupStats: make(map[string]ProcessingGroupStats),
		},
		eventRouter:         NewEventRouter(),
		backpressureManager: NewBackpressureManager(),
		circuitBreaker:      NewCircuitBreaker(),
	}
}

// NewEventRouter creates a new event router
func NewEventRouter() *EventRouter {
	return &EventRouter{
		rules:        []RoutingRule{},
		defaultGroup: "default",
		stats: EventRouterStats{
			RuleStats: make(map[string]int64),
		},
	}
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager() *BackpressureManager {
	return &BackpressureManager{
		enabled:               true,
		maxQueueSize:          10000,
		dropPolicy:            "oldest",
		throttleThreshold:     0.8,
		currentLoad:           0.0,
		dropCount:             0,
		throttleCount:         0,
		backpressureCallbacks: []BackpressureCallback{},
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		enabled:          true,
		failureThreshold: 5,
		resetTimeout:     30 * time.Second,
		state:            CircuitBreakerClosed,
		failureCount:     0,
		successCount:     0,
		totalRequests:    0,
		stats:            CircuitBreakerStats{},
	}
}

// Start starts the event pipeline
func (ep *EventPipeline) Start() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.running {
		return fmt.Errorf("event pipeline is already running")
	}

	// Start processing groups
	for _, group := range ep.processingGroups {
		if err := group.Start(); err != nil {
			return fmt.Errorf("failed to start processing group %s: %w", group.ID, err)
		}
	}

	ep.running = true

	// Start main processing loop
	ep.wg.Add(1)
	go ep.processingLoop()

	// Start statistics collection
	go ep.statsLoop()

	return nil
}

// Stop stops the event pipeline
func (ep *EventPipeline) Stop() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if !ep.running {
		return fmt.Errorf("event pipeline is not running")
	}

	ep.cancel()
	ep.running = false

	// Stop processing groups
	for _, group := range ep.processingGroups {
		group.Stop()
	}

	// Wait for processing loop to finish
	ep.wg.Wait()

	// Close channels
	close(ep.inputChannel)
	close(ep.outputChannel)
	close(ep.errorChannel)

	return nil
}

// AddStage adds a processing stage to the pipeline
func (ep *EventPipeline) AddStage(stage PipelineStage) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Create processing group for this stage
	group := &ProcessingGroup{
		ID:            stage.ID,
		Workers:       []*Worker{},
		InputChannel:  make(chan WatchEvent, stage.BufferSize),
		OutputChannel: make(chan ProcessedEvent, stage.BufferSize),
		ErrorChannel:  make(chan error, stage.BufferSize),
		Stats:         ProcessingGroupStats{},
		Running:       false,
	}

	// Create workers
	for i := 0; i < stage.Concurrency; i++ {
		worker := &Worker{
			ID:        fmt.Sprintf("%s-worker-%d", stage.ID, i),
			GroupID:   stage.ID,
			Processor: stage.Processor,
			Stats:     WorkerStats{},
			Running:   false,
		}
		group.Workers = append(group.Workers, worker)
	}

	ep.stages = append(ep.stages, stage)
	ep.processingGroups[stage.ID] = group
	ep.stats.StageStats[stage.ID] = PipelineStageStats{}
	ep.stats.GroupStats[stage.ID] = ProcessingGroupStats{}

	return nil
}

// SendEvent sends an event to the pipeline
func (ep *EventPipeline) SendEvent(event WatchEvent) error {
	// Check circuit breaker
	if !ep.circuitBreaker.AllowRequest() {
		return fmt.Errorf("circuit breaker is open")
	}

	// Check backpressure
	if ep.backpressureManager.ShouldDrop() {
		ep.backpressureManager.RecordDrop()
		return fmt.Errorf("backpressure detected, dropping event")
	}

	select {
	case ep.inputChannel <- event:
		return nil
	default:
		// Channel is full
		ep.backpressureManager.RecordDrop()
		return fmt.Errorf("input channel is full")
	}
}

// OutputChannel returns the output channel
func (ep *EventPipeline) OutputChannel() <-chan ProcessedEvent {
	return ep.outputChannel
}

// ErrorChannel returns the error channel
func (ep *EventPipeline) ErrorChannel() <-chan error {
	return ep.errorChannel
}

// GetStats returns pipeline statistics
func (ep *EventPipeline) GetStats() EventPipelineStats {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	// Update group stats
	for groupID, group := range ep.processingGroups {
		ep.stats.GroupStats[groupID] = group.Stats
	}

	return ep.stats
}

// processingLoop is the main processing loop
func (ep *EventPipeline) processingLoop() {
	defer ep.wg.Done()

	for {
		select {
		case <-ep.ctx.Done():
			return
		case event, ok := <-ep.inputChannel:
			if !ok {
				return
			}

			startTime := time.Now()

			// Route event to appropriate processing group
			groupID := ep.eventRouter.RouteEvent(event)

			if group, exists := ep.processingGroups[groupID]; exists {
				// Send to processing group
				select {
				case group.InputChannel <- event:
					// Successfully sent
				default:
					// Group is busy, send to error channel
					select {
					case ep.errorChannel <- fmt.Errorf("processing group %s is busy", groupID):
					default:
						// Error channel is full too
					}
				}
			} else {
				// No group found, send to error channel
				select {
				case ep.errorChannel <- fmt.Errorf("no processing group found for event %s", event.ID):
				default:
					// Error channel is full
				}
			}

			// Update stats
			ep.mu.Lock()
			ep.stats.TotalProcessed++
			ep.stats.LastActivity = time.Now()

			latency := time.Since(startTime)
			if ep.stats.AverageLatency == 0 {
				ep.stats.AverageLatency = latency
			} else {
				ep.stats.AverageLatency = time.Duration((int64(ep.stats.AverageLatency) + int64(latency)) / 2)
			}
			ep.mu.Unlock()
		}
	}
}

// statsLoop periodically updates statistics
func (ep *EventPipeline) statsLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ep.ctx.Done():
			return
		case <-ticker.C:
			ep.updateStats()
		}
	}
}

// updateStats updates pipeline statistics
func (ep *EventPipeline) updateStats() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Update processing rate
	if ep.stats.TotalProcessed > 0 && !ep.stats.LastActivity.IsZero() {
		duration := time.Since(ep.stats.LastActivity)
		if duration > 0 {
			ep.stats.ProcessingRate = float64(ep.stats.TotalProcessed) / duration.Seconds()
		}
	}

	// Update throughput
	ep.stats.ThroughputPerSecond = ep.stats.ProcessingRate

	// Update backpressure events
	ep.stats.BackpressureEvents = ep.backpressureManager.GetDropCount()

	// Update circuit breaker stats
	ep.stats.CircuitBreakerTrips = ep.circuitBreaker.GetTripCount()
}

// ProcessingGroup methods
func (pg *ProcessingGroup) Start() error {
	pg.ctx, pg.cancel = context.WithCancel(context.Background())
	pg.Running = true

	// Start all workers
	for _, worker := range pg.Workers {
		pg.wg.Add(1)
		go worker.Start(pg.ctx, pg.InputChannel, pg.OutputChannel, pg.ErrorChannel, &pg.wg)
	}

	return nil
}

func (pg *ProcessingGroup) Stop() {
	if pg.cancel != nil {
		pg.cancel()
	}
	pg.Running = false

	// Wait for all workers to finish
	pg.wg.Wait()

	// Close channels
	close(pg.InputChannel)
	close(pg.OutputChannel)
	close(pg.ErrorChannel)
}

// Worker methods
func (w *Worker) Start(ctx context.Context, input <-chan WatchEvent, output chan<- ProcessedEvent, errorChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.Running = true
	w.Stats.StartTime = time.Now()

	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-input:
			if !ok {
				return
			}

			startTime := time.Now()

			// Process the event
			processed, err := w.Processor.Process(w.ctx, event)
			if err != nil {
				w.Stats.Errors++
				select {
				case errorChan <- err:
				default:
					// Error channel is full
				}
				continue
			}

			// Send processed event
			select {
			case output <- processed:
				w.Stats.Processed++
				w.Stats.LastActivity = time.Now()

				// Update latency
				latency := time.Since(startTime)
				if w.Stats.AverageLatency == 0 {
					w.Stats.AverageLatency = latency
				} else {
					w.Stats.AverageLatency = time.Duration((int64(w.Stats.AverageLatency) + int64(latency)) / 2)
				}
			default:
				// Output channel is full
				w.Stats.Errors++
			}
		}
	}
}

// EventRouter methods
func (er *EventRouter) RouteEvent(event WatchEvent) string {
	er.mu.RLock()
	defer er.mu.RUnlock()

	// Check routing rules
	for _, rule := range er.rules {
		if !rule.Enabled {
			continue
		}

		if er.evaluateRule(rule, event) {
			er.stats.RuleStats[rule.ID]++
			return rule.Target
		}
	}

	// No rule matched, use default
	er.stats.DefaultRouted++
	return er.defaultGroup
}

func (er *EventRouter) evaluateRule(rule RoutingRule, event WatchEvent) bool {
	for _, condition := range rule.Conditions {
		if !er.evaluateCondition(condition, event) {
			return false
		}
	}
	return true
}

func (er *EventRouter) evaluateCondition(condition RoutingCondition, event WatchEvent) bool {
	var fieldValue interface{}

	switch condition.Field {
	case "provider":
		fieldValue = event.Provider
	case "type":
		fieldValue = event.Type
	case "resource.type":
		fieldValue = event.Resource.Type
	case "resource.region":
		fieldValue = event.Resource.Region
	case "resource.namespace":
		fieldValue = event.Resource.Namespace
	}

	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if searchStr, ok := condition.Value.(string); ok {
				return str == searchStr // Simplified
			}
		}
	}

	return false
}

// BackpressureManager methods
func (bm *BackpressureManager) ShouldDrop() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	return bm.enabled && bm.currentLoad > bm.throttleThreshold
}

func (bm *BackpressureManager) RecordDrop() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.dropCount++

	// Notify callbacks
	for _, callback := range bm.backpressureCallbacks {
		callback(bm.currentLoad, "drop")
	}
}

func (bm *BackpressureManager) GetDropCount() int64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.dropCount
}

// CircuitBreaker methods
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.enabled {
		return true
	}

	cb.totalRequests++

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.stats.LastStateChange = time.Now()
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	}

	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++

	if cb.state == CircuitBreakerHalfOpen {
		cb.state = CircuitBreakerClosed
		cb.failureCount = 0
		cb.stats.LastStateChange = time.Now()
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = CircuitBreakerOpen
		cb.stats.LastStateChange = time.Now()
	}
}

func (cb *CircuitBreaker) GetTripCount() int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Simple implementation - count how many times we've been open
	if cb.state == CircuitBreakerOpen {
		return 1
	}
	return 0
}

// IsRunning returns whether the pipeline is running
func (ep *EventPipeline) IsRunning() bool {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.running
}
