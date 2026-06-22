// Package brew implements the "brew" subcommand for mirrorctl,
// managing Homebrew mirror configuration via git remote URLs and
// environment variables.
package brew

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

type Mirror struct {
	Name       string
	BrewRemote string // HOMEBREW_BREW_GIT_REMOTE
	CoreRemote string // HOMEBREW_CORE_GIT_REMOTE
	BottleURL  string // HOMEBREW_BOTTLE_DOMAIN
	Desc       string
}

var Mirrors = []Mirror{
	{
		Name:       "tuna",
		BrewRemote: "https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/brew.git",
		CoreRemote: "https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/homebrew-core.git",
		BottleURL:  "https://mirrors.tuna.tsinghua.edu.cn/homebrew-bottles",
		Desc:       "Tsinghua University TUNA",
	},
	{
		Name:       "ustc",
		BrewRemote: "https://mirrors.ustc.edu.cn/brew.git",
		CoreRemote: "https://mirrors.ustc.edu.cn/homebrew-core.git",
		BottleURL:  "https://mirrors.ustc.edu.cn/homebrew-bottles",
		Desc:       "University of Science and Technology of China",
	},
	{
		Name:       "huawei",
		BrewRemote: "https://repo.huaweicloud.com/homebrew/brew.git",
		CoreRemote: "https://repo.huaweicloud.com/homebrew/homebrew-core.git",
		BottleURL:  "https://repo.huaweicloud.com/homebrew-bottles",
		Desc:       "Huawei Cloud",
	},
}

var defaultBrewRemote = "https://github.com/Homebrew/brew.git"
var defaultCoreRemote = "https://github.com/Homebrew/homebrew-core.git"

func findMirror(name string) *Mirror {
	for i := range Mirrors {
		if Mirrors[i].Name == name {
			return &Mirrors[i]
		}
	}
	return nil
}

func brewEnvDir() string  { return util.ExpandHome("~/.config/mirrorctl") }
func brewEnvPath() string { return brewEnvDir() + "/brew.env" }

func brewEnvContent(m Mirror) string {
	var buf strings.Builder
	fmt.Fprintln(&buf, "# mirrorctl: Homebrew mirror configuration")
	fmt.Fprintf(&buf, "export HOMEBREW_BREW_GIT_REMOTE=\"%s\"\n", m.BrewRemote)
	fmt.Fprintf(&buf, "export HOMEBREW_CORE_GIT_REMOTE=\"%s\"\n", m.CoreRemote)
	fmt.Fprintf(&buf, "export HOMEBREW_BOTTLE_DOMAIN=\"%s\"\n", m.BottleURL)
	return buf.String()
}

func brewPath() string {
	p, err := exec.LookPath("brew")
	if err != nil {
		return ""
	}
	return p
}

func brewRepoPath() string {
	out, err := exec.Command("brew", "--repo").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func brewCorePath() string {
	out, err := exec.Command("brew", "--repo", "homebrew/core").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitGetRemote(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitSetRemote(repoPath, url string) error {
	cmd := exec.Command("git", "-C", repoPath, "remote", "set-url", "origin", url)
	return cmd.Run()
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brew",
		Short: "Switch Homebrew mirror (brew + core git remote, bottle domain)",
		Long:  "Manage Homebrew mirror configuration via git remote URLs and environment variables.\n\nSets Homebrew/brew and Homebrew/homebrew-core git remotes, writes\nHOMEBREW_BOTTLE_DOMAIN to ~/.config/mirrorctl/brew.env, and prints\ninstructions for sourcing the env file in your shell.",
		Example: `  mirrorctl brew config          Show current Homebrew mirror configuration
  mirrorctl brew list            List all available brew mirrors
  mirrorctl brew set tuna        Switch to TUNA mirror
  mirrorctl brew unset           Restore to default (GitHub upstream)`,
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
			Use: "config", Short: "Show current Homebrew mirror configuration",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdConfig()) },
		},
		&cobra.Command{
			Use: "list", Short: "List all supported brew mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdList()) },
		},
		&cobra.Command{
			Use: "set <name>", Short: "Set Homebrew mirror (e.g. tuna / ustc / huawei)",
			Args: cobra.ExactArgs(1), Run: func(_ *cobra.Command, args []string) { os.Exit(cmdSet(args[0])) },
		},
		&cobra.Command{
			Use: "unset", Short: "Restore Homebrew to default (GitHub upstream)",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdUnset()) },
		},
		&cobra.Command{
			Use: "test", Short: "Test connectivity and latency of all brew mirrors",
			Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { os.Exit(cmdTest()) },
		},
	)
	return cmd
}

// --- config ---

func cmdConfig() int {
	if brewPath() == "" {
		util.Warn("brew is not installed or not in PATH")
		return 1
	}
	fmt.Printf("brew: %s\n", brewPath())

	// Check repo paths
	if rp := brewRepoPath(); rp != "" {
		u := gitGetRemote(rp)
		fmt.Printf("brew repo origin: %s\n", u)
	} else {
		fmt.Println("brew repo: not found")
	}

	if cp := brewCorePath(); cp != "" {
		u := gitGetRemote(cp)
		fmt.Printf("core repo origin: %s\n", u)
	} else {
		fmt.Println("core repo: not found")
	}

	// Check env file
	envPath := brewEnvPath()
	if util.FileExists(envPath) {
		data, _ := util.ReadFile(envPath)
		fmt.Printf("\nEnv file: %s\n", envPath)
		fmt.Print(string(data))
	} else {
		fmt.Println("\nNo brew.env file found.")
		fmt.Println("Set HOMEBREW_BOTTLE_DOMAIN manually or run `mirrorctl brew set <name>`.")
	}

	return 0
}

