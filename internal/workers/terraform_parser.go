package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// TerraformStateFile represents a Terraform state file
type TerraformStateFile struct {
	Path     string
	Size     int64
	ModTime  time.Time
	Priority int // Higher priority files are processed first
}

// ParseResult holds the result of parsing a Terraform state file
type ParseResult struct {
	FilePath  string
	Resources []types.Resource
	StateInfo StateInfo
	Error     error
	ParseTime time.Duration
	WorkerID  int
}

// StateInfo holds information about the Terraform state
type StateInfo struct {
	Version          int    `json:"version"`
	TerraformVersion string `json:"terraform_version"`
	Serial           int64  `json:"serial"`
	ResourceCount    int
	OutputCount      int
}

// TerraformParseJob represents a parsing job
type TerraformParseJob struct {
	StateFile    TerraformStateFile
	ParseOptions ParseOptions
	ResultChan   chan<- ParseResult
}

// ParseOptions configures parsing behavior
type ParseOptions struct {
	MaxResourceSize int64         // Maximum size of a single resource
	StreamingMode   bool          // Use streaming for large files
	ValidationMode  bool          // Enable resource validation
	Timeout         time.Duration // Timeout for parsing
	RetryCount      int           // Number of retries on failure
}

// ConcurrentTerraformParser handles concurrent parsing of Terraform state files
type ConcurrentTerraformParser struct {
	workerCount     int
	jobChan         chan TerraformParseJob
	workers         []*terraformWorker
	resultCollector *ResultCollector

	// Configuration
	maxFileSize  int64
	bufferSize   int
	parseTimeout time.Duration

	// State management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	totalParsed int64
	totalErrors int64
	totalTime   time.Duration
	mu          sync.RWMutex

	// Memory management
	memoryPool   *MemoryPool
	resourcePool *ResourcePool
}

// terraformWorker represents a single parser worker
type terraformWorker struct {
	id     int
	parser *ConcurrentTerraformParser
	stats  terraformWorkerStats
}

// terraformWorkerStats holds statistics for a parser worker
type terraformWorkerStats struct {
	filesParsed    int64
	resourcesFound int64
	errors         int64
	totalTime      time.Duration
	lastActive     time.Time
	mu             sync.RWMutex
}

// ResultCollector aggregates parsing results
type ResultCollector struct {
	results        []ParseResult
	errors         []error
	totalResources int
	mu             sync.RWMutex
}

// MemoryPool manages memory allocation for parsing
type MemoryPool struct {
	buffers  chan []byte
	capacity int
	size     int
}

// ResourcePool manages resource object pooling
type ResourcePool struct {
	pool sync.Pool
}

// NewConcurrentTerraformParser creates a new concurrent parser
func NewConcurrentTerraformParser(opts ...TerraformParserOption) *ConcurrentTerraformParser {
	parser := &ConcurrentTerraformParser{
		workerCount:     runtime.NumCPU(),
		bufferSize:      100,
		maxFileSize:     500 * 1024 * 1024, // 500MB
		parseTimeout:    60 * time.Second,
		resultCollector: &ResultCollector{},
		memoryPool:      NewMemoryPool(32, 1024*1024), // 32 buffers of 1MB each
		resourcePool:    NewResourcePool(),
	}

	// Apply options
	for _, opt := range opts {
		opt(parser)
	}

	// Initialize channels
	parser.jobChan = make(chan TerraformParseJob, parser.bufferSize)

	// Create context
	parser.ctx, parser.cancel = context.WithCancel(context.Background())

	// Initialize workers
	parser.workers = make([]*terraformWorker, parser.workerCount)
	for i := 0; i < parser.workerCount; i++ {
		parser.workers[i] = &terraformWorker{
			id:     i,
			parser: parser,
		}
	}

	return parser
}

// TerraformParserOption configures the parser
type TerraformParserOption func(*ConcurrentTerraformParser)

// WithTerraformWorkerCount sets the number of worker goroutines
func WithTerraformWorkerCount(count int) TerraformParserOption {
	return func(p *ConcurrentTerraformParser) {
		if count > 0 {
			p.workerCount = count
		}
	}
}

