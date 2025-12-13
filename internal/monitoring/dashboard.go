// Package monitoring provides runtime metrics collection and dashboard functionality for system monitoring.
package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/mrz1836/go-broadcast/internal/profiling"
)

// ErrInvalidPort is returned when a port number is outside the valid range (1-65535).
var ErrInvalidPort = errors.New("invalid port")

// deepCopyMetrics creates a deep copy of the metrics map to prevent data races.
// This ensures callers cannot modify internal state through returned references.
func deepCopyMetrics(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	result := make(map[string]interface{}, len(src))
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMetrics(val)
		default:
			result[k] = v
		}
	}
	return result
}

// MetricsCollector collects and aggregates runtime metrics
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]interface{}

	// Configuration
	collectInterval time.Duration
	retainHistory   int

	// History storage
	history []MetricsSnapshot

	// Control
	cancel context.CancelFunc

	// Components
	profiler *profiling.MemoryProfiler
}

// MetricsSnapshot represents metrics at a point in time
type MetricsSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// DashboardConfig configures the monitoring dashboard
type DashboardConfig struct {
	Port            int           `json:"port"`
	CollectInterval time.Duration `json:"collect_interval"`
	RetainHistory   int           `json:"retain_history"`
	EnableProfiling bool          `json:"enable_profiling"`
	ProfileDir      string        `json:"profile_dir"`
}

// DefaultDashboardConfig returns default dashboard configuration
func DefaultDashboardConfig() DashboardConfig {
	return DashboardConfig{
		Port:            8080,
		CollectInterval: time.Second,
		RetainHistory:   300, // 5 minutes of history at 1-second intervals
		EnableProfiling: false,
		ProfileDir:      "./profiles",
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config DashboardConfig) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	mc := &MetricsCollector{
		metrics:         make(map[string]interface{}),
		collectInterval: config.CollectInterval,
		retainHistory:   config.RetainHistory,
		history:         make([]MetricsSnapshot, 0, config.RetainHistory),
		cancel:          cancel,
	}

	// Initialize profiler BEFORE starting collection goroutine
	// to prevent race condition on profiler access
	if config.EnableProfiling {
		mc.profiler = profiling.NewMemoryProfiler(config.ProfileDir)
		if err := mc.profiler.Enable(); err != nil {
			log.Printf("Warning: failed to enable profiler: %v\n", err)
		}
	}

	// Now safe to start collection goroutine
	go mc.collect(ctx)

	return mc
}

// GetCurrentMetrics returns a deep copy of the current metrics.
// The returned map is safe to modify without affecting internal state.
func (mc *MetricsCollector) GetCurrentMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a deep copy to prevent data races on nested maps
	return deepCopyMetrics(mc.metrics)
}

// GetMetricsHistory returns a deep copy of historical metrics.
// The returned slice is safe to modify without affecting internal state.
func (mc *MetricsCollector) GetMetricsHistory() []MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a deep copy including nested metrics maps
	history := make([]MetricsSnapshot, len(mc.history))
	for i, snapshot := range mc.history {
		history[i] = MetricsSnapshot{
			Timestamp: snapshot.Timestamp,
			Metrics:   deepCopyMetrics(snapshot.Metrics),
		}
	}

	return history
}

// ServeHTTP implements http.Handler for metrics endpoint
func (mc *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers first (these can always be sent)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Prepare response data
	var data interface{}
	switch r.URL.Query().Get("type") {
	case "history":
		data = map[string]interface{}{"history": mc.GetMetricsHistory()}
	default:
		data = mc.GetCurrentMetrics()
	}

	// Buffer the JSON response before sending headers
	// This allows proper error handling with http.Error
	buf, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}

	// Now safe to set Content-Type and write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf); err != nil {
		log.Printf("Warning: failed to write metrics response: %v", err)
	}
}

// Stop stops the metrics collection
func (mc *MetricsCollector) Stop() {
	mc.cancel()

	if mc.profiler != nil {
		if err := mc.profiler.Disable(); err != nil {
			log.Printf("Warning: failed to disable profiler: %v", err)
		}
	}
}

// collect periodically gathers metrics
func (mc *MetricsCollector) collect(ctx context.Context) {
	// Ensure collectInterval is positive to prevent panic
	if mc.collectInterval <= 0 {
		mc.collectInterval = time.Second // Default fallback
	}

	ticker := time.NewTicker(mc.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.updateMetrics()
		case <-ctx.Done():
			return
		}
	}
}

