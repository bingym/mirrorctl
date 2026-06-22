// Package pypi implements the "pypi" subcommand for mirrorctl,
// managing pip / PyPI mirror configurations.
package pypi

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

// Mirror represents a single PyPI mirror source.
type Mirror struct {
	Name        string // short name, e.g. "tuna"
	URL         string // full index-url
	TrustedHost string // hostname for trusted-host (may be empty)
	Desc        string // human-readable description
}

// Mirrors is the static table of known PyPI mirrors.
// Add new mirrors here; all logic below is table-driven.
var Mirrors = []Mirror{
	{
		Name:        "tuna",
		URL:         "https://pypi.tuna.tsinghua.edu.cn/simple/",
		TrustedHost: "pypi.tuna.tsinghua.edu.cn",
		Desc:        "Tsinghua University TUNA",
	},
	{
		Name:        "aliyun",
		URL:         "https://mirrors.aliyun.com/pypi/simple/",
		TrustedHost: "mirrors.aliyun.com",
		Desc:        "Alibaba Cloud",
	},
	{
		Name:        "ustc",
		URL:         "https://pypi.mirrors.ustc.edu.cn/simple/",
		TrustedHost: "pypi.mirrors.ustc.edu.cn",
		Desc:        "University of Science and Technology of China",
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

// configPath returns the XDG pip config path: ~/.config/pip/pip.conf
func configPath() string {
	return util.ExpandHome("~/.config/pip/pip.conf")
}

// legacyPath returns the legacy pip config path: ~/.pip/pip.conf
func legacyPath() string {
	return util.ExpandHome("~/.pip/pip.conf")
}

// Command returns the cobra command for the "pypi" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pypi",
		Short: "Switch pip / PyPI mirror source",
		Long:  "Manage pip / PyPI mirror configuration.",
		Example: `  mirrorctl pypi config
  mirrorctl pypi set tuna
  mirrorctl pypi list
  mirrorctl pypi test
  mirrorctl pypi unset`,
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
			Short: "Show current PyPI mirror configuration",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdConfig())
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all supported PyPI mirrors",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdList())
			},
		},
		&cobra.Command{
			Use:   "set <name>",
			Short: "Set PyPI mirror (e.g. tuna / aliyun / ustc)",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdSet(args[0]))
			},
		},
		&cobra.Command{
			Use:   "unset",
			Short: "Remove PyPI mirror config (auto-backup to .bak)",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdUnset())
			},
		},
		&cobra.Command{
			Use:   "test",
			Short: "Test connectivity and latency of all PyPI mirrors",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdTest())
			},
		},
	)

	return cmd
}

// --- config ---

