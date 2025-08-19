// Package main demonstrates a simple monitoring dashboard example
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/mrz1836/go-broadcast/internal/monitoring"
	"github.com/mrz1836/go-broadcast/internal/pool"
)

func main() {
	log.Println("🔍 Simple Monitoring Dashboard Example")
	log.Println("=====================================")

	// Create a simple configuration
	config := monitoring.DefaultDashboardConfig()
	config.Port = 8080
	config.CollectInterval = time.Second
	config.RetainHistory = 300 // 5 minutes of history

	log.Printf("📊 Starting dashboard on http://localhost:%d", config.Port)
	log.Println("💡 Press Ctrl+C to stop")

	// Create and start dashboard
	dashboard := monitoring.NewDashboard(config)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Catch interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start dashboard in background
	go func() {
		if err := dashboard.StartBackground(ctx); err != nil {
			log.Printf("Dashboard error: %v", err)
		}
	}()

	// Simulate some workload to generate interesting metrics
	go simulateWorkload()

	// Wait for interrupt
	<-sigChan
	log.Println("🛑 Shutting down dashboard...")

	// Cancel context to stop dashboard
	cancel()

	// Give it a moment to cleanup
	time.Sleep(100 * time.Millisecond)
	log.Println("✅ Dashboard stopped")
}

// simulateWorkload creates some activity to make the dashboard interesting
func simulateWorkload() {
	// Create a buffer pool for some memory activity
	bufferPool := pool.NewBufferPool()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	counter := 0
	for range ticker.C {
		counter++

		// Do some memory allocation
		data := make([]byte, 1024*counter%100)
		_ = data

		// Use the buffer pool
		buffer := bufferPool.GetBuffer(1024)
		fmt.Fprintf(buffer, "Iteration %d - %s", counter, time.Now().Format(time.RFC3339))
		bufferPool.PutBuffer(buffer)

		// Create some goroutines periodically
		if counter%3 == 0 {
			for i := 0; i < 5; i++ {
				go func(id int) {
					time.Sleep(time.Duration(id*100) * time.Millisecond)
					// Simulate some work
					runtime.GC()
				}(i)
			}
		}

		// Force garbage collection occasionally
		if counter%5 == 0 {
			runtime.GC()
		}

		// Print progress
		if counter%10 == 0 {
			log.Printf("📈 Workload iteration %d (Goroutines: %d)", counter, runtime.NumGoroutine())
		}
	}
}
