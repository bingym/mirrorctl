// Package pnpm implements the "pnpm" subcommand for mirrorctl,
// managing pnpm registry configuration via `pnpm config`.
package pnpm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

type Mirror struct {
	Name string
	URL  string
	Desc string
}

var Mirrors = []Mirror{
	{
		Name: "npmmirror",
		URL:  "https://registry.npmmirror.com/",
		Desc: "npmmirror (formerly Taobao)",
	},
	{
		Name: "tuna",
		URL:  "https://mirrors.tuna.tsinghua.edu.cn/npm/",
		Desc: "Tsinghua University TUNA",
	},
	{
		Name: "huawei",
		URL:  "https://repo.huaweicloud.com/repository/npm/",
		Desc: "Huawei Cloud",
	},
	{
		Name: "tencent",
		URL:  "https://mirrors.tencent.com/npm/",
		Desc: "Tencent Cloud",
	},
}

func findMirror(name string) *Mirror {
	for i := range Mirrors {
		if Mirrors[i].Name == name {
			return &Mirrors[i]
		}
	}
	return nil
}

func ensurePNPM() {
	if _, err := exec.LookPath("pnpm"); err != nil {
		util.Die("pnpm not found in PATH; install pnpm first")
	}
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pnpm",
		Short: "Switch pnpm registry mirror",
		Long:  "Manage pnpm registry mirror configuration via `pnpm config`.",
		Example: `  mirrorctl pnpm config          Show current pnpm registry
  mirrorctl pnpm list            List all available pnpm mirrors
  mirrorctl pnpm set npmmirror   Switch to npmmirror
  mirrorctl pnpm unset           Restore default`,
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
			Use: "config", Short: "Show current pnpm registry configuration",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdConfig()) },
		},
		&cobra.Command{
			Use: "list", Short: "List all supported pnpm mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdList()) },
		},
		&cobra.Command{
			Use: "set <name>", Short: "Set pnpm registry mirror (e.g. npmmirror / tuna)",
			Args: cobra.ExactArgs(1), Run: func(_ *cobra.Command, args []string) { os.Exit(cmdSet(args[0])) },
		},
		&cobra.Command{
			Use: "unset", Short: "Remove pnpm registry setting",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdUnset()) },
		},
		&cobra.Command{
			Use: "test", Short: "Test connectivity and latency of all pnpm mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdTest()) },
		},
	)
	return cmd
}

// --- config ---

func cmdConfig() int {
	ensurePNPM()
	out, err := exec.Command("pnpm", "config", "get", "registry").Output()
	if err != nil {
		util.Warn("cannot read pnpm registry: %v", err)
		return 1
	}
	reg := strings.TrimSpace(string(out))
	fmt.Printf("Current pnpm registry: %s\n", reg)
	if reg == "https://registry.npmjs.org/" {
		fmt.Println("(pnpm default registry)")
		return 0
	}
	for _, m := range Mirrors {
		if m.URL == reg {
			fmt.Printf("Mirror: %s (%s)\n", m.Name, m.Desc)
			return 0
		}
	}
	fmt.Println("(not in mirrorctl's known mirror list)")
	return 0
}

// --- list ---

func cmdList() int {
	fmt.Println("Available pnpm mirrors:")
	fmt.Println()
	fmt.Printf("  %-10s %-48s %s\n", "name", "registry URL", "Description")
	fmt.Printf("  %-10s %-48s %s\n", "----", "-------------", "-----------")
	for _, m := range Mirrors {
		fmt.Printf("  %-10s %-48s %s\n", m.Name, m.URL, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl pnpm set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	ensurePNPM()
	m := findMirror(name)
	if m == nil {
		util.Warn("unknown pnpm mirror: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors { names[i] = mir.Name }
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl pnpm list` for details.")
		return 1
	}
	cmd := exec.Command("pnpm", "config", "set", "registry", m.URL)
	if out, err := cmd.CombinedOutput(); err != nil {
		util.Warn("pnpm config set failed: %v\n%s", err, string(out))
		return 1
	}
	fmt.Printf("pnpm registry set to %s (%s)\n", m.Name, m.Desc)
	return 0
}

// --- unset ---

func cmdUnset() int {
	ensurePNPM()
	out, err := exec.Command("pnpm", "config", "get", "registry").Output()
	if err != nil {
		util.Warn("cannot read pnpm registry: %v", err)
		return 1
	}
	current := strings.TrimSpace(string(out))
	if current == "https://registry.npmjs.org/" {
		fmt.Println("pnpm registry is already the default, nothing to unset.")
		return 0
	}
	cmd := exec.Command("pnpm", "config", "delete", "registry")
	if out, err := cmd.CombinedOutput(); err != nil {
		util.Warn("pnpm config delete failed: %v\n%s", err, string(out))
		return 1
	}
	fmt.Printf("Removed pnpm registry setting (was %s)\n", current)
	return 0
}

// --- test ---

func cmdTest() int {
	targets := make([]util.MirrorTarget, len(Mirrors))
	for i, m := range Mirrors {
		targets[i] = util.MirrorTarget{Name: m.Name, Desc: m.Desc, URL: m.URL}
	}
	results := util.TestMirrors(targets)
	util.PrintTestResults("Testing pnpm mirrors...", results)
	return 0
}