// --- list ---

func cmdList() int {
	fmt.Println("Available Homebrew mirrors:")
	fmt.Println()
	fmt.Printf("  %-8s %-58s %s\n", "name", "git remote URL (brew / core)", "Description")
	fmt.Printf("  %-8s %-58s %s\n", "----", "---------------------------", "-----------")
	for _, m := range Mirrors {
		remoteLine := fmt.Sprintf("%s / %s", m.BrewRemote, m.CoreRemote)
		fmt.Printf("  %-8s %-58s %s\n", m.Name, remoteLine, m.Desc)
	}
	fmt.Println()
	fmt.Println("Also configures HOMEBREW_BOTTLE_DOMAIN to the mirror's bottle URL.")
	fmt.Println("Use `mirrorctl brew set <name>` to switch mirrors.")
	return 0
}

// --- set ---

func cmdSet(name string) int {
	m := findMirror(name)
	if m == nil {
		util.Warn("unknown brew mirror: %s", name)
		names := make([]string, len(Mirrors))
		for i, mir := range Mirrors { names[i] = mir.Name }
		fmt.Fprintf(os.Stderr, "Available mirrors: %s\n", strings.Join(names, " "))
		fmt.Fprintln(os.Stderr, "Run `mirrorctl brew list` for details.")
		return 1
	}

	if brewPath() == "" {
		util.Warn("brew is not installed or not in PATH")
		return 1
	}

	// Set brew repo remote
	rp := brewRepoPath()
	if rp == "" {
		util.Warn("cannot determine brew repo path")
		return 1
	}
	if err := gitSetRemote(rp, m.BrewRemote); err != nil {
		util.Warn("failed to set brew origin: %v", err)
		return 1
	}
	fmt.Printf("brew repo origin -> %s\n", m.BrewRemote)

	// Set core tap remote
	cp := brewCorePath()
	if cp == "" {
		util.Warn("cannot determine core tap path")
		return 1
	}
	if err := gitSetRemote(cp, m.CoreRemote); err != nil {
		util.Warn("failed to set core origin: %v", err)
		return 1
	}
	fmt.Printf("core repo origin -> %s\n", m.CoreRemote)

	// Write env file
	util.MkdirAll(brewEnvDir())
	util.WriteFileAtomic(brewEnvPath(), []byte(brewEnvContent(*m)))

	fmt.Printf("\nHomebrew mirror set to %s (%s)\n", m.Name, m.Desc)
	fmt.Printf("Bottle domain: %s\n", m.BottleURL)
	fmt.Printf("Env file: %s\n\n", brewEnvPath())
	fmt.Println("To apply the environment variables, add this to your ~/.zshrc or ~/.bashrc:")
	fmt.Printf("  source %s\n", brewEnvPath())

	return 0
}

// --- unset ---

func cmdUnset() int {
	if brewPath() == "" {
		util.Warn("brew is not installed or not in PATH")
		return 1
	}

	// Restore brew repo remote
	rp := brewRepoPath()
	if rp == "" {
		util.Warn("cannot determine brew repo path")
		return 1
	}
	if err := gitSetRemote(rp, defaultBrewRemote); err != nil {
		util.Warn("failed to restore brew origin: %v", err)
		return 1
	}
	fmt.Printf("brew repo origin restored -> %s\n", defaultBrewRemote)

	// Restore core tap remote
	cp := brewCorePath()
	if cp == "" {
		util.Warn("cannot determine core tap path")
		return 1
	}
	if err := gitSetRemote(cp, defaultCoreRemote); err != nil {
		util.Warn("failed to restore core origin: %v", err)
		return 1
	}
	fmt.Printf("core repo origin restored -> %s\n", defaultCoreRemote)

	// Remove env file
	envPath := brewEnvPath()
	if util.FileExists(envPath) {
		if err := os.Remove(envPath); err != nil {
			util.Warn("cannot remove %s: %v", envPath, err)
			return 1
		}
		fmt.Printf("Removed: %s\n", envPath)
	}

	fmt.Println("\nHomebrew restored to default (GitHub upstream).")
	fmt.Println("Don't forget to remove the `source` line from your shell config if you added it.")

	return 0
}

// --- test ---

func cmdTest() int {
	targets := make([]util.MirrorTarget, 0, len(Mirrors)*3)
	for _, m := range Mirrors {
		targets = append(targets, util.MirrorTarget{
			Name: m.Name, Desc: m.Desc + " (brew.git)", URL: strings.TrimSuffix(m.BrewRemote, ".git"),
		})
		targets = append(targets, util.MirrorTarget{
			Name: m.Name, Desc: m.Desc + " (core.git)", URL: strings.TrimSuffix(m.CoreRemote, ".git"),
		})
		targets = append(targets, util.MirrorTarget{
			Name: m.Name, Desc: m.Desc + " (bottles)", URL: m.BottleURL,
		})
	}
	results := util.TestMirrors(targets)
	util.PrintTestResults("Testing Homebrew mirrors (brew.git + core.git + bottles)...", results)
	return 0
}


