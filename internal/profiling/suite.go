package profiling

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"
)

// Suite errors
var (
	ErrProfilingSessionActive = errors.New("profiling session already active")
	ErrNoActiveSession        = errors.New("no active profiling session")
)

// ProfileSuite manages comprehensive profiling across multiple dimensions
type ProfileSuite struct {
	outputDir   string
	memProfiler *MemoryProfiler
	mu          sync.RWMutex

	// Active profiling session
	currentSession *ComprehensiveSession

	// Configuration
	config ProfileConfig

	// Session history
	sessionHistory []SessionSummary
}

// ComprehensiveSession represents a comprehensive profiling session
type ComprehensiveSession struct {
	Name      string
	StartTime time.Time
	OutputDir string

	// Profile files
	cpuFile    *os.File
	memSession *Session
	traceFile  *os.File

	// Additional profiles
	goroutineFile string
	blockFile     string
	mutexFile     string
	allocsFile    string

	// Session state
	started bool
	stopped bool
	mu      sync.Mutex

	// Performance metrics
	startSnapshot MemorySnapshot
	endSnapshot   MemorySnapshot
}

// ProfileConfig contains configuration for profiling sessions
type ProfileConfig struct {
	// Profiling types to enable
	EnableCPU    bool
	EnableMemory bool
	EnableTrace  bool
	EnableBlock  bool
	EnableMutex  bool

	// Block profiling rate (0 to disable)
	BlockProfileRate int

	// Mutex profiling fraction (0 to disable)
	MutexProfileFraction int

	// Report generation
	GenerateReports bool
	ReportFormat    string // "text", "html", "both"

	// Cleanup
	AutoCleanup       bool
	MaxSessionsToKeep int
}

// SessionSummary contains summary information about a completed session
type SessionSummary struct {
	Name         string        `json:"name"`
	StartTime    time.Time     `json:"start_time"`
	Duration     time.Duration `json:"duration"`
	OutputDir    string        `json:"output_dir"`
	ProfileTypes []string      `json:"profile_types"`
	FileCount    int           `json:"file_count"`
	TotalSize    int64         `json:"total_size_bytes"`
}

// NewProfileSuite creates a new comprehensive profiling suite
func NewProfileSuite(outputDir string) *ProfileSuite {
	return &ProfileSuite{
		outputDir:   outputDir,
		memProfiler: NewMemoryProfiler(filepath.Join(outputDir, "memory")),
		config: ProfileConfig{
			EnableCPU:            true,
			EnableMemory:         true,
			EnableTrace:          true,
			EnableBlock:          true,
			EnableMutex:          true,
			BlockProfileRate:     1,
			MutexProfileFraction: 1,
			GenerateReports:      true,
			ReportFormat:         "both",
			AutoCleanup:          true,
			MaxSessionsToKeep:    10,
		},
		sessionHistory: make([]SessionSummary, 0),
	}
}

// Configure updates the profiling configuration
func (ps *ProfileSuite) Configure(config ProfileConfig) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.config = config
}

// StartProfiling begins a comprehensive profiling session
func (ps *ProfileSuite) StartProfiling(name string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.currentSession != nil {
		return fmt.Errorf("%w: %s", ErrProfilingSessionActive, ps.currentSession.Name)
	}

	// Create output directory
	sessionDir := filepath.Join(ps.outputDir, fmt.Sprintf("%s_%s", name, time.Now().Format("20060102_150405")))
	if err := os.MkdirAll(sessionDir, 0o750); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	session := &ComprehensiveSession{
		Name:      name,
		StartTime: time.Now(),
		OutputDir: sessionDir,
	}

	// Enable profiling for the session
	if err := ps.enableProfiling(); err != nil {
		return fmt.Errorf("failed to enable profiling: %w", err)
	}

	// Start profiling components
	if err := ps.startSession(session); err != nil {
		return fmt.Errorf("failed to start profiling session: %w", err)
	}

	ps.currentSession = session
	return nil
}

// StopProfiling stops the current profiling session and generates reports
func (ps *ProfileSuite) StopProfiling() error {
	return ps.stopProfilingWithContext(context.Background())
}

