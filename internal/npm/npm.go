// Package npm implements the "npm" subcommand for mirrorctl,
// managing npm registry configuration via ~/.npmrc.
package npm

import (
	"bufio"
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

func configPath() string   { return util.ExpandHome("~/.npmrc") }

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npm",
		Short: "Switch npm registry mirror",
		Long:  "Manage npm registry mirror configuration via ~/.npmrc.",
		Example: `  mirrorctl npm config          Show current npm registry
  mirrorctl npm list            List all available npm mirrors
  mirrorctl npm set npmmirror   Switch to npmmirror (Taobao)
  mirrorctl npm unset           Restore from backup`,
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
			Use: "config", Short: "Show current npm registry configuration",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdConfig()) },
		},
		&cobra.Command{
			Use: "list", Short: "List all supported npm mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdList()) },
		},
		&cobra.Command{
			Use: "set <name>", Short: "Set npm registry mirror (e.g. npmmirror / tuna)",
			Args: cobra.ExactArgs(1), Run: func(_ *cobra.Command, args []string) { os.Exit(cmdSet(args[0])) },
		},
		&cobra.Command{
			Use: "unset", Short: "Restore npm registry from backup (.bak)",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdUnset()) },
		},
		&cobra.Command{
			Use: "test", Short: "Test connectivity and latency of all npm mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdTest()) },
		},
	)
	return cmd
}

// --- config ---

func extractRegistry(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "registry=") {
			return strings.TrimSpace(line[len("registry="):])
		}
	}
	return ""
}

func cmdConfig() int {
	path := configPath()
	if !util.FileExists(path) {
		fmt.Println("No npm registry configured (using npm default).")
		return 0
	}
	data, err := util.ReadFile(path)
	if err != nil {
		util.Warn("cannot read %s: %v", path, err)
		return 1
	}
	reg := extractRegistry(string(data))
	fmt.Printf("Config file: %s\n", path)
	if reg == "" {
		fmt.Println("No registry setting found (using npm default).")
		return 0
	}
	fmt.Printf("registry: %s\n", reg)
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
	fmt.Println("Available npm mirrors:")
	fmt.Println()
	fmt.Printf("  %-10s %-48s %s\n", "name", "registry URL", "Description")
	fmt.Printf("  %-10s %-48s %s\n", "----", "-------------", "-----------")
	for _, m := range Mirrors {
		fmt.Printf("  %-10s %-48s %s\n", m.Name, m.URL, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl npm set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	m := findMirror(name)
	if m == nil {
		util.Warn("unknown npm mirror: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors { names[i] = mir.Name }
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl npm list` for details.")
		return 1
	}

	path := configPath()
	util.MkdirAll(util.ExpandHome("~"))

	// Read existing .npmrc, update or append registry line
	var buf strings.Builder
	replaced := false
	if data, _ := util.ReadFile(path); data != nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(strings.TrimSpace(line), "registry=") {
				fmt.Fprintf(&buf, "registry=%s\n", m.URL)
				replaced = true
			} else {
				fmt.Fprintln(&buf, line)
			}
		}
	}
	if !replaced {
		fmt.Fprintf(&buf, "registry=%s\n", m.URL)
	}

	util.WriteFileAtomic(path, []byte(buf.String()))
	fmt.Printf("npm registry set to %s (%s)\n", m.Name, m.Desc)
	fmt.Printf("Config: %s\n", path)
	return 0
}

// --- unset ---

func cmdUnset() int {
	path := configPath()
	if !util.FileExists(path) {
		fmt.Println("No npm config found, nothing to remove.")
		return 0
	}

	bak, err := util.BackupFile(path)
	if err != nil {
		util.Warn("backup failed: %v", err)
		return 1
	}

	// Remove registry line, keep other settings
	var buf strings.Builder
	data, _ := util.ReadFile(path)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "registry=") {
			fmt.Fprintln(&buf, line)
		}
	}
	util.WriteFileAtomic(path, []byte(strings.TrimSpace(buf.String())))

	fmt.Printf("Removed npm registry setting.\n")
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
	util.PrintTestResults("Testing npm mirrors...", results)
	return 0
}
