//go:build race
// +build race

package integration

func init() {
	raceEnabledFlag = true
}
