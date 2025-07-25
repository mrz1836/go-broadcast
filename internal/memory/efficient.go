// Package memory provides efficient memory management utilities including string interning, lazy loading, and memory pressure monitoring.
package memory

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// StringIntern provides string interning for repeated values to reduce memory usage
// This is particularly useful for repository names, branch names, and other repeated identifiers
type StringIntern struct {
	mu     sync.RWMutex
	values map[string]string

	// Statistics for monitoring
	stats struct {
		hits    int64 // Cache hits (returned existing interned string)
		misses  int64 // Cache misses (created new interned string)
		evicted int64 // Strings evicted due to size limits
		size    int64 // Current cache size
	}

	maxSize int // Maximum number of strings to intern (0 = unlimited)
}

// DefaultStringInternSize is the default maximum cache size for string interning
const DefaultStringInternSize = 10000

// NewStringIntern creates a new string interning system
func NewStringIntern() *StringIntern {
	return NewStringInternWithSize(DefaultStringInternSize)
}

// NewStringInternWithSize creates a string interning system with a specific max size
func NewStringInternWithSize(maxSize int) *StringIntern {
	return &StringIntern{
		values:  make(map[string]string),
		maxSize: maxSize,
	}
}

// Intern returns a canonical instance of the string, reducing memory usage for duplicates
//
// This function is thread-safe and uses read-write locks for optimal performance.
// Common use cases include repository names, branch names, and file paths that appear frequently.
func (si *StringIntern) Intern(s string) string {
	// Fast path: check if string is already interned (read lock only)
	si.mu.RLock()
	if interned, ok := si.values[s]; ok {
		si.mu.RUnlock()
		atomic.AddInt64(&si.stats.hits, 1)
		return interned
	}
	si.mu.RUnlock()

	// Slow path: need to intern the string (write lock required)
	si.mu.Lock()
	defer si.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have added it)
	if interned, ok := si.values[s]; ok {
		atomic.AddInt64(&si.stats.hits, 1)
		return interned
	}

	// Check if we need to evict old entries due to size limits
	if si.maxSize > 0 && len(si.values) >= si.maxSize {
		si.evictOldest()
	}

	// Intern the string
	si.values[s] = s
	atomic.AddInt64(&si.stats.misses, 1)
	atomic.StoreInt64(&si.stats.size, int64(len(si.values)))

	return s
}

// GetStats returns current string interning statistics
func (si *StringIntern) GetStats() StringInternStats {
	si.mu.RLock()
	defer si.mu.RUnlock()

	return StringInternStats{
		Hits:    atomic.LoadInt64(&si.stats.hits),
		Misses:  atomic.LoadInt64(&si.stats.misses),
		Evicted: atomic.LoadInt64(&si.stats.evicted),
		Size:    atomic.LoadInt64(&si.stats.size),
		MaxSize: int64(si.maxSize),
	}
}

// evictOldest removes approximately 10% of entries to make room for new ones
// This is called when maxSize is reached
func (si *StringIntern) evictOldest() {
	evictCount := maxInt(1, len(si.values)/10) // Remove at least 1, up to 10%
	evicted := 0

	// Simple eviction strategy: remove entries in iteration order
	// This is not LRU but is fast and works well for most use cases
	for key := range si.values {
		if evicted >= evictCount {
			break
		}
		delete(si.values, key)
		evicted++
	}

	atomic.AddInt64(&si.stats.evicted, int64(evicted))
}

// StringInternStats contains string interning statistics
type StringInternStats struct {
	Hits    int64 `json:"hits"`     // Number of cache hits
	Misses  int64 `json:"misses"`   // Number of cache misses
	Evicted int64 `json:"evicted"`  // Number of evicted entries
	Size    int64 `json:"size"`     // Current cache size
	MaxSize int64 `json:"max_size"` // Maximum cache size (0 = unlimited)
}

