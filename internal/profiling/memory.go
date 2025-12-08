// Package profiling provides comprehensive memory and performance profiling capabilities for the broadcast system.
package profiling

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"
)

// Profiling errors
var (
	ErrProfilerNotEnabled    = errors.New("profiler is not enabled")
	ErrSessionExists         = errors.New("profiling session already exists")
	ErrSessionNotFound       = errors.New("profiling session not found")
	ErrSessionAlreadyStarted = errors.New("session already started")
	ErrProfileNotFound       = errors.New("profile not found")
)

// MemoryProfiler provides comprehensive memory profiling capabilities
// It captures heap profiles, allocation traces, and runtime statistics
type MemoryProfiler struct {
	outputDir string
	enabled   bool
	mu        sync.RWMutex

	// Active profiling sessions
	sessions map[string]*Session

	// Statistics
	profileCount  int64
	totalSessions int64
}

// Session represents an active profiling session
type Session struct {
	Name      string
	StartTime time.Time
	OutputDir string

	// Profile files
	cpuFile   *os.File
	heapFile  string
	traceFile *os.File

	// Session state
	started bool
	stopped bool
	mu      sync.Mutex
}

// NewMemoryProfiler creates a new memory profiler with the specified output directory
func NewMemoryProfiler(outputDir string) *MemoryProfiler {
	return &MemoryProfiler{
		outputDir: outputDir,
		sessions:  make(map[string]*Session),
	}
}

// Enable enables the memory profiler and creates the output directory
func (mp *MemoryProfiler) Enable() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.enabled {
		return nil
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(mp.outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create profile output directory: %w", err)
	}

	mp.enabled = true
	return nil
}

// Disable disables the memory profiler and stops all active sessions
func (mp *MemoryProfiler) Disable() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if !mp.enabled {
		return nil
	}

	// Stop all active sessions
	for _, session := range mp.sessions {
		if err := mp.stopSession(session); err != nil {
			// Log error but continue stopping other sessions
			log.Printf("Warning: failed to stop profiling session %s: %v\n", session.Name, err)
		}
	}

	mp.enabled = false
	mp.sessions = make(map[string]*Session)

	return nil
}

// StartProfiling begins a comprehensive profiling session
func (mp *MemoryProfiler) StartProfiling(name string) (*Session, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if !mp.enabled {
		return nil, ErrProfilerNotEnabled
	}

	// Check if session already exists
	if _, exists := mp.sessions[name]; exists {
		return nil, fmt.Errorf("%w: %s", ErrSessionExists, name)
	}

	// Create session
	sessionDir := filepath.Join(mp.outputDir, name)
	if err := os.MkdirAll(sessionDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	session := &Session{
		Name:      name,
		StartTime: time.Now(),
		OutputDir: sessionDir,
	}

	// Start profiling
	if err := mp.startSession(session); err != nil {
		return nil, fmt.Errorf("failed to start profiling session: %w", err)
	}

	mp.sessions[name] = session
	mp.totalSessions++

	return session, nil
}

// StopProfiling stops the specified profiling session
func (mp *MemoryProfiler) StopProfiling(name string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	session, exists := mp.sessions[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, name)
	}

	if err := mp.stopSession(session); err != nil {
		return err
	}

	delete(mp.sessions, name)
	mp.profileCount++

	return nil
}

// startSession initiates all profiling types for a session
func (mp *MemoryProfiler) startSession(session *Session) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.started {
		return ErrSessionAlreadyStarted
	}

	// Start CPU profiling
	cpuFile := filepath.Join(session.OutputDir, "cpu.prof")
	cpu, err := os.Create(cpuFile) //nolint:gosec // Creating profile output file
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}
	session.cpuFile = cpu

	if startErr := pprof.StartCPUProfile(cpu); startErr != nil {
		_ = cpu.Close() // Ignore error during cleanup
		return fmt.Errorf("failed to start CPU profiling: %w", startErr)
	}

	// Prepare heap profile path
	session.heapFile = filepath.Join(session.OutputDir, "heap.prof")

	// Start execution trace
	traceFile := filepath.Join(session.OutputDir, "trace.out")
	traceF, err := os.Create(traceFile) //nolint:gosec // Creating profile output file
	if err != nil {
		pprof.StopCPUProfile()
		_ = cpu.Close() // Ignore error during cleanup
		return fmt.Errorf("failed to create trace file: %w", err)
	}
	session.traceFile = traceF

	if err := trace.Start(traceF); err != nil {
		pprof.StopCPUProfile()
		_ = cpu.Close()    // Ignore error during cleanup
		_ = traceF.Close() // Ignore error during cleanup
		return fmt.Errorf("failed to start execution trace: %w", err)
	}

	session.started = true
	return nil
}

