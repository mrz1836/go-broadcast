//go:build mage

// Magefile for go-broadcast specific tasks
package main

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

// BenchHeavy runs intensive benchmarks excluded from CI
func BenchHeavy() error {
	fmt.Println("🏋️ Running heavy benchmarks (may take 10-30 minutes)...")
	fmt.Println("These benchmarks include:")
	fmt.Println("- Worker pool stress tests (1000+ concurrent tasks)")
	fmt.Println("- Large directory sync simulations")
	fmt.Println("- Memory efficiency tests with large datasets")
	fmt.Println("- Real-world scenario simulations")
	fmt.Println("")

	return sh.RunV("go", "test", "-bench=.", "-benchmem",
		"-tags=bench_heavy", "-benchtime=1s", "-timeout=60m", "./...")
}

// BenchAll runs all benchmarks including heavy ones
func BenchAll() error {
	fmt.Println("🎯 Running all benchmarks (may take 30-60 minutes)...")
	fmt.Println("")

	// Run default benchmarks first
	fmt.Println("1/2: Running quick benchmarks...")
	if err := sh.RunV("go", "test", "-bench=.", "-benchmem",
		"-benchtime=100ms", "-timeout=20m", "./..."); err != nil {
		return fmt.Errorf("quick benchmarks failed: %w", err)
	}

	fmt.Println("")
	fmt.Println("2/2: Running heavy benchmarks...")
	// Then run heavy benchmarks
	return BenchHeavy()
}

// BenchQuick runs only the quick benchmarks (same as magex bench)
func BenchQuick() error {
	fmt.Println("⚡ Running quick benchmarks only...")
	return sh.RunV("go", "test", "-bench=.", "-benchmem",
		"-benchtime=100ms", "-timeout=20m", "./...")
}