// HitRate calculates the cache hit rate as a percentage
func (sis StringInternStats) HitRate() float64 {
	total := sis.Hits + sis.Misses
	if total == 0 {
		return 0
	}
	return float64(sis.Hits) / float64(total) * 100
}

// PreallocateSlice creates a slice with the right capacity to minimize allocations
//
// This function helps optimize memory usage by pre-allocating slices with appropriate
// capacity based on estimated size, reducing the number of grow operations.
//
// Parameters:
// - estimatedSize: Expected number of elements the slice will contain
//
// Returns:
// - Empty slice with capacity set to estimatedSize
//
// Usage Example:
//
//	repos := PreallocateSlice[string](100) // Expect ~100 repository names
//	for _, repo := range repositories {
//	    repos = append(repos, repo.Name)
//	}
func PreallocateSlice[T any](estimatedSize int) []T {
	if estimatedSize <= 0 {
		return nil
	}
	return make([]T, 0, estimatedSize)
}

// PreallocateMap creates a map with appropriate initial size to minimize allocations
//
// Go maps benefit from having their size hint set during creation to avoid
// multiple rehashing operations as they grow.
//
// Parameters:
// - estimatedSize: Expected number of key-value pairs the map will contain
//
// Returns:
// - Empty map with size hint set to estimatedSize
//
// Usage Example:
//
//	branchMap := PreallocateMap[string, BranchInfo](50) // Expect ~50 branches
//	for _, branch := range branches {
//	    branchMap[branch.Name] = branch
//	}
func PreallocateMap[K comparable, V any](estimatedSize int) map[K]V {
	if estimatedSize <= 0 {
		return make(map[K]V)
	}
	return make(map[K]V, estimatedSize)
}

// ReuseSlice clears a slice and returns it for reuse, preserving capacity
//
// This function helps reduce garbage collection pressure by reusing existing
// slice backing arrays instead of allocating new ones.
//
// Parameters:
// - slice: Slice to clear and reuse
//
// Returns:
// - The same slice with length 0 but original capacity preserved
//
// Usage Example:
//
//	var fileList []string
//	for batch := range batches {
//	    fileList = ReuseSlice(fileList)
//	    // ... populate fileList ...
//	}
func ReuseSlice[T any](slice []T) []T {
	return slice[:0]
}

// PressureMonitor tracks memory usage and can trigger alerts when thresholds are exceeded
type PressureMonitor struct {
	mu                 sync.RWMutex
	thresholds         Thresholds
	lastStats          runtime.MemStats
	alertCallback      func(Alert)
	monitoringEnabled  bool
	monitoringInterval time.Duration
	stopChan           chan struct{}

	// Statistics
	alertCount         int64
	highPressureEvents int64
	gcForcedCount      int64
}

// Thresholds defines memory usage thresholds for monitoring
type Thresholds struct {
	HeapAllocMB   uint64 // Alert when heap allocation exceeds this (MB)
	HeapSysMB     uint64 // Alert when heap system memory exceeds this (MB)
	GCPercent     uint64 // Alert when GC percentage exceeds this
	NumGoroutines int    // Alert when goroutine count exceeds this
}

// DefaultThresholds returns sensible defaults for most applications
func DefaultThresholds() Thresholds {
	return Thresholds{
		HeapAllocMB:   100,  // 100MB heap allocation
		HeapSysMB:     200,  // 200MB heap system memory
		GCPercent:     10,   // 10% time spent in GC
		NumGoroutines: 1000, // 1000 goroutines
	}
}

// Alert represents a memory pressure alert
type Alert struct {
	Type      string           `json:"type"`
	Message   string           `json:"message"`
	Timestamp time.Time        `json:"timestamp"`
	Stats     runtime.MemStats `json:"stats"`
	Severity  AlertSeverity    `json:"severity"`
}

// AlertSeverity indicates the severity of a memory alert
type AlertSeverity int