// stopSession stops all profiling for a session and generates reports
func (mp *MemoryProfiler) stopSession(session *Session) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.started || session.stopped {
		return nil
	}

	// Stop CPU profiling
	pprof.StopCPUProfile()
	if session.cpuFile != nil {
		_ = session.cpuFile.Close() // Ignore error during cleanup
	}

	// Stop execution trace
	trace.Stop()
	if session.traceFile != nil {
		_ = session.traceFile.Close() // Ignore error during cleanup
	}

	// Capture heap profile
	if err := mp.captureHeapProfile(session.heapFile); err != nil {
		return fmt.Errorf("failed to capture heap profile: %w", err)
	}

	// Capture additional profiles
	mp.captureAdditionalProfiles(session.OutputDir)

	// Generate analysis report
	if err := mp.generateAnalysisReport(session); err != nil {
		return fmt.Errorf("failed to generate analysis report: %w", err)
	}

	session.stopped = true
	return nil
}

// captureHeapProfile captures a heap profile
func (mp *MemoryProfiler) captureHeapProfile(filename string) error {
	// Force garbage collection to get accurate heap state
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup

	f, err := os.Create(filename) //nolint:gosec // Creating profile output file
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Ignore error in defer

	return pprof.WriteHeapProfile(f)
}

// captureAdditionalProfiles captures goroutine, mutex, and block profiles
func (mp *MemoryProfiler) captureAdditionalProfiles(outputDir string) {
	profiles := []struct {
		name     string
		filename string
		debug    int
	}{
		{"goroutine", "goroutine.prof", 1},
		{"mutex", "mutex.prof", 1},
		{"block", "block.prof", 1},
		{"allocs", "allocs.prof", 0},
		{"threadcreate", "threadcreate.prof", 1},
	}

	for _, profile := range profiles {
		if err := mp.captureProfile(profile.name, filepath.Join(outputDir, profile.filename), profile.debug); err != nil {
			// Log warning but continue with other profiles
			log.Printf("Warning: failed to capture %s profile: %v\n", profile.name, err)
		}
	}
}

// captureProfile captures a specific pprof profile
func (mp *MemoryProfiler) captureProfile(name, filename string, debug int) error {
	profile := pprof.Lookup(name)
	if profile == nil {
		return fmt.Errorf("%w: %s", ErrProfileNotFound, name)
	}

	f, err := os.Create(filename) //nolint:gosec // Creating profile output file
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Ignore error in defer

	return profile.WriteTo(f, debug)
}

// writeToReport is a helper function that safely writes to report files, ignoring errors
func writeToReport(writer io.Writer, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(writer, format, args...)
}

