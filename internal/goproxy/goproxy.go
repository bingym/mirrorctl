// Package goproxy implements the "goproxy" subcommand for mirrorctl,
// managing Go module proxy (GOPROXY) configuration via `go env`.
package goproxy

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

// Mirror represents a single Go module proxy mirror.
type Mirror struct {
	Name string // short name, e.g. "goproxy.io"
	URL  string // base proxy URL (without ,direct suffix)
	Desc string // human-readable description
}

// Mirrors is the static table of known Go module proxy mirrors.
// Add new mirrors here; all logic below is table-driven.
var Mirrors = []Mirror{
	{
		Name: "goproxy.io",
		URL:  "https://goproxy.io",
		Desc: "goproxy.io - Global Go module proxy",
	},
	{
		Name: "goproxy.cn",
		URL:  "https://goproxy.cn",
		Desc: "goproxy.cn - China Go module proxy (by Qiniu)",
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

// ensureGo checks that `go` is available in PATH.
func ensureGo() {
	_, err := exec.LookPath("go")
	if err != nil {
		util.Die("go command not found in PATH; Go toolchain is required for goproxy operations")
	}
}

// Command returns the cobra command for the "goproxy" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goproxy",
		Short: "Switch Go module proxy (GOPROXY)",
		Long:  "Manage Go module proxy configuration via `go env -w GOPROXY=...`.",
		Example: `  mirrorctl goproxy config
  mirrorctl goproxy set goproxy.cn
  mirrorctl goproxy set goproxy.io
  mirrorctl goproxy list
  mirrorctl goproxy test
  mirrorctl goproxy unset`,
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
			Short: "Show current GOPROXY setting",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdConfig())
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all supported Go module proxy mirrors",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdList())
			},
		},
		&cobra.Command{
			Use:   "set <name>",
			Short: "Set GOPROXY mirror (e.g. goproxy.cn / goproxy.io)",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdSet(args[0]))
			},
		},
		&cobra.Command{
			Use:   "unset",
			Short: "Remove GOPROXY setting (go env -u GOPROXY)",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdUnset())
			},
		},
		&cobra.Command{
			Use:   "test",
			Short: "Test connectivity and latency of all Go module proxy mirrors",
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
	ensureGo()

	out, err := exec.Command("go", "env", "GOPROXY").Output()
	if err != nil {
		util.Warn("cannot read GOPROXY: %v", err)
		return 1
	}

	goproxy := strings.TrimSpace(string(out))
	fmt.Printf("Current GOPROXY: %s\n", goproxy)

	// If it's the default, note it
	if goproxy == "https://proxy.golang.org,direct" {
		fmt.Println("(Go default proxy)")
		return 0
	}

	// Reverse-lookup mirror name: extract base URL before ",direct"
	base := strings.TrimSuffix(goproxy, ",direct")
	found := false
	for _, m := range Mirrors {
		if m.URL == base {
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
	fmt.Println("Available Go module proxy mirrors:")
	fmt.Println()
	fmt.Printf("  %-12s %-36s %s\n", "name", "URL", "Description")
	fmt.Printf("  %-12s %-36s %s\n", "----", "---", "-----------")
	for _, m := range Mirrors {
		fmt.Printf("  %-12s %-36s %s\n", m.Name, m.URL, m.Desc)
	}
	fmt.Println()
	fmt.Println("Use `mirrorctl goproxy set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	ensureGo()

	m := findMirror(name)
	if m == nil {
		util.Warn("unknown Go module proxy mirror: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors {
			names[i] = mir.Name
		}
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl goproxy list` for details.")
		return 1
	}

	// GOPROXY format: <url>,direct
	goproxy := m.URL + ",direct"

	cmd := exec.Command("go", "env", "-w", "GOPROXY="+goproxy)
	if out, err := cmd.CombinedOutput(); err != nil {
		util.Warn("go env -w GOPROXY failed: %v\n%s", err, string(out))
		return 1
	}

	fmt.Printf("GOPROXY set to %s\n", goproxy)
	fmt.Printf("Mirror: %s (%s)\n", m.Name, m.Desc)

	return 0
}

// --- unset ---

func cmdUnset() int {
	ensureGo()

	// Check current value first
	out, err := exec.Command("go", "env", "GOPROXY").Output()
	if err != nil {
		util.Warn("cannot read GOPROXY: %v", err)
		return 1
	}

	current := strings.TrimSpace(string(out))
	if current == "https://proxy.golang.org,direct" {
		fmt.Println("GOPROXY is already at the Go default, nothing to unset.")
		return 0
	}

	cmd := exec.Command("go", "env", "-u", "GOPROXY")
	if out, err := cmd.CombinedOutput(); err != nil {
		util.Warn("go env -u GOPROXY failed: %v\n%s", err, string(out))
		return 1
	}

	fmt.Printf("Removed GOPROXY setting (was %s)\n", current)
	fmt.Println("Go will use the default proxy: https://proxy.golang.org,direct")

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
	fmt.Println("Testing Go module proxy mirrors...")
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

	fmt.Printf("  %-12s %-44s %s\n", "name", "description", "latency")
	fmt.Printf("  %-12s %-44s %s\n", "----", "-----------", "-------")
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("  %-12s %-44s %s\n", r.name, r.desc, r.err.Error())
		} else {
			ms := float64(r.latency) / float64(time.Millisecond)
			fmt.Printf("  %-12s %-44s %.0f ms\n", r.name, r.desc, ms)
		}
	}
	fmt.Println()
	return 0
}