// stopProfilingWithContext stops the current profiling session and generates reports with context
func (ps *ProfileSuite) stopProfilingWithContext(ctx context.Context) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.currentSession == nil {
		return ErrNoActiveSession
	}

	session := ps.currentSession

	// Stop profiling components
	if err := ps.stopSession(session); err != nil {
		return fmt.Errorf("failed to stop profiling session: %w", err)
	}

	// Generate reports if enabled
	if ps.config.GenerateReports {
		if err := ps.generateReports(ctx, session); err != nil {
			return fmt.Errorf("failed to generate reports: %w", err)
		}
	}

	// Add to session history
	summary := ps.createSessionSummary(session)
	ps.sessionHistory = append(ps.sessionHistory, summary)

	// Cleanup old sessions if enabled
	if ps.config.AutoCleanup {
		ps.cleanupOldSessions()
	}

	ps.currentSession = nil
	return nil
}

// ProfileWithFunc runs a function while profiling it comprehensively
func (ps *ProfileSuite) ProfileWithFunc(name string, fn func() error) error {
	if err := ps.StartProfiling(name); err != nil {
		return err
	}

	defer func() {
		if stopErr := ps.StopProfiling(); stopErr != nil {
			log.Printf("Warning: failed to stop profiling: %v\n", stopErr)
		}
	}()

	return fn()
}

// ProfileWithContext runs a function with context while profiling
func (ps *ProfileSuite) ProfileWithContext(ctx context.Context, name string, fn func(context.Context) error) error {
	if err := ps.StartProfiling(name); err != nil {
		return err
	}

	defer func() {
		if stopErr := ps.stopProfilingWithContext(ctx); stopErr != nil {
			log.Printf("Warning: failed to stop profiling: %v\n", stopErr)
		}
	}()

	return fn(ctx)
}

// enableProfiling enables the required profiling types
func (ps *ProfileSuite) enableProfiling() error {
	// Enable memory profiler
	if ps.config.EnableMemory {
		if err := ps.memProfiler.Enable(); err != nil {
			return fmt.Errorf("failed to enable memory profiler: %w", err)
		}
	}

	// Enable block profiling
	if ps.config.EnableBlock && ps.config.BlockProfileRate > 0 {
		runtime.SetBlockProfileRate(ps.config.BlockProfileRate)
	}

	// Enable mutex profiling
	if ps.config.EnableMutex && ps.config.MutexProfileFraction > 0 {
		runtime.SetMutexProfileFraction(ps.config.MutexProfileFraction)
	}

	return nil
}

// startSession starts all enabled profiling types for a session
func (ps *ProfileSuite) startSession(session *ComprehensiveSession) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.started {
		return ErrSessionAlreadyStarted
	}

	// Capture initial memory snapshot
	session.startSnapshot = CaptureMemStats(fmt.Sprintf("%s_start", session.Name))

	// Start CPU profiling
	if ps.config.EnableCPU {
		cpuFile := filepath.Join(session.OutputDir, "cpu.prof")
		cpu, err := os.Create(cpuFile) //nolint:gosec // Creating profile output file
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}
		session.cpuFile = cpu

		if err := pprof.StartCPUProfile(cpu); err != nil {
			_ = cpu.Close() // Ignore error during cleanup
			return fmt.Errorf("failed to start CPU profiling: %w", err)
		}
	}

	// Start memory profiling session
	if ps.config.EnableMemory {
		memSession, err := ps.memProfiler.StartProfiling(session.Name)
		if err != nil {
			return fmt.Errorf("failed to start memory profiling: %w", err)
		}
		session.memSession = memSession
	}

	// Start execution trace
	if ps.config.EnableTrace {
		traceFile := filepath.Join(session.OutputDir, "trace.out")
		traceF, err := os.Create(traceFile) //nolint:gosec // Creating trace output file
		if err != nil {
			return fmt.Errorf("failed to create trace file: %w", err)
		}
		session.traceFile = traceF

		if err := trace.Start(traceF); err != nil {
			_ = traceF.Close() // Ignore error during cleanup
			return fmt.Errorf("failed to start execution trace: %w", err)
		}
	}

	// Prepare additional profile file paths
	session.goroutineFile = filepath.Join(session.OutputDir, "goroutine.prof")
	session.blockFile = filepath.Join(session.OutputDir, "block.prof")
	session.mutexFile = filepath.Join(session.OutputDir, "mutex.prof")
	session.allocsFile = filepath.Join(session.OutputDir, "allocs.prof")

	session.started = true
	return nil
}

