// Command ifaceguard runs the ifaceguard linter as a standalone tool.
//
// Usage:
//
//	ifaceguard [flags] [packages]
//
// For detailed flag descriptions, run:
//
//	ifaceguard -help
package main

import (
	"os"

	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/n-r-w/ifaceguard/internal/driver"
)

func main() {
	cfg := config.Default()
	a, err := analyzer.New(cfg)
	if err != nil {
		panic(err)
	}

	exitCode := driver.Run(a, driver.Options{
		Args:   os.Args[1:],
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	os.Exit(exitCode)
}