// WithTerraformBufferSize sets the buffer size for job channels
func WithTerraformBufferSize(size int) TerraformParserOption {
	return func(p *ConcurrentTerraformParser) {
		if size > 0 {
			p.bufferSize = size
		}
	}
}

// WithTerraformMaxFileSize sets the maximum file size to parse
func WithTerraformMaxFileSize(size int64) TerraformParserOption {
	return func(p *ConcurrentTerraformParser) {
		p.maxFileSize = size
	}
}

// WithTerraformTimeout sets the parsing timeout
func WithTerraformTimeout(timeout time.Duration) TerraformParserOption {
	return func(p *ConcurrentTerraformParser) {
		p.parseTimeout = timeout
	}
}

// ParseStatesConcurrent parses multiple Terraform state files concurrently
func (p *ConcurrentTerraformParser) ParseStatesConcurrent(statePaths []string) ([]types.Resource, error) {
	if len(statePaths) == 0 {
		return nil, fmt.Errorf("no state paths provided")
	}

	// Start workers
	if err := p.start(); err != nil {
		return nil, fmt.Errorf("failed to start parser: %w", err)
	}
	defer p.stop()

	// Create state files with metadata
	stateFiles := make([]TerraformStateFile, 0, len(statePaths))
	for _, path := range statePaths {
		stateFile, err := p.createStateFile(path)
		if err != nil {
			fmt.Printf("Warning: Failed to create state file for %s: %v\n", path, err)
			continue
		}
		stateFiles = append(stateFiles, stateFile)
	}

	if len(stateFiles) == 0 {
		return nil, fmt.Errorf("no valid state files found")
	}

	// Sort by priority (larger files first for better load balancing)
	p.sortStateFilesByPriority(stateFiles)

	// Create result channel
	resultChan := make(chan ParseResult, len(stateFiles))

	// Submit parse jobs
	for _, stateFile := range stateFiles {
		job := TerraformParseJob{
			StateFile: stateFile,
			ParseOptions: ParseOptions{
				MaxResourceSize: 10 * 1024 * 1024,              // 10MB per resource
				StreamingMode:   stateFile.Size > 50*1024*1024, // Stream if > 50MB
				ValidationMode:  true,
				Timeout:         p.parseTimeout,
				RetryCount:      2,
			},
			ResultChan: resultChan,
		}

		select {
		case p.jobChan <- job:
		case <-p.ctx.Done():
			return nil, fmt.Errorf("parser context cancelled")
		}
	}

	// Collect results
	var allResources []types.Resource
	var errors []error

	for i := 0; i < len(stateFiles); i++ {
		select {
		case result := <-resultChan:
			if result.Error != nil {
				errors = append(errors, fmt.Errorf("failed to parse %s: %w", result.FilePath, result.Error))
			} else {
				allResources = append(allResources, result.Resources...)
			}
		case <-time.After(p.parseTimeout * 2):
			errors = append(errors, fmt.Errorf("timeout waiting for parse results"))
			break
		}
	}

	// Return results even if some files failed
	if len(errors) > 0 && len(allResources) == 0 {
		return nil, fmt.Errorf("all state files failed to parse: %v", errors)
	}

	return allResources, nil
}

// start initializes and starts the parser workers
func (p *ConcurrentTerraformParser) start() error {
	// Start workers
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return nil
}

// stop gracefully shuts down the parser
func (p *ConcurrentTerraformParser) stop() {
	// Cancel context
	p.cancel()

	// Close job channel
	close(p.jobChan)

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Normal shutdown
	case <-time.After(10 * time.Second):
		// Force shutdown
		fmt.Println("Warning: Parser shutdown timed out")
	}
}

