//go:build race && (integration || performance)

package integration

func init() {
	raceEnabledFlag = true
}
