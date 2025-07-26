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

	// Start collection with the context
	go mc.collect(ctx)

	// Initialize profiler if enabled
	if config.EnableProfiling {
		mc.profiler = profiling.NewMemoryProfiler(config.ProfileDir)
		if err := mc.profiler.Enable(); err != nil {
			log.Printf("Warning: failed to enable profiler: %v\n", err)
		}
	}

	// Collection is already started in the constructor

	return mc
}

// GetCurrentMetrics returns the current metrics
func (mc *MetricsCollector) GetCurrentMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range mc.metrics {
		result[k] = v
	}

	return result
}

// GetMetricsHistory returns historical metrics
func (mc *MetricsCollector) GetMetricsHistory() []MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]MetricsSnapshot, len(mc.history))
	copy(history, mc.history)

	return history
}

// ServeHTTP implements http.Handler for metrics endpoint
func (mc *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check query parameters for different data types
	switch r.URL.Query().Get("type") {
	case "history":
		history := mc.GetMetricsHistory()
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"history": history,
		}); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		}
	default:
		current := mc.GetCurrentMetrics()
		if err := json.NewEncoder(w).Encode(current); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		}
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
		gcMetrics["last_gc_time"] = time.Unix(0, int64(memStats.LastGC)).Unix() //nolint:gosec // GC timestamp unlikely to overflow
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

	// Add to history
	snapshot := MetricsSnapshot{
		Timestamp: now,
		Metrics:   make(map[string]interface{}),
	}

	// Deep copy metrics for history
	for k, v := range currentMetrics {
		snapshot.Metrics[k] = v
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

// StartBackground starts the dashboard server in the background
func (d *Dashboard) StartBackground(ctx context.Context) error {
	go func() {
		if err := d.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Dashboard server error: %v\n", err)
		}
	}()

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

// StartDashboard is a convenience function to start a dashboard with default config
func StartDashboard(port int) error {
	config := DefaultDashboardConfig()
	config.Port = port

	dashboard := NewDashboard(config)
	return dashboard.Start()
}

// StartDashboardWithProfiling starts a dashboard with profiling enabled
func StartDashboardWithProfiling(port int, profileDir string) error {
	config := DefaultDashboardConfig()
	config.Port = port
	config.EnableProfiling = true
	config.ProfileDir = profileDir

	dashboard := NewDashboard(config)
	return dashboard.Start()
}