// generateAnalysisReport creates a human-readable analysis report
func (mp *MemoryProfiler) generateAnalysisReport(session *Session) error {
	reportFile := filepath.Join(session.OutputDir, "analysis_report.txt")

	f, err := os.Create(reportFile) //nolint:gosec // Creating report output file
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Ignore error in defer

	writer := bufio.NewWriter(f)
	defer func() { _ = writer.Flush() }() // Ignore error in defer

	// Write report header
	_, _ = fmt.Fprintf(writer, "Memory Profiling Analysis Report\n")
	_, _ = fmt.Fprintf(writer, "================================\n\n")
	_, _ = fmt.Fprintf(writer, "Session: %s\n", session.Name)
	_, _ = fmt.Fprintf(writer, "Start Time: %s\n", session.StartTime.Format(time.RFC3339))
	_, _ = fmt.Fprintf(writer, "Duration: %s\n", time.Since(session.StartTime))
	_, _ = fmt.Fprintf(writer, "Output Directory: %s\n\n", session.OutputDir)

	// Memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	_, _ = fmt.Fprintf(writer, "Memory Statistics\n")
	_, _ = fmt.Fprintf(writer, "-----------------\n")
	_, _ = fmt.Fprintf(writer, "Heap Alloc: %s\n", formatBytes(memStats.Alloc))
	_, _ = fmt.Fprintf(writer, "Heap Total: %s\n", formatBytes(memStats.TotalAlloc))
	_, _ = fmt.Fprintf(writer, "Heap Sys: %s\n", formatBytes(memStats.HeapSys))
	_, _ = fmt.Fprintf(writer, "Heap Objects: %d\n", memStats.HeapObjects)
	_, _ = fmt.Fprintf(writer, "Stack Sys: %s\n", formatBytes(memStats.StackSys))
	_, _ = fmt.Fprintf(writer, "MSpan Sys: %s\n", formatBytes(memStats.MSpanSys))
	_, _ = fmt.Fprintf(writer, "MCache Sys: %s\n", formatBytes(memStats.MCacheSys))
	_, _ = fmt.Fprintf(writer, "GC Sys: %s\n", formatBytes(memStats.GCSys))
	_, _ = fmt.Fprintf(writer, "Other Sys: %s\n", formatBytes(memStats.OtherSys))
	_, _ = fmt.Fprintf(writer, "Next GC: %s\n", formatBytes(memStats.NextGC))
	_, _ = fmt.Fprintf(writer, "Num GC: %d\n", memStats.NumGC)
	_, _ = fmt.Fprintf(writer, "Num Forced GC: %d\n", memStats.NumForcedGC)

	if memStats.NumGC > 0 {
		_, _ = fmt.Fprintf(writer, "Last GC: %s ago\n", time.Since(time.Unix(0, int64(memStats.LastGC)))) //nolint:gosec // LastGC is nanoseconds since epoch, safe for int64 until year 2262
		avgPause := float64(memStats.PauseTotalNs) / float64(memStats.NumGC) / 1e6
		_, _ = fmt.Fprintf(writer, "Average GC Pause: %.2f ms\n", avgPause)
	}

	writeToReport(writer, "\nRuntime Statistics\n")
	writeToReport(writer, "------------------\n")
	writeToReport(writer, "Goroutines: %d\n", runtime.NumGoroutine())
	writeToReport(writer, "CPUs: %d\n", runtime.NumCPU())
	writeToReport(writer, "GOMAXPROCS: %d\n", runtime.GOMAXPROCS(-1))
	writeToReport(writer, "Go Version: %s\n", runtime.Version())
	writeToReport(writer, "Compiler: %s\n", runtime.Compiler)
	writeToReport(writer, "Architecture: %s\n", runtime.GOARCH)
	writeToReport(writer, "OS: %s\n", runtime.GOOS)

	// File information
	writeToReport(writer, "\nGenerated Profile Files\n")
	writeToReport(writer, "-----------------------\n")
	profileFiles := []string{
		"cpu.prof",
		"heap.prof",
		"trace.out",
		"goroutine.prof",
		"mutex.prof",
		"block.prof",
		"allocs.prof",
		"threadcreate.prof",
	}

	for _, filename := range profileFiles {
		filePath := filepath.Join(session.OutputDir, filename)
		if info, err := os.Stat(filePath); err == nil {
			writeToReport(writer, "%s: %s (%s)\n", filename, formatBytes(uint64(max(0, info.Size()))), info.ModTime().Format(time.RFC3339)) //nolint:gosec // file sizes are non-negative and bounded
		}
	}

	// Analysis commands
	writeToReport(writer, "\nAnalysis Commands\n")
	writeToReport(writer, "-----------------\n")
	writeToReport(writer, "CPU Profile Top Functions:\n")
	writeToReport(writer, "  go tool pprof -top %s\n", filepath.Join(session.OutputDir, "cpu.prof"))
	writeToReport(writer, "\nHeap Profile Top Allocations:\n")
	writeToReport(writer, "  go tool pprof -top %s\n", filepath.Join(session.OutputDir, "heap.prof"))
	writeToReport(writer, "\nInteractive CPU Analysis:\n")
	writeToReport(writer, "  go tool pprof %s\n", filepath.Join(session.OutputDir, "cpu.prof"))
	writeToReport(writer, "\nInteractive Heap Analysis:\n")
	writeToReport(writer, "  go tool pprof %s\n", filepath.Join(session.OutputDir, "heap.prof"))
	writeToReport(writer, "\nExecution Trace Analysis:\n")
	writeToReport(writer, "  go tool trace %s\n", filepath.Join(session.OutputDir, "trace.out"))

	return nil
}

// CaptureMemStats captures current memory statistics
func CaptureMemStats(label string) MemorySnapshot {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return MemorySnapshot{
		Label:      label,
		Timestamp:  time.Now(),
		MemStats:   stats,
		Goroutines: runtime.NumGoroutine(),
	}
}

// MemorySnapshot represents memory state at a point in time
type MemorySnapshot struct {
	Label      string           `json:"label"`
	Timestamp  time.Time        `json:"timestamp"`
	MemStats   runtime.MemStats `json:"mem_stats"`
	Goroutines int              `json:"goroutines"`
}