// updateMetrics collects current system metrics
func (mc *MetricsCollector) updateMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get current timestamp
	now := time.Now()

	// Create metrics map
	currentMetrics := map[string]interface{}{
		"timestamp": now.Unix(),
		"memory": map[string]interface{}{
			"alloc_mb":       float64(memStats.Alloc) / 1024 / 1024,
			"total_alloc_mb": float64(memStats.TotalAlloc) / 1024 / 1024,
			"sys_mb":         float64(memStats.Sys) / 1024 / 1024,
			"heap_alloc_mb":  float64(memStats.HeapAlloc) / 1024 / 1024,
			"heap_sys_mb":    float64(memStats.HeapSys) / 1024 / 1024,
			"heap_objects":   memStats.HeapObjects,
			"stack_sys_mb":   float64(memStats.StackSys) / 1024 / 1024,
			"next_gc_mb":     float64(memStats.NextGC) / 1024 / 1024,
		},
		"gc": map[string]interface{}{
			"num_gc":         memStats.NumGC,
			"num_forced":     memStats.NumForcedGC,
			"pause_total_ms": float64(memStats.PauseTotalNs) / 1e6,
		},
		"runtime": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"num_cpu":    runtime.NumCPU(),
			"gomaxprocs": runtime.GOMAXPROCS(-1),
			"go_version": runtime.Version(),
		},
	}

	// Add GC pause information if available
	if memStats.NumGC > 0 {
		// Get the most recent GC pause
		recentPause := memStats.PauseNs[(memStats.NumGC+255)%256]
		avgPause := float64(memStats.PauseTotalNs) / float64(memStats.NumGC) / 1e6

		gcMetrics := currentMetrics["gc"].(map[string]interface{})
		gcMetrics["last_pause_ms"] = float64(recentPause) / 1e6
		gcMetrics["avg_pause_ms"] = avgPause
		gcMetrics["last_gc_time"] = time.Unix(0, int64(memStats.LastGC)).Unix() //nolint:gosec // LastGC is nanoseconds since epoch, safe for int64 until year 2262
	}

	// Add profiler statistics if available
	if mc.profiler != nil {
		profilerStats := mc.profiler.GetProfilerStats()
		currentMetrics["profiler"] = map[string]interface{}{
			"enabled":         profilerStats.Enabled,
			"active_sessions": profilerStats.ActiveSessions,
			"total_sessions":  profilerStats.TotalSessions,
			"profile_count":   profilerStats.ProfileCount,
		}
	}

	// Update current metrics
	mc.mu.Lock()
	mc.metrics = currentMetrics

	// Add to history with deep copy to prevent mutation
	snapshot := MetricsSnapshot{
		Timestamp: now,
		Metrics:   deepCopyMetrics(currentMetrics),
	}

	mc.history = append(mc.history, snapshot)

	// Trim history if needed
	if len(mc.history) > mc.retainHistory {
		mc.history = mc.history[1:]
	}

	mc.mu.Unlock()
}

// Dashboard manages the HTTP dashboard server
type Dashboard struct {
	collector *MetricsCollector
	server    *http.Server
	config    DashboardConfig
	startTime time.Time
}

// NewDashboard creates a new monitoring dashboard
func NewDashboard(config DashboardConfig) *Dashboard {
	collector := NewMetricsCollector(config)

	mux := http.NewServeMux()

	dashboard := &Dashboard{
		collector: collector,
		server:    nil, // Will be set below
		config:    config,
		startTime: time.Now(),
	}

	// API endpoints
	mux.Handle("/api/metrics", collector)
	mux.HandleFunc("/api/health", dashboard.healthHandler)

	// Static dashboard page
	mux.HandleFunc("/", dashboardHandler)
	mux.HandleFunc("/dashboard.js", dashboardJSHandler)
	mux.HandleFunc("/dashboard.css", dashboardCSSHandler)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second, // Prevent Slowloris attacks
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	dashboard.server = server
	return dashboard
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	log.Printf("Performance dashboard starting on http://localhost:%d\n", d.config.Port)
	return d.server.ListenAndServe()
}

// StartBackground starts the dashboard server in the background.
// Returns an error if the server fails to start (e.g., port in use).
// Blocks until the context is canceled, then performs graceful shutdown.
func (d *Dashboard) StartBackground(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		if err := d.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	// Give server brief startup time to detect immediate failures
	// (e.g., port already in use)
	select {
	case err := <-errChan:
		if err != nil {
			d.collector.Stop()
			return fmt.Errorf("failed to start dashboard server: %w", err)
		}
	case <-time.After(50 * time.Millisecond):
		// Server started successfully, proceed to wait for context
	case <-ctx.Done():
		// Context canceled during startup
		return d.Stop(ctx)
	}

	// Wait for context cancellation
	<-ctx.Done()
	return d.Stop(ctx)
}

// Stop stops the dashboard server
func (d *Dashboard) Stop(ctx context.Context) error {
	d.collector.Stop()
	return d.server.Shutdown(ctx)
}

// GetCollector returns the metrics collector
func (d *Dashboard) GetCollector() *MetricsCollector {
	return d.collector
}

// healthHandler handles health check requests
func (d *Dashboard) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(d.startTime).Seconds(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Warning: failed to encode health response: %v", err)
	}
}

// dashboardHandler serves the main dashboard HTML page
func dashboardHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if _, err := fmt.Fprint(w, dashboardHTML); err != nil {
		log.Printf("Warning: failed to write dashboard HTML: %v", err)
	}
}

// dashboardJSHandler serves the dashboard JavaScript
func dashboardJSHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	if _, err := fmt.Fprint(w, dashboardJS); err != nil {
		log.Printf("Warning: failed to write dashboard JS: %v", err)
	}
}

// dashboardCSSHandler serves the dashboard CSS
func dashboardCSSHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	if _, err := fmt.Fprint(w, dashboardCSS); err != nil {
		log.Printf("Warning: failed to write dashboard CSS: %v", err)
	}
}

// StartDashboard is a convenience function to start a dashboard with default config.
// Returns an error if port is invalid (must be 1-65535).
func StartDashboard(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: %d must be between 1 and 65535", ErrInvalidPort, port)
	}

	config := DefaultDashboardConfig()
	config.Port = port

	dashboard := NewDashboard(config)
	return dashboard.Start()
}

// StartDashboardWithProfiling starts a dashboard with profiling enabled.
// Returns an error if port is invalid (must be 1-65535).
func StartDashboardWithProfiling(port int, profileDir string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: %d must be between 1 and 65535", ErrInvalidPort, port)
	}

	config := DefaultDashboardConfig()
	config.Port = port
	config.EnableProfiling = true
	config.ProfileDir = profileDir

	dashboard := NewDashboard(config)
	return dashboard.Start()
}
