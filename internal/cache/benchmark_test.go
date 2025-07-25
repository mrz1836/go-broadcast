// Package cache provides benchmarking tests for cache implementations and performance analysis.
package cache

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// mockAPIResponse simulates an API response structure
type mockAPIResponse struct {
	ID      int                    `json:"id"`
	Name    string                 `json:"name"`
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data"`
	Created time.Time              `json:"created"`
}

// simulateAPICall simulates a slow API call
func simulateAPICall(id int, latency time.Duration) (interface{}, error) {
	time.Sleep(latency)

	return &mockAPIResponse{
		ID:     id,
		Name:   fmt.Sprintf("Item_%d", id),
		Status: "active",
		Data: map[string]interface{}{
			"value":     rand.Intn(1000), //nolint:gosec // Using weak random for benchmark data is acceptable
			"timestamp": time.Now(),
			"tags":      []string{"tag1", "tag2", "tag3"},
		},
		Created: time.Now(),
	}, nil
}

// BenchmarkCacheBasicOperations tests basic cache operations
func BenchmarkCacheBasicOperations(b *testing.B) {
	cache := NewTTLCache(time.Minute, 1000)
	defer cache.Close()

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i%100)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}
	})

	b.Run("Get_Hit", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i%100)
			_, _ = cache.Get(key)
		}
	})

	b.Run("Get_Miss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("missing_key_%d", i)
			_, _ = cache.Get(key)
		}
	})

	b.Run("GetOrLoad", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("load_key_%d", i%50) // 50% hit rate
			_, _ = cache.GetOrLoad(key, func() (interface{}, error) {
				return fmt.Sprintf("loaded_value_%d", i), nil
			})
		}
	})
}

// BenchmarkCacheHitRates tests cache performance with different hit rates
func BenchmarkCacheHitRates(b *testing.B) {
	scenarios := []struct {
		name    string
		hitRate float64
		keyPool int
	}{
		{"HitRate_10", 0.1, 1000},
		{"HitRate_50", 0.5, 200},
		{"HitRate_80", 0.8, 50},
		{"HitRate_95", 0.95, 20},
		{"HitRate_99", 0.99, 10},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			cache := NewTTLCache(time.Minute, scenario.keyPool*2)
			defer cache.Close()

			// Pre-populate cache for expected hit rate
			hitKeys := int(float64(scenario.keyPool) * scenario.hitRate)
			for i := 0; i < hitKeys; i++ {
				key := fmt.Sprintf("hit_key_%d", i)
				cache.Set(key, fmt.Sprintf("value_%d", i))
			}

			b.ResetTimer()
			hits := 0
			misses := 0

			for i := 0; i < b.N; i++ {
				var key string
				if rand.Float64() < scenario.hitRate { //nolint:gosec // Using weak random for benchmark is acceptable
					// Generate a key that should hit
					key = fmt.Sprintf("hit_key_%d", rand.Intn(hitKeys)) //nolint:gosec // Using weak random for benchmark is acceptable
				} else {
					// Generate a key that should miss
					key = fmt.Sprintf("miss_key_%d", rand.Intn(scenario.keyPool)) //nolint:gosec // Using weak random for benchmark is acceptable
				}

				if _, found := cache.Get(key); found {
					hits++
				} else {
					misses++
				}
			}

			actualHitRate := float64(hits) / float64(hits+misses)
			b.ReportMetric(actualHitRate*100, "hit_rate_%")
		})
	}
}