// Compare compares two memory snapshots and returns the differences
func (ms MemorySnapshot) Compare(other MemorySnapshot) MemoryComparison {
	return MemoryComparison{
		From:            ms,
		To:              other,
		Duration:        other.Timestamp.Sub(ms.Timestamp),
		AllocDelta:      int64(other.MemStats.Alloc) - int64(ms.MemStats.Alloc),           //nolint:gosec // memory alloc values fit in int64 (max ~9 exabytes)
		TotalAllocDelta: int64(other.MemStats.TotalAlloc) - int64(ms.MemStats.TotalAlloc), //nolint:gosec // memory values fit in int64
		HeapSysDelta:    int64(other.MemStats.HeapSys) - int64(ms.MemStats.HeapSys),       //nolint:gosec // memory values fit in int64
		GCDelta:         int64(other.MemStats.NumGC) - int64(ms.MemStats.NumGC),
		GoroutineDelta:  other.Goroutines - ms.Goroutines,
	}
}

// MemoryComparison represents the difference between two memory snapshots
type MemoryComparison struct {
	From            MemorySnapshot `json:"from"`
	To              MemorySnapshot `json:"to"`
	Duration        time.Duration  `json:"duration"`
	AllocDelta      int64          `json:"alloc_delta"`
	TotalAllocDelta int64          `json:"total_alloc_delta"`
	HeapSysDelta    int64          `json:"heap_sys_delta"`
	GCDelta         int64          `json:"gc_delta"`
	GoroutineDelta  int            `json:"goroutine_delta"`
}

// String returns a human-readable representation of the memory comparison
func (mc MemoryComparison) String() string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Memory Comparison: %s → %s (%.2fs)\n",
		mc.From.Label, mc.To.Label, mc.Duration.Seconds()))
	result.WriteString(fmt.Sprintf("  Heap Alloc: %+s\n", FormatBytesDelta(mc.AllocDelta)))
	result.WriteString(fmt.Sprintf("  Total Alloc: %+s\n", FormatBytesDelta(mc.TotalAllocDelta)))
	result.WriteString(fmt.Sprintf("  Heap Sys: %+s\n", FormatBytesDelta(mc.HeapSysDelta)))
	result.WriteString(fmt.Sprintf("  GC Count: %+d\n", mc.GCDelta))
	result.WriteString(fmt.Sprintf("  Goroutines: %+d\n", mc.GoroutineDelta))

	return result.String()
}

// ProfileWithContext runs a function while profiling and returns memory comparison
func ProfileWithContext(_ context.Context, profiler *MemoryProfiler, name string, fn func() error) (MemoryComparison, error) {
	// Capture initial state
	startSnapshot := CaptureMemStats(fmt.Sprintf("%s_start", name))

	// Start profiling if profiler is provided
	if profiler != nil {
		_, err := profiler.StartProfiling(name)
		if err != nil {
			return MemoryComparison{}, fmt.Errorf("failed to start profiling: %w", err)
		}
		defer func() {
			if stopErr := profiler.StopProfiling(name); stopErr != nil {
				log.Printf("Warning: failed to stop profiling: %v\n", stopErr)
			}
		}()
	}

	// Run the function
	fnErr := fn()

	// Capture final state
	endSnapshot := CaptureMemStats(fmt.Sprintf("%s_end", name))

	// Return comparison and function error
	comparison := startSnapshot.Compare(endSnapshot)
	return comparison, fnErr
}

// GetProfilerStats returns statistics about the profiler usage
func (mp *MemoryProfiler) GetProfilerStats() ProfilerStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return ProfilerStats{
		Enabled:        mp.enabled,
		OutputDir:      mp.outputDir,
		ActiveSessions: len(mp.sessions),
		TotalSessions:  mp.totalSessions,
		ProfileCount:   mp.profileCount,
	}
}

// ProfilerStats contains profiler usage statistics
type ProfilerStats struct {
	Enabled        bool   `json:"enabled"`
	OutputDir      string `json:"output_dir"`
	ActiveSessions int    `json:"active_sessions"`
	TotalSessions  int64  `json:"total_sessions"`
	ProfileCount   int64  `json:"profile_count"`
}

// formatBytes formats a byte count as a human-readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatBytesDelta formats a byte delta with appropriate sign and units
func FormatBytesDelta(delta int64) string {
	if delta == 0 {
		return "0 B"
	}

	sign := ""
	if delta > 0 {
		sign = "+"
	}

	absBytes := uint64(delta)
	if delta < 0 {
		absBytes = uint64(-delta)
		sign = "-"
	}

	return sign + formatBytes(absBytes)
}
