// Package github implements the "github" subcommand for mirrorctl,
// providing GitHub URL conversion, download, and clone acceleration via gh-proxy.com.
package github

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

const maxRetries = 3

// defaultProxy is the base URL for the GitHub acceleration proxy.
const defaultProxy = "https://gh-proxy.com/"

// proxyURL prepends the default proxy prefix to a raw GitHub URL.
func proxyURL(raw string) string {
	return defaultProxy + raw
}

// isGitHubURL reports whether raw looks like a GitHub-hosted URL.
func isGitHubURL(raw string) bool {
	return strings.Contains(raw, "github.com") ||
		strings.Contains(raw, "raw.githubusercontent.com") ||
		strings.Contains(raw, "gist.githubusercontent.com") ||
		strings.Contains(raw, "api.github.com")
}

// convertURL transforms a GitHub URL (or shorthand) into a proxy-accelerated URL.
// It accepts HTTPS URLs, git@ SSH URLs, and user/repo shorthand.
func convertURL(raw string) (string, error) {
	if strings.HasPrefix(raw, defaultProxy) {
		return raw, nil
	}

	raw = strings.TrimSpace(raw)

	// git@github.com:user/repo.git → https://github.com/user/repo.git
	if strings.HasPrefix(raw, "git@github.com:") {
		path := strings.TrimPrefix(raw, "git@github.com:")
		return proxyURL("https://github.com/" + path), nil
	}

	// user/repo shorthand (no protocol, single slash)
	if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "git@") {
		parts := strings.SplitN(raw, "/", 3)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			suffix := ""
			if !strings.HasSuffix(raw, ".git") {
				suffix = ".git"
			}
			return proxyURL("https://github.com/" + raw + suffix), nil
		}
	}

	if !isGitHubURL(raw) {
		return "", fmt.Errorf("not a supported GitHub URL: %s", raw)
	}

	return proxyURL(raw), nil
}

// Command returns the cobra command for the "github" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Accelerate GitHub access via proxy (gh-proxy.com)",
		Long: `Convert, download, and clone GitHub repositories through an acceleration proxy.

The proxy prefix (https://gh-proxy.com/) is prepended to GitHub URLs to route
traffic through CDN-accelerated nodes, improving access speed in regions where
GitHub is slow or unreliable.`,
		Example: `  mirrorctl github convert https://github.com/user/repo
  mirrorctl github download https://github.com/user/repo/releases/download/v1.0/file.tar.gz
  mirrorctl github clone https://github.com/user/repo.git
  mirrorctl github clone user/repo`,
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
			Use:   "convert <github-url>",
			Short: "Convert a GitHub URL to a proxy accelerator URL",
			Long: `Convert a GitHub URL to its proxy-accelerated equivalent.

Supports:
  - HTTPS URLs (github.com, raw.githubusercontent.com, gist.githubusercontent.com, api.github.com)
  - SSH git@ URLs (git@github.com:user/repo.git)
  - Shorthand (user/repo)

The converted URL is printed to stdout for use with any tool (wget, curl, git, etc.).`,
			Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdConvert(args[0]))
			},
		},
		&cobra.Command{
			Use:   "download <github-url>",
			Short: "Download a file from GitHub via proxy accelerator",
			Long: `Download a GitHub resource (release asset, raw file, archive, tarball, etc.)
through the proxy accelerator using curl.`,
			Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdDownload(args[0]))
			},
		},
		&cobra.Command{
			Use:   "clone <github-url>",
			Short: "Clone a GitHub repository via proxy accelerator",
			Long: `Clone a GitHub repository through the proxy accelerator. Supports:
  - Full HTTPS URLs: https://github.com/user/repo.git
  - SSH git@ URLs:   git@github.com:user/repo.git
  - Shorthand:       user/repo`,
			Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdClone(args[0]))
			},
		},
	)

	return cmd
}

// --- convert ---

func cmdConvert(rawURL string) int {
	proxy, err := convertURL(rawURL)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}
	fmt.Println(proxy)
	return 0
}

// --- download ---

func cmdDownload(rawURL string) int {
	proxy, err := convertURL(rawURL)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}

	if _, err := exec.LookPath("curl"); err != nil {
		util.Warn("curl not found in PATH; install curl or use `mirrorctl github convert %s` and download manually", rawURL)
		return 1
	}

	fmt.Printf("Downloading via proxy: %s\n", proxy)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cmd := exec.Command("curl", "-L", "-O", proxy)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return 0
		} else {
			lastErr = err
		}

		if attempt < maxRetries {
			fmt.Fprintf(os.Stderr, "retrying (%d/%d) after 2s...\n", attempt, maxRetries-1)
			time.Sleep(2 * time.Second)
		}
	}

	util.Warn("download failed after %d attempts: %v", maxRetries, lastErr)
	return 1
}

// --- clone ---

func cmdClone(rawURL string) int {
	proxy, err := convertURL(rawURL)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}

	if _, err := exec.LookPath("git"); err != nil {
		util.Warn("git not found in PATH; install git or use `mirrorctl github convert %s` and clone manually", rawURL)
		return 1
	}

	fmt.Printf("Cloning via proxy: %s\n", proxy)
	cmd := exec.Command("git", "clone", proxy)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		util.Warn("clone failed: %v", err)
		return 1
	}
	return 0
}
