package journald

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogFilter provides intelligent filtering and rate limiting for log entries
type LogFilter struct {
	filters        []Filter
	rateLimit      *RateLimit
	bloomFilter    *BloomFilter
	duplicateCache *DuplicateCache
	stats          FilterStats
	mu             sync.RWMutex
}

// Filter interface for different filtering strategies
type Filter interface {
	ShouldProcess(entry LogEntry) bool
	Name() string
	GetStats() FilterStats
}

// RateLimit manages rate limiting for log processing
type RateLimit struct {
	maxPerSecond int
	maxPerMinute int
	maxPerHour   int
	buckets      map[string]*TokenBucket
	globalBucket *TokenBucket
	mu           sync.RWMutex
}

// TokenBucket implements token bucket rate limiting
type TokenBucket struct {
	capacity   int
	tokens     int
	refillRate int // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// BloomFilter provides memory-efficient duplicate detection
type BloomFilter struct {
	bitArray  []bool
	size      int
	hashFuncs []HashFunc
	elements  int
	mu        sync.RWMutex
}

// HashFunc represents a hash function for bloom filter
type HashFunc func(data []byte) uint32

// DuplicateCache tracks recent duplicate messages
type DuplicateCache struct {
	cache   map[string]*DuplicateEntry
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
}

// DuplicateEntry tracks duplicate message information
type DuplicateEntry struct {
	hash       string
	count      int64
	firstSeen  time.Time
	lastSeen   time.Time
	suppressed int64
}

// FilterStats tracks filtering statistics
type FilterStats struct {
	TotalProcessed  int64            `json:"total_processed"`
	TotalPassed     int64            `json:"total_passed"`
	TotalFiltered   int64            `json:"total_filtered"`
	TotalSuppressed int64            `json:"total_suppressed"`
	FilterRate      float64          `json:"filter_rate"`
	ProcessingTime  time.Duration    `json:"processing_time"`
	LastUpdated     time.Time        `json:"last_updated"`
	FilterBreakdown map[string]int64 `json:"filter_breakdown"`
}

// FilterConfig configures the log filter
type FilterConfig struct {
	MaxEntriesPerSec    int           `json:"max_entries_per_sec"`
	MaxEntriesPerMin    int           `json:"max_entries_per_min"`
	MaxEntriesPerHour   int           `json:"max_entries_per_hour"`
	BloomFilterSize     int           `json:"bloom_filter_size"`
	DuplicateCacheSize  int           `json:"duplicate_cache_size"`
	DuplicateTTL        time.Duration `json:"duplicate_ttl"`
	EnableDeduplication bool          `json:"enable_deduplication"`
	MinPriority         int           `json:"min_priority"`
	ExcludePatterns     []string      `json:"exclude_patterns"`
	IncludePatterns     []string      `json:"include_patterns"`
	ExcludeUnits        []string      `json:"exclude_units"`
	IncludeUnits        []string      `json:"include_units"`
	EnableSampling      bool          `json:"enable_sampling"`
	SampleRate          float64       `json:"sample_rate"`
}

// NewLogFilter creates a new log filter with the specified configuration
func NewLogFilter(config FilterConfig) *LogFilter {
	// Set defaults
	if config.MaxEntriesPerSec == 0 {
		config.MaxEntriesPerSec = 10000
	}
	if config.MaxEntriesPerMin == 0 {
		config.MaxEntriesPerMin = 100000
	}
	if config.MaxEntriesPerHour == 0 {
		config.MaxEntriesPerHour = 1000000
	}
	if config.BloomFilterSize == 0 {
		config.BloomFilterSize = 1000000
	}
	if config.DuplicateCacheSize == 0 {
		config.DuplicateCacheSize = 10000
	}
	if config.DuplicateTTL == 0 {
		config.DuplicateTTL = 5 * time.Minute
	}
	if config.SampleRate == 0 {
		config.SampleRate = 1.0
	}

	filter := &LogFilter{
		filters:   make([]Filter, 0),
		rateLimit: NewRateLimit(config.MaxEntriesPerSec, config.MaxEntriesPerMin, config.MaxEntriesPerHour),
		stats: FilterStats{
			FilterBreakdown: make(map[string]int64),
		},
	}

	if config.EnableDeduplication {
		filter.bloomFilter = NewBloomFilter(config.BloomFilterSize)
		filter.duplicateCache = NewDuplicateCache(config.DuplicateCacheSize, config.DuplicateTTL)
	}

	// Add filters based on configuration
	if config.MinPriority > 0 {
		filter.AddFilter(NewPriorityFilter(config.MinPriority))
	}

	if len(config.ExcludePatterns) > 0 {
		filter.AddFilter(NewPatternFilter(config.ExcludePatterns, false))
	}

	if len(config.IncludePatterns) > 0 {
		filter.AddFilter(NewPatternFilter(config.IncludePatterns, true))
	}

	if len(config.ExcludeUnits) > 0 {
		filter.AddFilter(NewUnitFilter(config.ExcludeUnits, false))
	}

	if len(config.IncludeUnits) > 0 {
		filter.AddFilter(NewUnitFilter(config.IncludeUnits, true))
	}

	if config.EnableSampling && config.SampleRate < 1.0 {
		filter.AddFilter(NewSamplingFilter(config.SampleRate))
	}

	return filter
}

// NewRateLimit creates a new rate limiter
func NewRateLimit(perSec, perMin, perHour int) *RateLimit {
	return &RateLimit{
		maxPerSecond: perSec,
		maxPerMinute: perMin,
		maxPerHour:   perHour,
		buckets:      make(map[string]*TokenBucket),
		globalBucket: NewTokenBucket(perSec, perSec),
	}
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed based on rate limits
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Add tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
	tb.tokens += tokensToAdd
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// AllowEntry checks if an entry should be allowed through rate limiting
func (rl *RateLimit) AllowEntry(entry LogEntry) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check global rate limit first
	if !rl.globalBucket.Allow() {
		return false
	}

	// Check per-unit rate limit
	unitKey := entry.Unit
	if unitKey == "" {
		unitKey = "_global"
	}

	bucket, exists := rl.buckets[unitKey]
	if !exists {
		bucket = NewTokenBucket(rl.maxPerSecond/10, rl.maxPerSecond/10) // Per-unit limit
		rl.buckets[unitKey] = bucket
	}

	return bucket.Allow()
}

// NewBloomFilter creates a new bloom filter
func NewBloomFilter(size int) *BloomFilter {
	return &BloomFilter{
		bitArray:  make([]bool, size),
		size:      size,
		hashFuncs: []HashFunc{hash1, hash2, hash3},
		elements:  0,
	}
}

// Add adds an element to the bloom filter
func (bf *BloomFilter) Add(data []byte) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	for _, hashFunc := range bf.hashFuncs {
		index := hashFunc(data) % uint32(bf.size)
		bf.bitArray[index] = true
	}
	bf.elements++
}

