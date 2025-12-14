//go:build integration || performance

package integration

import "runtime"

// isRaceEnabled returns true if the race detector is enabled
func isRaceEnabled() bool {
	// The race detector is enabled when the -race flag is used
	// We can detect this by checking if the runtime/race package is active
	// This is a build-time constant that gets set when -race is used
	return runtime.GOOS != "js" && runtime.GOOS != "wasm" && getRaceEnabled()
}

// getRaceEnabled returns the race detection status
// This is set by the race_enabled.go file when built with -race
func getRaceEnabled() bool {
	return raceEnabledFlag
}

// raceEnabledFlag is a build tag constant that's true when -race is enabled
//
//nolint:gochecknoglobals // Required for race detection at build time
var raceEnabledFlag = false
