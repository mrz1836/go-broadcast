// Package main provides the GoFortress coverage CLI tool
package main

import (
	"os"

	"github.com/mrz1836/go-broadcast/coverage/cmd/gofortress-coverage/cmd"
)

func main() { //nolint:revive // function naming
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