// stopSession stops all profiling for a session
func (ps *ProfileSuite) stopSession(session *ComprehensiveSession) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.started || session.stopped {
		return nil
	}

	// Stop CPU profiling
	if ps.config.EnableCPU && session.cpuFile != nil {
		pprof.StopCPUProfile()
		_ = session.cpuFile.Close() // Ignore error during cleanup
	}

	// Stop memory profiling session
	if ps.config.EnableMemory && session.memSession != nil {
		if err := ps.memProfiler.StopProfiling(session.memSession.Name); err != nil {
			log.Printf("Warning: failed to stop memory profiling: %v\n", err)
		}
	}

	// Stop execution trace
	if ps.config.EnableTrace && session.traceFile != nil {
		trace.Stop()
		_ = session.traceFile.Close() // Ignore error during cleanup
	}

	// Capture additional profiles
	ps.captureAdditionalProfiles(session)

	// Capture final memory snapshot
	session.endSnapshot = CaptureMemStats(fmt.Sprintf("%s_end", session.Name))

	session.stopped = true
	return nil
}

// captureAdditionalProfiles captures goroutine, block, mutex, and allocs profiles
func (ps *ProfileSuite) captureAdditionalProfiles(session *ComprehensiveSession) {
	profiles := []struct {
		name     string
		filename string
		debug    int
		enabled  bool
	}{
		{"goroutine", session.goroutineFile, 1, true},
		{"block", session.blockFile, 1, ps.config.EnableBlock},
		{"mutex", session.mutexFile, 1, ps.config.EnableMutex},
		{"allocs", session.allocsFile, 0, ps.config.EnableMemory},
	}

	for _, profileInfo := range profiles {
		if !profileInfo.enabled {
			continue
		}

		profile := pprof.Lookup(profileInfo.name)
		if profile == nil {
			continue
		}

		f, err := os.Create(profileInfo.filename)
		if err != nil {
			log.Printf("Warning: failed to create %s profile file: %v\n", profileInfo.name, err)
			continue
		}

		if err := profile.WriteTo(f, profileInfo.debug); err != nil {
			log.Printf("Warning: failed to write %s profile: %v\n", profileInfo.name, err)
		}

		_ = f.Close() // Ignore error during cleanup
	}
}

// generateReports generates comprehensive analysis reports
func (ps *ProfileSuite) generateReports(ctx context.Context, session *ComprehensiveSession) error {
	if ps.config.ReportFormat == "text" || ps.config.ReportFormat == "both" {
		if err := ps.generateTextReport(session); err != nil {
			return fmt.Errorf("failed to generate text report: %w", err)
		}
	}

	if ps.config.ReportFormat == "html" || ps.config.ReportFormat == "both" {
		if err := ps.generateHTMLReport(ctx, session); err != nil {
			return fmt.Errorf("failed to generate HTML report: %w", err)
		}
	}

	return nil
}