// BenchmarkCacheWithAPISimulation tests cache performance with realistic API simulation
func BenchmarkCacheWithAPISimulation(b *testing.B) {
	latencies := []time.Duration{
		time.Millisecond,       // Fast API
		time.Millisecond * 10,  // Medium API
		time.Millisecond * 50,  // Slow API
		time.Millisecond * 100, // Very slow API
	}

	hitRates := []float64{0.5, 0.8, 0.95}

	for _, latency := range latencies {
		for _, hitRate := range hitRates {
			name := fmt.Sprintf("Latency_%s_HitRate_%.0f", latency, hitRate*100)
			b.Run(name, func(b *testing.B) {
				cache := NewTTLCache(time.Minute, 1000)
				defer cache.Close()

				keyPool := 100
				hitKeys := int(float64(keyPool) * hitRate)

				b.ResetTimer()
				totalLatency := time.Duration(0)

				for i := 0; i < b.N; i++ {
					var key string
					var id int

					if rand.Float64() < hitRate && hitKeys > 0 { //nolint:gosec // Using weak random for benchmark is acceptable
						id = rand.Intn(hitKeys) //nolint:gosec // Using weak random for benchmark is acceptable
						key = fmt.Sprintf("api_item_%d", id)
					} else {
						id = rand.Intn(keyPool) + keyPool //nolint:gosec // Using weak random for benchmark is acceptable
						key = fmt.Sprintf("api_item_%d", id)
					}

					start := time.Now()
					_, err := cache.GetOrLoad(key, func() (interface{}, error) {
						return simulateAPICall(id, latency)
					})
					elapsed := time.Since(start)
					totalLatency += elapsed

					if err != nil {
						b.Errorf("API call failed: %v", err)
					}
				}

				avgLatency := totalLatency / time.Duration(b.N)
				b.ReportMetric(float64(avgLatency.Nanoseconds())/1e6, "avg_latency_ms")

				// Report cache stats
				hits, misses, _, actualHitRate := cache.Stats()
				b.ReportMetric(actualHitRate*100, "actual_hit_rate_%")
				b.ReportMetric(float64(hits), "cache_hits")
				b.ReportMetric(float64(misses), "cache_misses")
			})
		}
	}
}

// BenchmarkCacheConcurrency tests cache performance under concurrent load
func BenchmarkCacheConcurrency(b *testing.B) {
	goroutineCounts := []int{1, 5, 10, 20, 50}

	for _, goroutines := range goroutineCounts {
		b.Run(fmt.Sprintf("Goroutines_%d", goroutines), func(b *testing.B) {
			cache := NewTTLCache(time.Minute, 1000)
			defer cache.Close()

			// Pre-populate cache
			for i := 0; i < 100; i++ {
				cache.Set(fmt.Sprintf("concurrent_key_%d", i), fmt.Sprintf("value_%d", i))
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					switch rand.Intn(3) { //nolint:gosec // Using weak random for benchmark is acceptable
					case 0: // Read (70% of operations)
						key := fmt.Sprintf("concurrent_key_%d", rand.Intn(100)) //nolint:gosec // Using weak random for benchmark is acceptable
						cache.Get(key)
					case 1: // Write (20% of operations)
						key := fmt.Sprintf("concurrent_key_%d", rand.Intn(150))      //nolint:gosec // Using weak random for benchmark is acceptable
						cache.Set(key, fmt.Sprintf("new_value_%d", rand.Intn(1000))) //nolint:gosec // Using weak random for benchmark is acceptable
					case 2: // GetOrLoad (10% of operations)
						key := fmt.Sprintf("concurrent_key_%d", rand.Intn(120)) //nolint:gosec // Using weak random for benchmark is acceptable
						_, _ = cache.GetOrLoad(key, func() (interface{}, error) {
							return fmt.Sprintf("loaded_%d", rand.Intn(1000)), nil //nolint:gosec // Using weak random for benchmark is acceptable
						})
					}
				}
			})
		})
	}
}

// BenchmarkCacheMemoryUsage tests memory efficiency
func BenchmarkCacheMemoryUsage(b *testing.B) {
	scenarios := []struct {
		name     string
		maxSize  int
		dataSize string
	}{
		{"Small_Cache_Small_Data", 100, "small"},
		{"Small_Cache_Large_Data", 100, "large"},
		{"Large_Cache_Small_Data", 10000, "small"},
		{"Large_Cache_Large_Data", 10000, "large"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			cache := NewTTLCache(time.Minute, scenario.maxSize)
			defer cache.Close()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("mem_key_%d", i%scenario.maxSize)

				var value interface{}
				switch scenario.dataSize {
				case "small":
					value = fmt.Sprintf("value_%d", i)
				case "large":
					// Create larger data structure
					data := make(map[string]interface{})
					for j := 0; j < 100; j++ {
						data[fmt.Sprintf("field_%d", j)] = fmt.Sprintf("data_%d_%d", i, j)
					}
					value = data
				}

				cache.Set(key, value)

				// Occasionally read to trigger cleanup
				if i%100 == 0 {
					cache.Get(key)
				}
			}
		})
	}
}