const (
	// AlertInfo indicates an informational memory alert
	AlertInfo AlertSeverity = iota
	// AlertWarning indicates a warning-level memory alert
	AlertWarning
	// AlertCritical indicates a critical memory alert
	AlertCritical
)

// String returns the string representation of AlertSeverity
func (mas AlertSeverity) String() string {
	switch mas {
	case AlertInfo:
		return "INFO"
	case AlertWarning:
		return "WARNING"
	case AlertCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// NewPressureMonitor creates a new memory pressure monitor
func NewPressureMonitor(thresholds Thresholds, alertCallback func(Alert)) *PressureMonitor {
	return &PressureMonitor{
		thresholds:         thresholds,
		alertCallback:      alertCallback,
		monitoringInterval: 30 * time.Second, // Default monitoring interval
		stopChan:           make(chan struct{}),
	}
}

// StartMonitoring begins continuous memory monitoring
func (mpm *PressureMonitor) StartMonitoring() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if mpm.monitoringEnabled {
		return // Already monitoring
	}

	mpm.monitoringEnabled = true
	go mpm.monitorLoop()
}

// StopMonitoring stops continuous memory monitoring
func (mpm *PressureMonitor) StopMonitoring() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if !mpm.monitoringEnabled {
		return // Not monitoring
	}

	mpm.monitoringEnabled = false
	close(mpm.stopChan)
	mpm.stopChan = make(chan struct{}) // Reset for potential restart
}

// ForceGC forces garbage collection when memory pressure is high
func (mpm *PressureMonitor) ForceGC() {
	runtime.GC()
	atomic.AddInt64(&mpm.gcForcedCount, 1)
}

// GetCurrentMemStats returns the current memory statistics
func (mpm *PressureMonitor) GetCurrentMemStats() runtime.MemStats {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	return mpm.lastStats
}

// GetMonitorStats returns monitoring statistics
func (mpm *PressureMonitor) GetMonitorStats() MonitorStats {
	return MonitorStats{
		AlertCount:         atomic.LoadInt64(&mpm.alertCount),
		HighPressureEvents: atomic.LoadInt64(&mpm.highPressureEvents),
		GCForcedCount:      atomic.LoadInt64(&mpm.gcForcedCount),
		MonitoringEnabled:  mpm.monitoringEnabled,
	}
}

// monitorLoop runs the continuous monitoring process
func (mpm *PressureMonitor) monitorLoop() {
	ticker := time.NewTicker(mpm.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mpm.stopChan:
			return
		case <-ticker.C:
			mpm.checkMemoryPressure()
		}
	}
}

// checkMemoryPressure examines current memory usage and triggers alerts if needed
func (mpm *PressureMonitor) checkMemoryPressure() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	mpm.mu.Lock()
	mpm.lastStats = stats
	mpm.mu.Unlock()

	// Check heap allocation
	heapAllocMB := stats.Alloc / 1024 / 1024
	if heapAllocMB > mpm.thresholds.HeapAllocMB {
		alert := Alert{
			Type:      "heap_alloc",
			Message:   fmt.Sprintf("Heap allocation (%d MB) exceeds threshold (%d MB)", heapAllocMB, mpm.thresholds.HeapAllocMB),
			Timestamp: time.Now(),
			Stats:     stats,
			Severity:  mpm.calculateSeverity("heap_alloc", float64(heapAllocMB), float64(mpm.thresholds.HeapAllocMB)),
		}
		mpm.sendAlert(alert)
	}

	// Check heap system memory
	heapSysMB := stats.HeapSys / 1024 / 1024
	if heapSysMB > mpm.thresholds.HeapSysMB {
		alert := Alert{
			Type:      "heap_sys",
			Message:   fmt.Sprintf("Heap system memory (%d MB) exceeds threshold (%d MB)", heapSysMB, mpm.thresholds.HeapSysMB),
			Timestamp: time.Now(),
			Stats:     stats,
			Severity:  mpm.calculateSeverity("heap_sys", float64(heapSysMB), float64(mpm.thresholds.HeapSysMB)),
		}
		mpm.sendAlert(alert)
	}

	// Check goroutine count
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > mpm.thresholds.NumGoroutines {
		alert := Alert{
			Type:      "goroutines",
			Message:   fmt.Sprintf("Goroutine count (%d) exceeds threshold (%d)", numGoroutines, mpm.thresholds.NumGoroutines),
			Timestamp: time.Now(),
			Stats:     stats,
			Severity:  mpm.calculateSeverity("goroutines", float64(numGoroutines), float64(mpm.thresholds.NumGoroutines)),
		}
		mpm.sendAlert(alert)
	}
}