// Contains checks if an element might be in the bloom filter
func (bf *BloomFilter) Contains(data []byte) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	for _, hashFunc := range bf.hashFuncs {
		index := hashFunc(data) % uint32(bf.size)
		if !bf.bitArray[index] {
			return false
		}
	}
	return true
}

// NewDuplicateCache creates a new duplicate cache
func NewDuplicateCache(maxSize int, ttl time.Duration) *DuplicateCache {
	return &DuplicateCache{
		cache:   make(map[string]*DuplicateEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// CheckDuplicate checks if an entry is a duplicate and updates tracking
func (dc *DuplicateCache) CheckDuplicate(entry LogEntry) (*DuplicateEntry, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Generate hash for the entry
	hash := dc.generateHash(entry)
	now := time.Now()

	// Clean expired entries
	dc.cleanExpired(now)

	if existing, exists := dc.cache[hash]; exists {
		existing.count++
		existing.lastSeen = now
		return existing, true
	}

	// Add new entry
	newEntry := &DuplicateEntry{
		hash:      hash,
		count:     1,
		firstSeen: now,
		lastSeen:  now,
	}

	// Evict if cache is full
	if len(dc.cache) >= dc.maxSize {
		dc.evictOldest()
	}

	dc.cache[hash] = newEntry
	return newEntry, false
}

// generateHash generates a hash for duplicate detection
func (dc *DuplicateCache) generateHash(entry LogEntry) string {
	// Normalize message by removing timestamps, PIDs, etc.
	normalized := normalizeForDuplicateDetection(entry.Message)
	return fmt.Sprintf("%s:%s:%d", entry.Unit, normalized, entry.Priority)
}

// cleanExpired removes expired entries from the cache
func (dc *DuplicateCache) cleanExpired(now time.Time) {
	for hash, entry := range dc.cache {
		if now.Sub(entry.lastSeen) > dc.ttl {
			delete(dc.cache, hash)
		}
	}
}

// evictOldest removes the oldest entry from the cache
func (dc *DuplicateCache) evictOldest() {
	var oldestHash string
	var oldestTime time.Time

	for hash, entry := range dc.cache {
		if oldestHash == "" || entry.firstSeen.Before(oldestTime) {
			oldestHash = hash
			oldestTime = entry.firstSeen
		}
	}

	if oldestHash != "" {
		delete(dc.cache, oldestHash)
	}
}

// AddFilter adds a new filter to the pipeline
func (lf *LogFilter) AddFilter(filter Filter) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	lf.filters = append(lf.filters, filter)
}

// ShouldProcess determines if a log entry should be processed
func (lf *LogFilter) ShouldProcess(entry LogEntry) bool {
	start := time.Now()
	defer func() {
		lf.mu.Lock()
		lf.stats.ProcessingTime += time.Since(start)
		lf.stats.TotalProcessed++
		lf.mu.Unlock()
	}()

	// Check rate limiting first
	if !lf.rateLimit.AllowEntry(entry) {
		lf.mu.Lock()
		lf.stats.TotalFiltered++
		lf.stats.FilterBreakdown["rate_limit"]++
		lf.mu.Unlock()
		return false
	}

	// Check for duplicates if enabled
	if lf.duplicateCache != nil {
		if dupEntry, isDup := lf.duplicateCache.CheckDuplicate(entry); isDup {
			// Suppress high-frequency duplicates
			if dupEntry.count > 10 && time.Since(dupEntry.firstSeen) < time.Minute {
				dupEntry.suppressed++
				lf.mu.Lock()
				lf.stats.TotalSuppressed++
				lf.stats.FilterBreakdown["duplicate"]++
				lf.mu.Unlock()
				return false
			}
		}

		// Add to bloom filter for fast duplicate checking
		if lf.bloomFilter != nil {
			msgBytes := []byte(entry.Message)
			if lf.bloomFilter.Contains(msgBytes) {
				// Might be duplicate, but bloom filter can have false positives
				// Already handled by duplicate cache above
			}
			lf.bloomFilter.Add(msgBytes)
		}
	}

	// Apply all filters
	lf.mu.RLock()
	filters := lf.filters
	lf.mu.RUnlock()

	for _, filter := range filters {
		if !filter.ShouldProcess(entry) {
			lf.mu.Lock()
			lf.stats.TotalFiltered++
			lf.stats.FilterBreakdown[filter.Name()]++
			lf.mu.Unlock()
			return false
		}
	}

	lf.mu.Lock()
	lf.stats.TotalPassed++
	lf.mu.Unlock()
	return true
}

// GetStats returns filtering statistics
func (lf *LogFilter) GetStats() FilterStats {
	lf.mu.RLock()
	defer lf.mu.RUnlock()

	stats := lf.stats
	if stats.TotalProcessed > 0 {
		stats.FilterRate = float64(stats.TotalFiltered) / float64(stats.TotalProcessed)
	}
	stats.LastUpdated = time.Now()

	return stats
}

// Specific filter implementations

// PriorityFilter filters by log priority
type PriorityFilter struct {
	minPriority int
	stats       FilterStats
}

func NewPriorityFilter(minPriority int) *PriorityFilter {
	return &PriorityFilter{
		minPriority: minPriority,
		stats:       FilterStats{FilterBreakdown: make(map[string]int64)},
	}
}

func (pf *PriorityFilter) ShouldProcess(entry LogEntry) bool {
	pf.stats.TotalProcessed++
	if entry.Priority > pf.minPriority {
		pf.stats.TotalFiltered++
		return false
	}
	pf.stats.TotalPassed++
	return true
}

func (pf *PriorityFilter) Name() string {
	return "priority"
}

func (pf *PriorityFilter) GetStats() FilterStats {
	return pf.stats
}

// PatternFilter filters by message patterns
type PatternFilter struct {
	patterns []regexp.Regexp
	include  bool
	stats    FilterStats
}

func NewPatternFilter(patterns []string, include bool) *PatternFilter {
	regexps := make([]regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		regexps[i] = *regexp.MustCompile(pattern)
	}

	return &PatternFilter{
		patterns: regexps,
		include:  include,
		stats:    FilterStats{FilterBreakdown: make(map[string]int64)},
	}
}

func (pf *PatternFilter) ShouldProcess(entry LogEntry) bool {
	pf.stats.TotalProcessed++

	for _, pattern := range pf.patterns {
		if pattern.MatchString(entry.Message) {
			if pf.include {
				pf.stats.TotalPassed++
				return true
			} else {
				pf.stats.TotalFiltered++
				return false
			}
		}
	}

	if pf.include {
		pf.stats.TotalFiltered++
		return false
	} else {
		pf.stats.TotalPassed++
		return true
	}
}

func (pf *PatternFilter) Name() string {
	if pf.include {
		return "pattern_include"
	}
	return "pattern_exclude"
}

func (pf *PatternFilter) GetStats() FilterStats {
	return pf.stats
}

// UnitFilter filters by systemd unit
type UnitFilter struct {
	units   map[string]bool
	include bool
	stats   FilterStats
}

func NewUnitFilter(units []string, include bool) *UnitFilter {
	unitMap := make(map[string]bool)
	for _, unit := range units {
		unitMap[unit] = true
	}

	return &UnitFilter{
		units:   unitMap,
		include: include,
		stats:   FilterStats{FilterBreakdown: make(map[string]int64)},
	}
}

func (uf *UnitFilter) ShouldProcess(entry LogEntry) bool {
	uf.stats.TotalProcessed++

	_, found := uf.units[entry.Unit]

	if uf.include {
		if found {
			uf.stats.TotalPassed++
			return true
		} else {
			uf.stats.TotalFiltered++
			return false
		}
	} else {
		if found {
			uf.stats.TotalFiltered++
			return false
		} else {
			uf.stats.TotalPassed++
			return true
		}
	}
}

func (uf *UnitFilter) Name() string {
	if uf.include {
		return "unit_include"
	}
	return "unit_exclude"
}

func (uf *UnitFilter) GetStats() FilterStats {
	return uf.stats
}

// SamplingFilter implements statistical sampling
type SamplingFilter struct {
	sampleRate float64
	counter    int64
	stats      FilterStats
	mu         sync.Mutex
}

func NewSamplingFilter(sampleRate float64) *SamplingFilter {
	return &SamplingFilter{
		sampleRate: sampleRate,
		stats:      FilterStats{FilterBreakdown: make(map[string]int64)},
	}
}

func (sf *SamplingFilter) ShouldProcess(entry LogEntry) bool {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.stats.TotalProcessed++
	sf.counter++

	// Simple deterministic sampling
	threshold := 1.0 / sf.sampleRate
	if float64(sf.counter) >= threshold {
		sf.counter = 0
		sf.stats.TotalPassed++
		return true
	}

	sf.stats.TotalFiltered++
	return false
}

func (sf *SamplingFilter) Name() string {
	return "sampling"
}

func (sf *SamplingFilter) GetStats() FilterStats {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return sf.stats
}

// Hash functions for bloom filter
func hash1(data []byte) uint32 {
	h := uint32(0)
	for _, b := range data {
		h = h*31 + uint32(b)
	}
	return h
}

func hash2(data []byte) uint32 {
	h := uint32(5381)
	for _, b := range data {
		h = ((h << 5) + h) + uint32(b)
	}
	return h
}

func hash3(data []byte) uint32 {
	h := uint32(0)
	for i, b := range data {
		h += uint32(b) * uint32(i+1)
	}
	return h
}

// normalizeForDuplicateDetection normalizes messages for duplicate detection
func normalizeForDuplicateDetection(message string) string {
	// Remove timestamps
	normalized := regexp.MustCompile(`\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}`).ReplaceAllString(message, "TIMESTAMP")

	// Remove PIDs
	normalized = regexp.MustCompile(`\[\d+\]`).ReplaceAllString(normalized, "[PID]")

	// Remove memory addresses
	normalized = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(normalized, "ADDR")

	// Remove numbers that might be variable
	normalized = regexp.MustCompile(`\b\d{4,}\b`).ReplaceAllString(normalized, "NUM")

	// Remove file descriptors
	normalized = regexp.MustCompile(`fd \d+`).ReplaceAllString(normalized, "fd NUM")

	// Normalize multiple spaces
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return strings.TrimSpace(normalized)
}
