//go:build race

package integration

func init() {
	raceEnabledFlag = true
}
