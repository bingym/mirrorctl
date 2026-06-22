// mirrorctl is a CLI tool for switching package manager mirror sources.
//
// Usage:
//
//	mirrorctl <type> <action> [value]
//
// Example:
//
//	mirrorctl pypi set tuna
package main

import (
	"os"

	"github.com/bingym/mirrorctl/internal/docker"
	"github.com/bingym/mirrorctl/internal/github"
	"github.com/bingym/mirrorctl/internal/goproxy"
	"github.com/bingym/mirrorctl/internal/pypi"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "mirrorctl",
	Short: "Switch package manager mirror sources",
	Long:  "mirrorctl - switch package manager mirror sources.\n\nUse subcommands to manage mirrors for different package managers.",
	Example: `  mirrorctl pypi config          Show current pip mirror
  mirrorctl pypi list            List all available PyPI mirrors
  mirrorctl pypi set tuna        Switch to Tsinghua mirror
  mirrorctl pypi unset           Restore default (backup to .bak)`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(pypi.Command())
	rootCmd.AddCommand(goproxy.Command())
	rootCmd.AddCommand(github.Command())
	rootCmd.AddCommand(docker.Command())
	rootCmd.SetVersionTemplate("mirrorctl {{.Version}}\n")
	rootCmd.Version = version
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		rootCmd.PrintErrln(err)
		os.Exit(1)
	}
}
