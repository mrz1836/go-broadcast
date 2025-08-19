// Package main demonstrates an advanced monitoring dashboard with profiling
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

	"github.com/mrz1836/go-broadcast/internal/algorithms"
	"github.com/mrz1836/go-broadcast/internal/memory"
	"github.com/mrz1836/go-broadcast/internal/monitoring"
	"github.com/mrz1836/go-broadcast/internal/pool"
)

func main() {
	log.Println("🚀 Advanced Monitoring Dashboard with Profiling")
	log.Println("================================================")

	// Create advanced configuration with profiling enabled
	config := monitoring.DefaultDashboardConfig()
	config.Port = 8081
	config.CollectInterval = 500 * time.Millisecond // More frequent collection
	config.RetainHistory = 600                      // 5 minutes of history at 500ms intervals
	config.EnableProfiling = true
	config.ProfileDir = "./profiles"

	log.Printf("📊 Starting advanced dashboard on http://localhost:%d", config.Port)
	log.Printf("🔬 Profiling enabled - profiles saved to: %s", config.ProfileDir)
	log.Println("💡 Press Ctrl+C to stop")

	// Ensure profile directory exists
	if err := os.MkdirAll(config.ProfileDir, 0o750); err != nil {
		log.Fatalf("Failed to create profile directory: %v", err)
	}

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

	// Start intensive workload simulation
	go intensiveWorkload()
	go memoryStressTest()
	go algorithmicWorkload()

	log.Println("🔥 Running intensive workloads...")
	log.Println("📈 Monitor the dashboard for performance metrics")
	log.Println("🔍 Check ./profiles/ directory for memory profiles")

	// Wait for interrupt
	<-sigChan
	log.Println("🛑 Shutting down dashboard...")

	// Cancel context to stop dashboard
	cancel()

	// Give it a moment to cleanup and save profiles
	time.Sleep(500 * time.Millisecond)
	log.Println("✅ Dashboard stopped")
	log.Println("📁 Profile data saved in ./profiles/")
}

// intensiveWorkload creates memory pressure and goroutine activity
func intensiveWorkload() {
	bufferPool := pool.NewBufferPool()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	counter := 0
	for range ticker.C {
		counter++

		// Create varying memory pressure
		dataSize := (counter % 20) * 1024 * 10 // 0-200KB allocations
		data := make([]byte, dataSize)
		for i := range data {
			data[i] = byte(i % 256)
		}

		// Stress the buffer pool
		for i := 0; i < 10; i++ {
			go func(id int) {
				buffer := bufferPool.GetBuffer(2048)
				defer bufferPool.PutBuffer(buffer)

				for j := 0; j < 100; j++ {
					fmt.Fprintf(buffer, "Worker %d iteration %d\n", id, j)
				}

				// Simulate some processing time
				time.Sleep(time.Duration(id*10) * time.Millisecond)
			}(i)
		}

		// Force GC periodically to see GC metrics
		if counter%5 == 0 {
			runtime.GC()
		}

		if counter%15 == 0 {
			log.Printf("🔥 Intensive workload: %d iterations (Goroutines: %d)", counter, runtime.NumGoroutine())
		}
	}
}

// memoryStressTest specifically tests memory allocation patterns
func memoryStressTest() {
	stringIntern := memory.NewStringIntern()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	iteration := 0
	for range ticker.C {
		iteration++

		// Allocate and use memory in different patterns
		switch iteration % 4 {
		case 0:
			// String interning test - small frequent allocations
			for i := 0; i < 1000; i++ {
				testStr := fmt.Sprintf("test-string-%d", i%100) // Reuse some strings
				interned := stringIntern.Intern(testStr)
				_ = interned
			}

		case 1:
			// Large allocations
			for i := 0; i < 10; i++ {
				data := make([]byte, 64*1024) // 64KB
				for j := range data {
					data[j] = byte(j % 256)
				}
				_ = data
			}

		case 2:
			// Mixed allocation sizes using memory helpers
			sizes := []int{256, 1024, 4096, 16384}
			for _, size := range sizes {
				slice := memory.PreallocateSlice[byte](size)
				for j := 0; j < size; j++ {
					slice = append(slice, byte(j%256))
				}
				_ = slice
			}

		case 3:
			// Create some retained memory (memory leak simulation)
			leakedData := make([][]byte, 100)
			for i := range leakedData {
				leakedData[i] = make([]byte, 1024)
			}
			// Intentionally don't free this immediately
			go func() {
				time.Sleep(5 * time.Second)
				_ = leakedData // Reference to prevent optimization
			}()
		}

		if iteration%10 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("💾 Memory stress test: Heap=%dMB, Sys=%dMB",
				m.HeapAlloc/1024/1024, m.Sys/1024/1024)
		}
	}
}

// algorithmicWorkload creates CPU-intensive tasks
func algorithmicWorkload() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	iteration := 0
	for range ticker.C {
		iteration++

		// Run CPU-intensive algorithms in goroutines
		for i := 0; i < 5; i++ {
			go func(workerID int) {
				// Generate test data
				data := make([]int, 10000)
				for j := range data {
					data[j] = (workerID*1000 + j) % 100000
				}

				// Run optimized algorithms (using available functions)
				// Test binary detection on the data
				dataBytes := make([]byte, len(data))
				for idx, v := range data {
					dataBytes[idx] = byte(v % 256)
				}

				isBinary := algorithms.IsBinaryOptimized(dataBytes)
				_ = isBinary

				// Test diff algorithm
				altData := make([]byte, len(dataBytes))
				copy(altData, dataBytes)
				// Modify some bytes
				for k := 0; k < len(altData)/10; k++ {
					altData[k*10] = byte((int(altData[k*10]) + 1) % 256)
				}

				diff, hasDiff := algorithms.DiffOptimized(dataBytes, altData, 1024)
				_ = diff
				_ = hasDiff

				// Some mathematical computation
				sum := 0
				for k := 0; k < 100000; k++ {
					sum += k * k % 997 // Prime number for better distribution
				}
				_ = sum
			}(i)
		}

		if iteration%5 == 0 {
			log.Printf("🧮 Algorithmic workload: %d iterations (CPU cores: %d)",
				iteration, runtime.NumCPU())
		}
	}
}