// extractIndexURL parses pip.conf content and returns the index-url
// under the [global] section, or "" if not found.
func extractIndexURL(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	inGlobal := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[") {
			inGlobal = line == "[global]"
			continue
		}

		if inGlobal && strings.HasPrefix(line, "index-url") {
			// Parse "index-url = <value>" or "index-url=<value>"
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func cmdConfig() int {
	path := configPath()
	legacy := legacyPath()

	if !util.FileExists(path) {
		fmt.Println("No PyPI mirror configured (using pip default index).")
		if util.FileExists(legacy) {
			fmt.Printf("Note: legacy config file %s detected; pip will still read it.\n", legacy)
		}
		return 0
	}

	fmt.Printf("Config file: %s\n\n", path)

	data, err := util.ReadFile(path)
	if err != nil {
		util.Warn("cannot read %s: %v", path, err)
		return 1
	}

	url := extractIndexURL(string(data))
	if url == "" {
		fmt.Println("Config file exists but no [global] index-url found.")
		return 0
	}

	fmt.Printf("Current index-url: %s\n", url)

	// Reverse-lookup mirror name
	found := false
	for _, m := range Mirrors {
		if m.URL == url {
			fmt.Printf("Mirror: %s (%s)\n", m.Name, m.Desc)
			found = true
			break
		}
	}
	if !found {
		fmt.Println("(not in mirrorctl's known mirror list)")
	}

	return 0
}

// --- list ---

func cmdList() int {
	fmt.Println("Available PyPI mirrors:")
	fmt.Println()
	fmt.Printf("  %-8s %-48s %s\n", "name", "index-url", "Description")
	fmt.Printf("  %-8s %-48s %s\n", "----", "---------", "-----------")
	for _, m := range Mirrors {
		fmt.Printf("  %-8s %-48s %s\n", m.Name, m.URL, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl pypi set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	m := findMirror(name)
	if m == nil {
		util.Warn("unknown PyPI mirror name: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors {
			names[i] = mir.Name
		}
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl pypi list` for details.")
		return 1
	}

	path := configPath()
	dir := util.ExpandHome("~/.config/pip")
	util.MkdirAll(dir)

	// Build config file content
	var buf strings.Builder
	fmt.Fprintf(&buf, "[global]\nindex-url = %s\n", m.URL)
	if m.TrustedHost != "" {
		fmt.Fprintf(&buf, "\n[install]\ntrusted-host = %s\n", m.TrustedHost)
	}

	util.WriteFileAtomic(path, []byte(buf.String()))

	fmt.Printf("PyPI mirror set to %s (%s)\n", m.Name, m.Desc)
	fmt.Printf("Config file: %s\n", path)
	fmt.Printf("index-url: %s\n", m.URL)

	return 0
}

// --- unset ---

func cmdUnset() int {
	path := configPath()

	if !util.FileExists(path) {
		fmt.Println("No PyPI mirror configured, nothing to remove.")
		return 0
	}

	bak, err := util.BackupFile(path)
	if err != nil {
		util.Warn("cannot backup %s: %v", path, err)
		return 1
	}

	if err := os.Remove(path); err != nil {
		util.DieErr("remove "+path, err)
	}

	fmt.Printf("Removed PyPI mirror config %s\n", path)
	fmt.Printf("Original file backed up to %s\n", bak)

	return 0
}

// --- test ---

// mirrorResult holds the latency test result for a single mirror.
type mirrorResult struct {
	name    string
	desc    string
	latency time.Duration
	err     error
}

// testMirror sends an HTTP HEAD request to the mirror and measures round-trip time.
func testMirror(m Mirror) mirrorResult {
	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	resp, err := client.Head(m.URL)
	elapsed := time.Since(start)
	if err != nil {
		return mirrorResult{name: m.Name, desc: m.Desc, err: err}
	}
	resp.Body.Close()
	return mirrorResult{name: m.Name, desc: m.Desc, latency: elapsed}
}

func cmdTest() int {
	fmt.Println("Testing PyPI mirrors...")
	fmt.Println()

	results := make([]mirrorResult, len(Mirrors))
	var wg sync.WaitGroup

	for i, m := range Mirrors {
		wg.Add(1)
		go func(idx int, mirror Mirror) {
			defer wg.Done()
			results[idx] = testMirror(mirror)
		}(i, m)
	}
	wg.Wait()

	// Sort: successful first by latency ascending, errors last (alphabetical)
	sort.SliceStable(results, func(i, j int) bool {
		ei, ej := results[i].err, results[j].err
		if ei != nil && ej != nil {
			return results[i].name < results[j].name
		}
		if ei != nil {
			return false
		}
		if ej != nil {
			return true
		}
		return results[i].latency < results[j].latency
	})

	fmt.Printf("  %-8s %-48s %s\n", "name", "description", "latency")
	fmt.Printf("  %-8s %-48s %s\n", "----", "-----------", "-------")
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("  %-8s %-48s %s\n", r.name, r.desc, r.err.Error())
		} else {
			ms := float64(r.latency) / float64(time.Millisecond)
			fmt.Printf("  %-8s %-48s %.0f ms\n", r.name, r.desc, ms)
		}
	}
	fmt.Println()
	return 0
}