// calculateSeverity determines alert severity based on how much the threshold is exceeded
func (mpm *PressureMonitor) calculateSeverity(_ string, actual, threshold float64) AlertSeverity {
	ratio := actual / threshold

	switch {
	case ratio >= 2.0: // 200% of threshold
		return AlertCritical
	case ratio >= 1.5: // 150% of threshold
		return AlertWarning
	default:
		return AlertInfo
	}
}

// sendAlert sends an alert through the callback if configured
func (mpm *PressureMonitor) sendAlert(alert Alert) {
	atomic.AddInt64(&mpm.alertCount, 1)

	if alert.Severity >= AlertWarning {
		atomic.AddInt64(&mpm.highPressureEvents, 1)
	}

	if mpm.alertCallback != nil {
		// Call alert callback in a separate goroutine to avoid blocking monitoring
		go mpm.alertCallback(alert)
	}
}

// MonitorStats contains memory monitoring statistics
type MonitorStats struct {
	AlertCount         int64 `json:"alert_count"`
	HighPressureEvents int64 `json:"high_pressure_events"`
	GCForcedCount      int64 `json:"gc_forced_count"`
	MonitoringEnabled  bool  `json:"monitoring_enabled"`
}

// LazyLoader provides on-demand loading for expensive resources
type LazyLoader[T any] struct {
	mu        sync.RWMutex
	loader    func() (T, error)
	value     T
	loaded    bool
	err       error
	loadCount int64
}

// NewLazyLoader creates a new lazy loader with the given loading function
func NewLazyLoader[T any](loader func() (T, error)) *LazyLoader[T] {
	return &LazyLoader[T]{
		loader: loader,
	}
}

// Get returns the loaded value, loading it if necessary
func (ll *LazyLoader[T]) Get() (T, error) {
	// Fast path: check if already loaded (read lock only)
	ll.mu.RLock()
	if ll.loaded {
		defer ll.mu.RUnlock()
		return ll.value, ll.err
	}
	ll.mu.RUnlock()

	// Slow path: need to load the value (write lock required)
	ll.mu.Lock()
	defer ll.mu.Unlock()

	// Double-check after acquiring write lock
	if ll.loaded {
		return ll.value, ll.err
	}

	// Load the value
	ll.value, ll.err = ll.loader()
	ll.loaded = true
	atomic.AddInt64(&ll.loadCount, 1)

	return ll.value, ll.err
}

// Reset clears the loaded value, forcing it to be reloaded on next access
func (ll *LazyLoader[T]) Reset() {
	ll.mu.Lock()
	defer ll.mu.Unlock()

	var zero T
	ll.value = zero
	ll.err = nil
	ll.loaded = false
}

// IsLoaded returns true if the value has been loaded
func (ll *LazyLoader[T]) IsLoaded() bool {
	ll.mu.RLock()
	defer ll.mu.RUnlock()

	return ll.loaded
}

// GetLoadCount returns the number of times the loader function has been called
func (ll *LazyLoader[T]) GetLoadCount() int64 {
	return atomic.LoadInt64(&ll.loadCount)
}

// Helper function for max calculation
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