// worker processes parsing jobs
func (p *ConcurrentTerraformParser) worker(workerID int) {
	defer p.wg.Done()

	worker := p.workers[workerID]

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobChan:
			if !ok {
				return
			}

			// Process the job
			result := p.processParseJob(job, workerID)

			// Update worker stats
			worker.updateStats(result)

			// Send result
			select {
			case job.ResultChan <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// processParseJob processes a single parsing job
func (p *ConcurrentTerraformParser) processParseJob(job TerraformParseJob, workerID int) ParseResult {
	startTime := time.Now()

	result := ParseResult{
		FilePath:  job.StateFile.Path,
		WorkerID:  workerID,
		ParseTime: 0,
	}

	// Check file size
	if job.StateFile.Size > p.maxFileSize {
		result.Error = fmt.Errorf("file size %d exceeds maximum %d", job.StateFile.Size, p.maxFileSize)
		return result
	}

	// Parse with timeout
	ctx, cancel := context.WithTimeout(p.ctx, job.ParseOptions.Timeout)
	defer cancel()

	// Parse the file
	resources, stateInfo, err := p.parseStateFile(ctx, job.StateFile, job.ParseOptions)

	result.ParseTime = time.Since(startTime)
	result.Resources = resources
	result.StateInfo = stateInfo
	result.Error = err

	return result
}

// parseStateFile parses a single Terraform state file
func (p *ConcurrentTerraformParser) parseStateFile(ctx context.Context, stateFile TerraformStateFile, opts ParseOptions) ([]types.Resource, StateInfo, error) {
	// Open file
	file, err := os.Open(stateFile.Path)
	if err != nil {
		return nil, StateInfo{}, fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	// Get buffer from pool
	buffer := p.memoryPool.Get()
	defer p.memoryPool.Put(buffer)

	// Choose parsing strategy based on file size
	if opts.StreamingMode {
		return p.parseStreamingMode(ctx, file, stateFile, opts)
	} else {
		return p.parseStandardMode(ctx, file, stateFile, opts)
	}
}

// parseStandardMode parses using standard JSON unmarshaling
func (p *ConcurrentTerraformParser) parseStandardMode(ctx context.Context, file *os.File, stateFile TerraformStateFile, opts ParseOptions) ([]types.Resource, StateInfo, error) {
	// Read entire file
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, StateInfo{}, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var tfState struct {
		Version          int    `json:"version"`
		TerraformVersion string `json:"terraform_version"`
		Serial           int64  `json:"serial"`
		Resources        []struct {
			Type      string                   `json:"type"`
			Name      string                   `json:"name"`
			Provider  string                   `json:"provider"`
			Instances []map[string]interface{} `json:"instances"`
		} `json:"resources"`
		Outputs map[string]interface{} `json:"outputs"`
	}

	if err := json.Unmarshal(data, &tfState); err != nil {
		return nil, StateInfo{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to resources
	resources := make([]types.Resource, 0, len(tfState.Resources)*2)
	for _, tfResource := range tfState.Resources {
		for i, instance := range tfResource.Instances {
			resource := p.resourcePool.Get()

			// Set basic fields
			resource.ID = fmt.Sprintf("%s.%s[%d]", tfResource.Type, tfResource.Name, i)
			resource.Type = tfResource.Type
			resource.Name = tfResource.Name
			resource.Provider = "terraform"

			// Set configuration from instance
			if attributes, ok := instance["attributes"].(map[string]interface{}); ok {
				resource.Configuration = attributes
			}

			// Set metadata
			resource.Metadata = types.ResourceMetadata{
				StateFile:      stateFile.Path,
				StateVersion:   fmt.Sprintf("%d", tfState.Version),
				AdditionalData: make(map[string]interface{}),
			}
			resource.Metadata.AdditionalData["terraform_version"] = tfState.TerraformVersion
			resource.Metadata.AdditionalData["serial"] = tfState.Serial

			// Validate if requested
			if opts.ValidationMode {
				if err := resource.Validate(); err != nil {
					fmt.Printf("Warning: Resource validation failed for %s: %v\n", resource.ID, err)
					continue
				}
			}

			resources = append(resources, *resource)
			p.resourcePool.Put(resource)
		}
	}

	stateInfo := StateInfo{
		Version:          tfState.Version,
		TerraformVersion: tfState.TerraformVersion,
		Serial:           tfState.Serial,
		ResourceCount:    len(resources),
		OutputCount:      len(tfState.Outputs),
	}

	return resources, stateInfo, nil
}

// parseStreamingMode parses using streaming JSON decoder
func (p *ConcurrentTerraformParser) parseStreamingMode(ctx context.Context, file *os.File, stateFile TerraformStateFile, opts ParseOptions) ([]types.Resource, StateInfo, error) {
	_ = json.NewDecoder(file)

	// For streaming mode, we need to implement a streaming JSON parser
	// This is a simplified version - in production, you'd want to use a proper streaming parser
	return p.parseStandardMode(ctx, file, stateFile, opts)
}

// createStateFile creates a TerraformStateFile from a path
func (p *ConcurrentTerraformParser) createStateFile(path string) (TerraformStateFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return TerraformStateFile{}, err
	}

	priority := 100
	if info.Size() > 100*1024*1024 { // Files > 100MB get higher priority
		priority = 200
	}

	return TerraformStateFile{
		Path:     path,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Priority: priority,
	}, nil
}

// sortStateFilesByPriority sorts state files by priority (higher first)
func (p *ConcurrentTerraformParser) sortStateFilesByPriority(stateFiles []TerraformStateFile) {
	// Simple bubble sort - in production, use sort.Slice
	for i := 0; i < len(stateFiles); i++ {
		for j := i + 1; j < len(stateFiles); j++ {
			if stateFiles[i].Priority < stateFiles[j].Priority {
				stateFiles[i], stateFiles[j] = stateFiles[j], stateFiles[i]
			}
		}
	}
}

// updateStats updates worker statistics
func (w *terraformWorker) updateStats(result ParseResult) {
	w.stats.mu.Lock()
	defer w.stats.mu.Unlock()

	w.stats.lastActive = time.Now()
	w.stats.totalTime += result.ParseTime
	w.stats.filesParsed++

	if result.Error != nil {
		w.stats.errors++
	} else {
		w.stats.resourcesFound += int64(len(result.Resources))
	}
}

// GetStats returns parsing statistics
func (p *ConcurrentTerraformParser) GetStats() TerraformParsingStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := TerraformParsingStats{
		WorkerCount: p.workerCount,
		WorkerStats: make([]TerraformWorkerStats, len(p.workers)),
	}

	for i, worker := range p.workers {
		worker.stats.mu.RLock()
		stats.WorkerStats[i] = TerraformWorkerStats{
			WorkerID:       i,
			FilesParsed:    worker.stats.filesParsed,
			ResourcesFound: worker.stats.resourcesFound,
			Errors:         worker.stats.errors,
			TotalTime:      worker.stats.totalTime,
			LastActive:     worker.stats.lastActive,
		}
		worker.stats.mu.RUnlock()
	}

	return stats
}

// TerraformParsingStats holds parsing statistics
type TerraformParsingStats struct {
	WorkerCount int
	WorkerStats []TerraformWorkerStats
}

// TerraformWorkerStats holds individual worker statistics
type TerraformWorkerStats struct {
	WorkerID       int
	FilesParsed    int64
	ResourcesFound int64
	Errors         int64
	TotalTime      time.Duration
	LastActive     time.Time
}

// NewMemoryPool creates a new memory pool
func NewMemoryPool(capacity int, size int) *MemoryPool {
	pool := &MemoryPool{
		buffers:  make(chan []byte, capacity),
		capacity: capacity,
		size:     size,
	}

	// Pre-allocate buffers
	for i := 0; i < capacity; i++ {
		pool.buffers <- make([]byte, size)
	}

	return pool
}

// Get gets a buffer from the pool
func (mp *MemoryPool) Get() []byte {
	select {
	case buffer := <-mp.buffers:
		return buffer
	default:
		return make([]byte, mp.size)
	}
}

// Put returns a buffer to the pool
func (mp *MemoryPool) Put(buffer []byte) {
	select {
	case mp.buffers <- buffer:
	default:
		// Pool is full, buffer will be garbage collected
	}
}

// NewResourcePool creates a new resource pool
func NewResourcePool() *ResourcePool {
	return &ResourcePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &types.Resource{}
			},
		},
	}
}

// Get gets a resource from the pool
func (rp *ResourcePool) Get() *types.Resource {
	resource := rp.pool.Get().(*types.Resource)
	// Reset the resource
	*resource = types.Resource{}
	return resource
}

// Put returns a resource to the pool
func (rp *ResourcePool) Put(resource *types.Resource) {
	rp.pool.Put(resource)
}
