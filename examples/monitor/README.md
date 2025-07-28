# Monitoring Dashboard Examples

This directory contains examples demonstrating how to use the built-in monitoring dashboard for performance monitoring and profiling in go-broadcast.

## 📊 What is the Monitoring Dashboard?

The monitoring dashboard is a built-in web-based performance monitoring system that provides:

- **Real-time Metrics**: Memory usage, goroutines, GC activity, heap objects
- **Interactive Charts**: Historical data visualization with Chart.js
- **System Information**: Uptime, status indicators, system stats
- **Performance Alerts**: Automatic threshold monitoring
- **Memory Profiling**: Optional profiling with profile generation
- **JSON API**: Programmatic access to metrics data

## 🚀 Quick Start

### Simple Dashboard Example

```bash
# Run the simple monitoring example
cd examples/monitor
go run simple.go
```

Then open your browser to: **http://localhost:8080**

### Advanced Dashboard with Profiling

```bash
# Run the advanced example with profiling
cd examples/monitor
go run advanced.go
```

Then open your browser to: **http://localhost:8081**

Profile data will be saved to `./profiles/` directory.

## 📁 Examples Overview

### 1. `simple.go` - Basic Monitoring

**What it demonstrates:**
- Basic dashboard setup with default configuration
- Simple workload simulation to generate metrics
- Graceful shutdown handling
- Memory allocation patterns and GC activity

**Key Features:**
- Dashboard on port 8080
- 1-second metric collection interval
- 5 minutes of historical data
- Simulated workload with goroutines and memory activity

**Perfect for:**
- Getting started with monitoring
- Development and debugging
- Basic performance analysis

### 2. `advanced.go` - Full-Featured Monitoring

**What it demonstrates:**
- Advanced configuration with profiling enabled
- Multiple intensive workload simulations
- Memory stress testing
- CPU-intensive algorithmic workloads
- Profile generation and storage

**Key Features:**
- Dashboard on port 8081
- 500ms metric collection interval (more responsive)
- Profiling enabled with profile storage
- Multiple concurrent workload simulations
- Advanced memory management testing

**Perfect for:**
- Performance optimization
- Memory leak detection
- Production-like monitoring
- Detailed profiling analysis

## 🔍 Dashboard Features

### Main Dashboard Interface

When you access the dashboard in your browser, you'll see:

1. **Status Bar**: Real-time connection status and uptime
2. **Real-time Stats Grid**: Current metrics display
3. **Memory Usage Chart**: Historical memory allocation over time
4. **Goroutines Chart**: Goroutine count trends
5. **Garbage Collection Chart**: GC activity and pause times
6. **System Information**: Go version, CPU count, etc.
7. **Performance Alerts**: Automatic threshold warnings

### API Endpoints

The dashboard also provides JSON API endpoints:

```bash
# Get current metrics
curl http://localhost:8080/api/metrics

# Get historical data
curl http://localhost:8080/api/metrics?type=history

# Health check
curl http://localhost:8080/api/health
```

## ⚙️ Configuration Options

### Basic Configuration

```go
config := monitoring.DefaultDashboardConfig()
config.Port = 8080                    // Dashboard port
config.CollectInterval = time.Second  // How often to collect metrics
config.RetainHistory = 300           // Number of data points to keep
```

### Advanced Configuration

```go
config := monitoring.DefaultDashboardConfig()
config.Port = 8081
config.CollectInterval = 500 * time.Millisecond  // More frequent collection
config.RetainHistory = 600                       // More history
config.EnableProfiling = true                    // Enable memory profiling
config.ProfileDir = "./profiles"                 // Profile storage directory
```

### Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `Port` | 8080 | HTTP port for dashboard |
| `CollectInterval` | 1s | How often to collect metrics |
| `RetainHistory` | 300 | Data points to keep in memory |
| `EnableProfiling` | false | Enable memory profiling |
| `ProfileDir` | "./profiles" | Directory for profile files |

## 💡 Integration Examples

### In Your Application

```go
package main

import (
    "github.com/mrz1836/go-broadcast/internal/monitoring"
)

func main() {
    // Simple dashboard
    go monitoring.StartDashboard(8080)
    
    // Your application code here
    // ...
}
```

### With Custom Configuration

```go
func startMonitoring() {
    config := monitoring.DefaultDashboardConfig()
    config.Port = 3000
    config.EnableProfiling = true
    
    dashboard := monitoring.NewDashboard(config)
    go dashboard.Start()
}
```

### In Tests

```go
func TestWithMonitoring(t *testing.T) {
    config := monitoring.DefaultDashboardConfig()
    config.Port = 0 // Random port
    
    dashboard := monitoring.NewDashboard(config)
    defer dashboard.Stop(context.Background())
    
    // Your test code here
}
```

## 🔧 Troubleshooting

### Common Issues

**Dashboard won't start:**
- Check if port is already in use
- Ensure you have permission to bind to the port
- Try a different port number

**No metrics showing:**
- Wait a few seconds for initial data collection
- Check browser console for JavaScript errors
- Verify the API endpoints are accessible

**Profiling not working:**
- Ensure `EnableProfiling` is set to `true`
- Check that the profile directory exists and is writable
- Verify disk space is available

### Debug Mode

Add verbose logging to see what's happening:

```go
log.SetLevel(log.DebugLevel)
dashboard := monitoring.NewDashboard(config)
```

## 📈 Understanding the Metrics

### Memory Metrics

- **Alloc**: Currently allocated heap memory
- **TotalAlloc**: Total bytes allocated (cumulative)
- **Sys**: Total memory obtained from OS
- **HeapAlloc**: Heap memory in use
- **HeapSys**: Heap memory obtained from OS
- **HeapObjects**: Number of objects on heap

### Garbage Collection Metrics

- **NumGC**: Number of GC cycles completed
- **PauseTotalNs**: Total GC pause time
- **LastPause**: Most recent GC pause duration
- **NextGC**: Target heap size for next GC

### Runtime Metrics

- **Goroutines**: Current number of goroutines
- **NumCPU**: Number of CPU cores
- **GOMAXPROCS**: Maximum number of OS threads

## 🎯 Performance Tips

### For Development
- Use the simple example with 1-second intervals
- Focus on memory allocation patterns
- Watch for goroutine leaks

### For Production Monitoring
- Use longer collection intervals (5-10 seconds)
- Enable profiling only when needed
- Monitor GC pause times and frequency
- Set up alerts for memory thresholds

### For Performance Testing
- Use the advanced example for stress testing
- Monitor memory growth patterns
- Watch for GC pressure indicators
- Profile before and after optimizations

## 🔗 Related Documentation

- [Performance Optimization Guide](../../docs/performance-optimization.md)
- [Profiling Guide](../../docs/profiling-guide.md)
- [Troubleshooting](../../docs/troubleshooting.md)

## 🤝 Contributing

Found a bug or want to improve the examples? Please:

1. Check existing issues
2. Create a detailed bug report
3. Submit a pull request with tests

---

**Happy Monitoring!** 📊✨