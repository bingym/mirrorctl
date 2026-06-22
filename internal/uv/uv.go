// Package uv implements the "uv" subcommand for mirrorctl,
// managing uv Python package index configuration via ~/.config/uv/uv.toml.
package uv

import (
	"fmt"
	"os"
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
		Name: "tuna",
		URL:  "https://pypi.tuna.tsinghua.edu.cn/simple/",
		Desc: "Tsinghua University TUNA",
	},
	{
		Name: "aliyun",
		URL:  "https://mirrors.aliyun.com/pypi/simple/",
		Desc: "Alibaba Cloud",
	},
	{
		Name: "ustc",
		URL:  "https://pypi.mirrors.ustc.edu.cn/simple/",
		Desc: "University of Science and Technology of China",
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

func configDir() string  { return util.ExpandHome("~/.config/uv") }
func configPath() string { return configDir() + "/uv.toml" }

// uvConfig constructs the uv.toml content that sets the PyPI index URL.
func uvConfig(m Mirror) string {
	return fmt.Sprintf(`[[index]]
url = "%s"
`, m.URL)
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uv",
		Short: "Switch uv Python package index mirror",
		Long:  "Manage uv Python package registry mirror configuration via ~/.config/uv/uv.toml.",
		Example: `  mirrorctl uv config          Show current uv index configuration
  mirrorctl uv list            List all available uv mirrors
  mirrorctl uv set tuna        Switch to TUNA mirror
  mirrorctl uv unset           Restore from backup`,
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
			Use: "config", Short: "Show current uv index configuration",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdConfig()) },
		},
		&cobra.Command{
			Use: "list", Short: "List all supported uv mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdList()) },
		},
		&cobra.Command{
			Use: "set <name>", Short: "Set uv PyPI index mirror (e.g. tuna / aliyun / ustc)",
			Args: cobra.ExactArgs(1), Run: func(_ *cobra.Command, args []string) { os.Exit(cmdSet(args[0])) },
		},
		&cobra.Command{
			Use: "unset", Short: "Restore uv configuration from backup (.bak)",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdUnset()) },
		},
		&cobra.Command{
			Use: "test", Short: "Test connectivity and latency of all uv mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdTest()) },
		},
	)
	return cmd
}

// --- config ---

func cmdConfig() int {
	path := configPath()
	if !util.FileExists(path) {
		fmt.Println("No uv.toml found (using uv default index).")
		return 0
	}
	data, err := util.ReadFile(path)
	if err != nil {
		util.Warn("cannot read %s: %v", path, err)
		return 1
	}
	fmt.Printf("Config file: %s\n\n", path)
	fmt.Print(string(data))
	fmt.Println()

	// Try to find index URL
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "url =") {
			url := strings.TrimSpace(strings.TrimPrefix(trimmed, "url ="))
			url = strings.Trim(url, `"'`)
			for _, m := range Mirrors {
				if m.URL == url {
					fmt.Printf("Mirror: %s (%s)\n", m.Name, m.Desc)
					return 0
				}
			}
			fmt.Println("(not in mirrorctl's known mirror list)")
			return 0
		}
	}
	return 0
}

// --- list ---

func cmdList() int {
	fmt.Println("Available uv mirrors:")
	fmt.Println()
	fmt.Printf("  %-8s %-52s %s\n", "name", "index URL", "Description")
	fmt.Printf("  %-8s %-52s %s\n", "----", "---------", "-----------")
	for _, m := range Mirrors {
		fmt.Printf("  %-8s %-52s %s\n", m.Name, m.URL, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl uv set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	m := findMirror(name)
	if m == nil {
		util.Warn("unknown uv mirror: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors { names[i] = mir.Name }
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl uv list` for details.")
		return 1
	}

	util.MkdirAll(configDir())
	util.WriteFileAtomic(configPath(), []byte(uvConfig(*m)))
	fmt.Printf("uv index set to %s (%s)\n", m.Name, m.Desc)
	fmt.Printf("Config: %s\n", configPath())
	return 0
}

// --- unset ---

func cmdUnset() int {
	path := configPath()
	if !util.FileExists(path) {
		fmt.Println("No uv.toml found, nothing to remove.")
		return 0
	}

	bak, err := util.BackupFile(path)
	if err != nil {
		util.Warn("backup failed: %v", err)
		return 1
	}

	if err := os.Remove(path); err != nil {
		util.DieErr("remove "+path, err)
	}

	fmt.Printf("Removed uv config %s\n", path)
	fmt.Printf("Backup: %s\n", bak)
	return 0
}

// --- test ---

func cmdTest() int {
	targets := make([]util.MirrorTarget, len(Mirrors))
	for i, m := range Mirrors {
		targets[i] = util.MirrorTarget{Name: m.Name, Desc: m.Desc, URL: m.URL}
	}
	results := util.TestMirrors(targets)
	util.PrintTestResults("Testing uv mirrors...", results)
	return 0
}
