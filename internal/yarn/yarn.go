// Package yarn implements the "yarn" subcommand for mirrorctl,
// managing Yarn registry configuration via ~/.yarnrc.yml (Berry format).
package yarn

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

func configPath() string { return util.ExpandHome("~/.yarnrc.yml") }

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yarn",
		Short: "Switch Yarn registry mirror (.yarnrc.yml)",
		Long:  "Manage Yarn (Berry) registry mirror configuration via ~/.yarnrc.yml.",
		Example: `  mirrorctl yarn config          Show current Yarn registry
  mirrorctl yarn list            List all available yarn mirrors
  mirrorctl yarn set npmmirror   Switch to npmmirror
  mirrorctl yarn unset           Restore from backup`,
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
			Use: "config", Short: "Show current Yarn registry configuration",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdConfig()) },
		},
		&cobra.Command{
			Use: "list", Short: "List all supported Yarn mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdList()) },
		},
		&cobra.Command{
			Use: "set <name>", Short: "Set Yarn registry mirror (e.g. npmmirror / tuna)",
			Args: cobra.ExactArgs(1), Run: func(_ *cobra.Command, args []string) { os.Exit(cmdSet(args[0])) },
		},
		&cobra.Command{
			Use: "unset", Short: "Restore Yarn registry from backup (.bak)",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdUnset()) },
		},
		&cobra.Command{
			Use: "test", Short: "Test connectivity and latency of all Yarn mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdTest()) },
		},
	)
	return cmd
}

// --- config ---

func cmdConfig() int {
	path := configPath()
	if !util.FileExists(path) {
		fmt.Println("No .yarnrc.yml found (using yarn default registry).")
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

	// Try to find npmRegistryServer
	url := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "npmRegistryServer:") {
			url = strings.TrimSpace(strings.TrimPrefix(line, "npmRegistryServer:"))
			url = strings.Trim(url, `"'`)
		}
	}
	if url == "" {
		return 0
	}
	for _, m := range Mirrors {
		if m.URL == url {
			fmt.Printf("Mirror: %s (%s)\n", m.Name, m.Desc)
			return 0
		}
	}
	fmt.Println("(not in mirrorctl's known mirror list)")
	return 0
}

// --- list ---

func cmdList() int {
	fmt.Println("Available Yarn mirrors:")
	fmt.Println()
	fmt.Printf("  %-10s %-48s %s\n", "name", "registry URL", "Description")
	fmt.Printf("  %-10s %-48s %s\n", "----", "-------------", "-----------")
	for _, m := range Mirrors {
		fmt.Printf("  %-10s %-48s %s\n", m.Name, m.URL, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl yarn set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	m := findMirror(name)
	if m == nil {
		util.Warn("unknown yarn mirror: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors { names[i] = mir.Name }
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl yarn list` for details.")
		return 1
	}

	path := configPath()
	util.MkdirAll(util.ExpandHome("~"))
	entry := fmt.Sprintf("npmRegistryServer: \"%s\"\n", m.URL)

	var buf strings.Builder
	replaced := false
	if data, _ := util.ReadFile(path); data != nil {
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "npmRegistryServer:") {
				fmt.Fprint(&buf, entry)
				replaced = true
			} else {
				fmt.Fprintln(&buf, line)
			}
		}
	}
	if !replaced {
		fmt.Fprint(&buf, entry)
	}

	util.WriteFileAtomic(path, []byte(buf.String()))
	fmt.Printf("Yarn registry set to %s (%s)\n", m.Name, m.Desc)
	fmt.Printf("Config: %s\n", path)
	return 0
}

// --- unset ---

func cmdUnset() int {
	path := configPath()
	if !util.FileExists(path) {
		fmt.Println("No .yarnrc.yml found, nothing to remove.")
		return 0
	}

	bak, err := util.BackupFile(path)
	if err != nil {
		util.Warn("backup failed: %v", err)
		return 1
	}

	var buf strings.Builder
	data, _ := util.ReadFile(path)
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "npmRegistryServer:") {
			fmt.Fprintln(&buf, line)
		}
	}
	util.WriteFileAtomic(path, []byte(strings.TrimSpace(buf.String())))

	fmt.Printf("Removed yarn registry setting.\n")
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
	util.PrintTestResults("Testing Yarn mirrors...", results)
	return 0
}