// generateTextReport creates a comprehensive text report
func (ps *ProfileSuite) generateTextReport(session *ComprehensiveSession) error {
	reportFile := filepath.Join(session.OutputDir, "comprehensive_report.txt")

	f, err := os.Create(reportFile) //nolint:gosec // Creating report output file
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Ignore error in defer

	duration := time.Since(session.StartTime)

	// Report header
	writeToReport(f, "Comprehensive Profiling Report\n")
	writeToReport(f, "==============================\n\n")
	writeToReport(f, "Session: %s\n", session.Name)
	writeToReport(f, "Start Time: %s\n", session.StartTime.Format(time.RFC3339))
	writeToReport(f, "Duration: %s\n", duration)
	writeToReport(f, "Output Directory: %s\n\n", session.OutputDir)

	// Configuration
	writeToReport(f, "Configuration\n")
	writeToReport(f, "-------------\n")
	writeToReport(f, "CPU Profiling: %t\n", ps.config.EnableCPU)
	writeToReport(f, "Memory Profiling: %t\n", ps.config.EnableMemory)
	writeToReport(f, "Execution Trace: %t\n", ps.config.EnableTrace)
	writeToReport(f, "Block Profiling: %t (rate: %d)\n", ps.config.EnableBlock, ps.config.BlockProfileRate)
	writeToReport(f, "Mutex Profiling: %t (fraction: %d)\n", ps.config.EnableMutex, ps.config.MutexProfileFraction)
	writeToReport(f, "\n")

	// Memory analysis
	if session.endSnapshot.Label != "" {
		comparison := session.startSnapshot.Compare(session.endSnapshot)
		writeToReport(f, "Memory Analysis\n")
		writeToReport(f, "---------------\n")
		writeToReport(f, "%s\n", comparison.String())
	}

	// Profile files summary
	writeToReport(f, "Generated Profile Files\n")
	writeToReport(f, "-----------------------\n")

	profileFiles := []string{
		"cpu.prof",
		"trace.out",
		"goroutine.prof",
		"block.prof",
		"mutex.prof",
		"allocs.prof",
	}

	for _, filename := range profileFiles {
		filePath := filepath.Join(session.OutputDir, filename)
		if info, err := os.Stat(filePath); err == nil {
			writeToReport(f, "%s: %s (%s)\n", filename, formatBytes(uint64(max(0, info.Size()))), info.ModTime().Format(time.RFC3339)) //nolint:gosec // file sizes are non-negative and bounded
		}
	}

	// Analysis commands
	writeToReport(f, "\nAnalysis Commands\n")
	writeToReport(f, "-----------------\n")

	if ps.config.EnableCPU {
		writeToReport(f, "CPU Profile Analysis:\n")
		writeToReport(f, "  go tool pprof -top %s\n", filepath.Join(session.OutputDir, "cpu.prof"))
		writeToReport(f, "  go tool pprof -web %s\n", filepath.Join(session.OutputDir, "cpu.prof"))
	}

	if ps.config.EnableTrace {
		writeToReport(f, "\nExecution Trace Analysis:\n")
		writeToReport(f, "  go tool trace %s\n", filepath.Join(session.OutputDir, "trace.out"))
	}

	if ps.config.EnableBlock {
		writeToReport(f, "\nBlock Profile Analysis:\n")
		writeToReport(f, "  go tool pprof -top %s\n", filepath.Join(session.OutputDir, "block.prof"))
	}

	if ps.config.EnableMutex {
		writeToReport(f, "\nMutex Profile Analysis:\n")
		writeToReport(f, "  go tool pprof -top %s\n", filepath.Join(session.OutputDir, "mutex.prof"))
	}

	return nil
}

