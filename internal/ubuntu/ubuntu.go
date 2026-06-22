// Package ubuntu implements the "ubuntu" subcommand for mirrorctl,
// managing Ubuntu APT mirror configuration.
package ubuntu

import (
	"fmt"
	"os"
	"strings"

	"github.com/bingym/mirrorctl/internal/apt"
	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

// Mirrors is the static table of known Ubuntu APT mirrors.
var Mirrors = []apt.Mirror{
	{
		Name:    "ustc",
		BaseURL: "https://mirrors.ustc.edu.cn",
		Desc:    "University of Science and Technology of China",
	},
	{
		Name:    "tuna",
		BaseURL: "https://mirrors.tuna.tsinghua.edu.cn",
		Desc:    "Tsinghua University TUNA",
	},
	{
		Name:    "aliyun",
		BaseURL: "https://mirrors.aliyun.com",
		Desc:    "Alibaba Cloud",
	},
	{
		Name:    "163",
		BaseURL: "https://mirrors.163.com",
		Desc:    "NetEase 163",
	},
	{
		Name:    "huawei",
		BaseURL: "https://mirrors.huaweicloud.com",
		Desc:    "Huawei Cloud",
	},
	{
		Name:    "tencent",
		BaseURL: "https://mirrors.tencent.com",
		Desc:    "Tencent Cloud",
	},
}

func findMirror(name string) *apt.Mirror {
	for i := range Mirrors {
		if Mirrors[i].Name == name {
			return &Mirrors[i]
		}
	}
	return nil
}

// Command returns the cobra command for the "ubuntu" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ubuntu",
		Short: "Switch Ubuntu APT mirror source",
		Long:  "Manage Ubuntu APT source configuration for package mirrors (sources.list / ubuntu.sources).",
		Example: `  mirrorctl ubuntu config          Show current Ubuntu mirror
  mirrorctl ubuntu list            List all available Ubuntu mirrors
  mirrorctl ubuntu set tuna        Switch to Tsinghua TUNA mirror
  mirrorctl ubuntu unset           Restore from backup`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "unknown command %q for %q\n\n", args[0], cmd.CommandPath())
				cmd.Help()
				os.Exit(1)
			}
			cmd.Help()
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "config",
			Short: "Show current Ubuntu APT mirror configuration",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdConfig())
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all supported Ubuntu mirrors",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdList())
			},
		},
		&cobra.Command{
			Use:   "set <name>",
			Short: "Set Ubuntu APT mirror (e.g. ustc / tuna / aliyun)",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdSet(args[0]))
			},
		},
		&cobra.Command{
			Use:   "unset",
			Short: "Restore Ubuntu APT mirror from backup (.bak)",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdUnset())
			},
		},
		&cobra.Command{
			Use:   "test",
			Short: "Test connectivity and latency of all Ubuntu mirrors",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdTest())
			},
		},
	)

	return cmd
}

// --- config ---

func cmdConfig() int {
	path, _, err := apt.DetectActiveSource(apt.Ubuntu)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}
	apt.ShowConfig(path, apt.Ubuntu, Mirrors)
	return 0
}

// --- list ---

func cmdList() int {
	fmt.Println("Available Ubuntu mirrors:")
	fmt.Println()
	fmt.Printf("  %-8s %-40s %s\n", "name", "URL", "Description")
	fmt.Printf("  %-8s %-40s %s\n", "----", "---", "-----------")
	for _, m := range Mirrors {
		url := m.BaseURL + "/ubuntu/"
		fmt.Printf("  %-8s %-40s %s\n", m.Name, url, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl ubuntu set <name>` to switch mirrors.")
	fmt.Println("Note: requires root privileges (sudo).")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	if !apt.CheckRoot() {
		util.Warn("root privileges required; try running with sudo")
		return 1
	}

	m := findMirror(name)
	if m == nil {
		util.Warn("unknown Ubuntu mirror name: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors {
			names[i] = mir.Name
		}
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl ubuntu list` for details.")
		return 1
	}

	path, _, err := apt.DetectActiveSource(apt.Ubuntu)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}

	bak, err := apt.BackupSources(path)
	if err != nil {
		util.Warn("backup failed: %v", err)
		return 1
	}
	fmt.Printf("Backed up to %s\n", bak)

	if err := apt.ReplaceMirror(path, apt.Ubuntu, *m); err != nil {
		util.Warn("failed to set mirror: %v", err)
		return 1
	}
	fmt.Printf("Ubuntu APT mirror set to %s (%s)\n", m.Name, m.Desc)

	fmt.Println("Running apt-get update...")
	if err := apt.RunUpdate(); err != nil {
		util.Warn("apt-get update failed: %v", err)
		return 1
	}
	fmt.Println("Done.")
	return 0
}

// --- unset ---

func cmdUnset() int {
	if !apt.CheckRoot() {
		util.Warn("root privileges required; try running with sudo")
		return 1
	}

	path, _, err := apt.DetectActiveSource(apt.Ubuntu)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}

	if err := apt.RestoreSources(path); err != nil {
		util.Warn("restore failed: %v", err)
		return 1
	}
	fmt.Printf("Restored %s from backup.\n", path)

	fmt.Println("Running apt-get update...")
	if err := apt.RunUpdate(); err != nil {
		util.Warn("apt-get update failed: %v", err)
		return 1
	}
	fmt.Println("Done.")
	return 0
}

// --- test ---

func cmdTest() int {
	targets := make([]util.MirrorTarget, len(Mirrors))
	for i, m := range Mirrors {
		targets[i] = util.MirrorTarget{Name: m.Name, Desc: m.Desc, URL: m.BaseURL + "/ubuntu/"}
	}
	results := util.TestMirrors(targets)
	util.PrintTestResults("Testing Ubuntu mirrors...", results)
	return 0
}
