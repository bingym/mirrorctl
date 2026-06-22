// Package docker implements the "docker" subcommand for mirrorctl,
// providing Docker image reference acceleration via gh-proxy.org/docker.
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

// proxyPrefix is the proxy registry prefix for Docker image acceleration.
// Used as a Docker image reference (host/path), not a URL.
const proxyPrefix = "gh-proxy.org/docker/"

// convertImage prepends the proxy prefix to a Docker image reference.
func convertImage(ref string) string {
	ref = strings.TrimSpace(ref)
	// Accept both "gh-proxy.org/docker/..." and "https://gh-proxy.org/docker/..."
	if strings.HasPrefix(ref, proxyPrefix) || strings.HasPrefix(ref, "https://"+proxyPrefix) {
		return ref
	}
	return proxyPrefix + ref
}

// Command returns the cobra command for the "docker" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Accelerate Docker image pulls via proxy (gh-proxy.org/docker)",
		Long: `Convert and pull Docker images through an acceleration proxy.

The proxy prefix "gh-proxy.org/docker/" is prepended to image references
to route pulls through CDN-accelerated nodes, improving download speed in regions
where container registries are slow or unreliable.

Supports all public registries: Docker Hub, Google GCR, GitHub GHCR,
Red Hat Quay, Microsoft MCR, Elastic, NVIDIA NGC, AWS ECR, GitLab,
Kubernetes (registry.k8s.io, k8s.gcr.io), and more.`,
		Example: `  mirrorctl docker convert nginx:latest
  mirrorctl docker convert ghcr.io/user/repo:tag
  mirrorctl docker pull nginx:latest
  mirrorctl docker pull --strip nginx:latest
  mirrorctl docker pull gcr.io/kaniko-project/executor:debug`,
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
			Use:   "convert <image>",
			Short: "Convert a Docker image reference to a proxy URL",
			Long: `Convert a Docker image reference to its proxy-accelerated equivalent.

Supports any public image reference:
  - Official images:  nginx, ubuntu:22.04
  - Namespaced:       library/nginx, prom/prometheus
  - With registry:    docker.io/nginx
  - Other registries: gcr.io/..., ghcr.io/..., quay.io/..., etc.

The converted reference is printed to stdout.`,
			Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(cmdConvert(args[0]))
			},
		},
	)

	pullCmd := &cobra.Command{
		Use:   "pull <image>",
		Short: "Pull a Docker image via proxy accelerator",
		Long: `Pull a Docker image through the proxy accelerator.

Runs: docker pull <proxy-prefix><image>

Any public image reference is supported:
  - Official images:  nginx, ubuntu:22.04
  - Other registries: gcr.io/..., ghcr.io/..., quay.io/..., etc.`,
		Args: cobra.ExactArgs(1),
	}
	strip := pullCmd.Flags().Bool("strip", false, "Strip proxy prefix after pull: tag to original name and remove proxy-tagged image")
	pullCmd.Run = func(cmd *cobra.Command, args []string) {
		os.Exit(cmdPull(args[0], *strip))
	}
	cmd.AddCommand(pullCmd)

	return cmd
}

// --- convert ---

func cmdConvert(ref string) int {
	result := convertImage(ref)
	fmt.Println(result)
	return 0
}

// --- pull ---

func cmdPull(ref string, strip bool) int {
	proxyRef := convertImage(ref)

	if _, err := exec.LookPath("docker"); err != nil {
		util.Warn("docker not found in PATH; install Docker or use `mirrorctl docker convert %s` and pull manually", ref)
		return 1
	}

	fmt.Printf("Pulling via proxy: %s\n", proxyRef)
	pull := exec.Command("docker", "pull", proxyRef)
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		util.Warn("docker pull failed: %v", err)
		return 1
	}

	if strip {
		fmt.Printf("Tagging image: %s -> %s\n", proxyRef, ref)
		tag := exec.Command("docker", "tag", proxyRef, ref)
		tag.Stdout = os.Stdout
		tag.Stderr = os.Stderr
		if err := tag.Run(); err != nil {
			util.Warn("docker tag failed: %v", err)
			return 1
		}

		fmt.Printf("Removing proxy-prefixed image: %s\n", proxyRef)
		rmi := exec.Command("docker", "rmi", proxyRef)
		rmi.Stdout = os.Stdout
		rmi.Stderr = os.Stderr
		if err := rmi.Run(); err != nil {
			util.Warn("docker rmi failed (image may still exist as dangling): %v", err)
			return 1
		}

		fmt.Println("Done. Image renamed to original reference.")
	}

	return 0
}