// BenchmarkCacheExpiration tests expiration and cleanup performance
func BenchmarkCacheExpiration(b *testing.B) {
	ttls := []time.Duration{
		time.Millisecond * 10,
		time.Millisecond * 100,
		time.Second,
	}

	for _, ttl := range ttls {
		b.Run(fmt.Sprintf("TTL_%s", ttl), func(b *testing.B) {
			cache := NewTTLCache(ttl, 1000)
			defer cache.Close()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("expire_key_%d", i%100)
				cache.Set(key, fmt.Sprintf("value_%d", i))

				// Simulate mixed read/write pattern
				if i%10 == 0 {
					// Wait for some entries to expire
					time.Sleep(ttl / 2)

					// Try to read expired and non-expired entries
					for j := 0; j < 10; j++ {
						readKey := fmt.Sprintf("expire_key_%d", (i-j)%100)
						cache.Get(readKey)
					}
				}
			}

			// Report final cache stats
			hits, misses, size, hitRate := cache.Stats()
			b.ReportMetric(float64(size), "final_cache_size")
			b.ReportMetric(hitRate*100, "hit_rate_%")
			b.ReportMetric(float64(hits), "total_hits")
			b.ReportMetric(float64(misses), "total_misses")
		})
	}
}

// BenchmarkCacheEviction tests eviction performance when cache is full
func BenchmarkCacheEviction(b *testing.B) {
	cacheSizes := []int{10, 100, 1000}

	for _, size := range cacheSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			cache := NewTTLCache(time.Minute, size)
			defer cache.Close()

			// Fill cache beyond capacity to trigger evictions
			for i := 0; i < size*2; i++ {
				key := fmt.Sprintf("evict_key_%d", i)
				cache.Set(key, fmt.Sprintf("value_%d", i))
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("evict_key_%d", i+size*2)
				cache.Set(key, fmt.Sprintf("new_value_%d", i))

				// Occasionally check if old entries were evicted
				if i%10 == 0 {
					oldKey := fmt.Sprintf("evict_key_%d", i/2)
					cache.Get(oldKey)
				}
			}

			// Verify cache stayed within size limits
			_, _, currentSize, _ := cache.Stats()
			if currentSize > size {
				b.Errorf("Cache size exceeded limit: %d > %d", currentSize, size)
			}
		})
	}
}

// BenchmarkCacheStatsOverhead tests the overhead of statistics collection
func BenchmarkCacheStatsOverhead(b *testing.B) {
	cache := NewTTLCache(time.Minute, 1000)
	defer cache.Close()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("stats_key_%d", i), fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _, _ = cache.Stats()
	}
}

// BenchmarkCacheWithJSONSerialization tests cache with realistic JSON data
func BenchmarkCacheWithJSONSerialization(b *testing.B) {
	cache := NewTTLCache(time.Minute, 1000)
	defer cache.Close()

	complexData := map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "John", "email": "john@example.com", "active": true},
			{"id": 2, "name": "Jane", "email": "jane@example.com", "active": false},
		},
		"metadata": map[string]interface{}{
			"version": "1.0",
			"created": time.Now(),
			"tags":    []string{"test", "benchmark", "cache"},
		},
		"settings": map[string]interface{}{
			"timeout":  30,
			"retries":  3,
			"debug":    true,
			"features": []string{"feature1", "feature2", "feature3"},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("json_key_%d", i%100)

		// Simulate API call that returns JSON data
		_, err := cache.GetOrLoad(key, func() (interface{}, error) {
			// Serialize and deserialize to simulate real JSON API
			jsonData, err := json.Marshal(complexData)
			if err != nil {
				return nil, err
			}

			var result map[string]interface{}
			err = json.Unmarshal(jsonData, &result)
			return result, err
		})
		if err != nil {
			b.Errorf("JSON processing failed: %v", err)
		}
	}

	// Report final stats
	hits, misses, _, hitRate := cache.Stats()
	b.ReportMetric(hitRate*100, "hit_rate_%")
	b.ReportMetric(float64(hits), "cache_hits")
	b.ReportMetric(float64(misses), "cache_misses")
}