// generateHTMLReport creates an HTML report using go tool pprof
func (ps *ProfileSuite) generateHTMLReport(ctx context.Context, session *ComprehensiveSession) error {
	if !ps.config.EnableCPU {
		return nil // Skip HTML report if no CPU profile
	}

	cpuProfile := filepath.Join(session.OutputDir, "cpu.prof")
	htmlReport := filepath.Join(session.OutputDir, "cpu_profile.html")

	// Check if CPU profile exists and has content before generating HTML report
	if info, err := os.Stat(cpuProfile); err != nil || info.Size() == 0 {
		// Create a placeholder HTML report if no valid CPU profile exists
		placeholder := `<!DOCTYPE html>
<html>
<head><title>CPU Profile Report</title></head>
<body>
<h1>CPU Profile Report</h1>
<p>No CPU profile data available for session: ` + session.Name + `</p>
<p>This may be because the profiled function executed too quickly to capture meaningful CPU profile data.</p>
</body>
</html>`
		return os.WriteFile(htmlReport, []byte(placeholder), 0o644) //nolint:gosec // HTML report file with standard permissions
	}

	// Use a simpler pprof command that generates SVG instead of starting HTTP server
	cmd := exec.CommandContext(ctx, "go", "tool", "pprof", "-svg", "-output", htmlReport+".svg", cpuProfile) //nolint:gosec // Go pprof tool with controlled arguments

	if err := cmd.Run(); err != nil {
		// If SVG generation fails, create a simple HTML report with text output
		textReport := filepath.Join(session.OutputDir, "cpu_profile.txt")
		textCmd := exec.CommandContext(ctx, "go", "tool", "pprof", "-text", "-output", textReport, cpuProfile) //nolint:gosec // Go pprof tool with controlled arguments

		if textErr := textCmd.Run(); textErr != nil {
			return fmt.Errorf("failed to generate HTML or text report: HTML error: %w, Text error: %w", err, textErr)
		}

		// Create HTML wrapper for text report
		if textContent, readErr := os.ReadFile(textReport); readErr == nil { //nolint:gosec // Reading generated text report file
			htmlContent := `<!DOCTYPE html>
<html>
<head><title>CPU Profile Report</title></head>
<body>
<h1>CPU Profile Report - ` + session.Name + `</h1>
<pre>` + string(textContent) + `</pre>
</body>
</html>`
			return os.WriteFile(htmlReport, []byte(htmlContent), 0o644) //nolint:gosec // HTML report file with standard permissions
		}

		return fmt.Errorf("failed to generate HTML report and read text report: %w", err)
	}

	return nil
}

// createSessionSummary creates a summary of the completed session
func (ps *ProfileSuite) createSessionSummary(session *ComprehensiveSession) SessionSummary {
	duration := time.Since(session.StartTime)

	// Count profile files and calculate total size
	fileCount := 0
	totalSize := int64(0)

	profileFiles := []string{
		"cpu.prof", "trace.out", "goroutine.prof",
		"block.prof", "mutex.prof", "allocs.prof",
		"comprehensive_report.txt", "cpu_profile.html",
	}

	profileTypes := make([]string, 0)

	for _, filename := range profileFiles {
		filePath := filepath.Join(session.OutputDir, filename)
		if info, err := os.Stat(filePath); err == nil {
			fileCount++
			totalSize += info.Size()

			// Add to profile types
			ext := strings.TrimSuffix(filename, filepath.Ext(filename))
			profileTypes = append(profileTypes, ext)
		}
	}

	return SessionSummary{
		Name:         session.Name,
		StartTime:    session.StartTime,
		Duration:     duration,
		OutputDir:    session.OutputDir,
		ProfileTypes: profileTypes,
		FileCount:    fileCount,
		TotalSize:    totalSize,
	}
}

// cleanupOldSessions removes old profiling sessions to save disk space
func (ps *ProfileSuite) cleanupOldSessions() {
	if len(ps.sessionHistory) <= ps.config.MaxSessionsToKeep {
		return
	}

	// Remove oldest sessions
	toRemove := len(ps.sessionHistory) - ps.config.MaxSessionsToKeep
	for i := 0; i < toRemove; i++ {
		session := ps.sessionHistory[i]
		if err := os.RemoveAll(session.OutputDir); err != nil {
			log.Printf("Warning: failed to cleanup session directory %s: %v\n", session.OutputDir, err)
		}
	}

	// Update session history
	ps.sessionHistory = ps.sessionHistory[toRemove:]
}

// GetSessionHistory returns the history of profiling sessions
func (ps *ProfileSuite) GetSessionHistory() []SessionSummary {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]SessionSummary, len(ps.sessionHistory))
	copy(history, ps.sessionHistory)
	return history
}

// GetCurrentSession returns information about the current profiling session
func (ps *ProfileSuite) GetCurrentSession() *ComprehensiveSession {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.currentSession
}

// IsActive returns true if a profiling session is currently active
func (ps *ProfileSuite) IsActive() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.currentSession != nil
}
