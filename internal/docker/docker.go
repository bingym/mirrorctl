// Package docker implements the "docker" subcommand for mirrorctl,
// providing Docker image reference acceleration via gh-proxy.org/docker or 1ms.run.
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

// proxyPrefix is the proxy registry prefix for the gh-proxy accelerator.
// Used as a Docker image reference (host/path), not a URL.
const proxyPrefix = "gh-proxy.org/docker/"

// oneMSDomains maps source registries to their 1ms.run accelerator domains.
var oneMSDomains = map[string]string{
	"docker.io":         "docker.1ms.run",
	"ghcr.io":           "ghcr.1ms.run",
	"gcr.io":            "gcr.1ms.run",
	"registry.k8s.io":   "k8s.1ms.run",
	"nvcr.io":           "nvcr.1ms.run",
	"quay.io":           "quay.1ms.run",
	"mcr.microsoft.com": "mcr.1ms.run",
	"docker.elastic.co": "elastic.1ms.run",
	"docker.1ms.run":    "docker.1ms.run",
	"ghcr.1ms.run":      "ghcr.1ms.run",
	"gcr.1ms.run":       "gcr.1ms.run",
	"k8s.1ms.run":       "k8s.1ms.run",
	"nvcr.1ms.run":      "nvcr.1ms.run",
	"quay.1ms.run":      "quay.1ms.run",
	"mcr.1ms.run":       "mcr.1ms.run",
	"elastic.1ms.run":   "elastic.1ms.run",
}

// splitRegistry splits a Docker image reference into its registry (as a
// solitary host, which may be empty) and the path. Docker references follow
// the grammar [registry/]path[:tag][@digest].
func splitRegistry(ref string) (registry, path string) {
	if i := strings.Index(ref, "/"); i >= 0 {
		return ref[:i], ref[i+1:]
	}
	return "", ref
}

// isRegistry reports whether a path segment is a registry host rather than a
// namespace/path component. Registry hosts contain a dot or colon, or are the
// special name "localhost".
func isRegistry(s string) bool {
	return s == "localhost" || strings.Contains(s, ".") || strings.Contains(s, ":")
}

// convert1ms rewrites a Docker image reference for the 1ms.run accelerator,
// replacing the registry host per oneMSDomains.
func convert1ms(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty image reference")
	}

	registry, path := splitRegistry(ref)
	if registry == "" {
		return "docker.1ms.run/" + path, nil
	}

	if !isRegistry(registry) {
		// A namespace/component path (e.g. library/nginx) belongs to Docker Hub.
		return "docker.1ms.run/" + registry + "/" + path, nil
	}

	if target, ok := oneMSDomains[registry]; ok {
		return target + "/" + path, nil
	}
	return "", fmt.Errorf("registry %q is not supported by the 1ms accelerator", registry)
}

// convertImage rewrites a Docker image reference using the given accelerator.
func convertImage(ref, proxy string) (string, error) {
	if proxy == "1ms" {
		return convert1ms(ref)
	}

	// Default: gh-proxy.org. Accept already-converted references.
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, proxyPrefix) || strings.HasPrefix(ref, "https://"+proxyPrefix) {
		return ref, nil
	}
	return proxyPrefix + ref, nil
}

// Command returns the cobra command for the "docker" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Accelerate Docker image pulls via proxy (gh-proxy.org/docker or 1ms.run)",
		Long: `Convert and pull Docker images through an acceleration proxy.

Two accelerators are available (select with --proxy):
  - gh-proxy (default): the prefix "gh-proxy.org/docker/" is prepended to image references.
  - 1ms: the registry host is replaced with a 1ms.run mirror
    (docker.io -> docker.1ms.run, ghcr.io -> ghcr.1ms.run, registry.k8s.io -> k8s.1ms.run, ...).

Supports all public registries: Docker Hub, Google GCR, GitHub GHCR,
Red Hat Quay, Microsoft MCR, Elastic, NVIDIA NGC, AWS ECR, GitLab,
Kubernetes (registry.k8s.io, k8s.gcr.io), and more.`,
		Example: `  mirrorctl docker convert nginx:latest
  mirrorctl docker convert --proxy 1ms nginx:latest
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

	cmd.PersistentFlags().String("proxy", "gh-proxy", "Accelerator to use: gh-proxy or 1ms")

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
				proxy, _ := cmd.Flags().GetString("proxy")
				os.Exit(cmdConvert(args[0], proxy))
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
		proxy, _ := cmd.Flags().GetString("proxy")
		os.Exit(cmdPull(args[0], *strip, proxy))
	}
	cmd.AddCommand(pullCmd)

	return cmd
}

// --- convert ---

func cmdConvert(ref, proxy string) int {
	result, err := convertImage(ref, proxy)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}
	fmt.Println(result)
	return 0
}

// --- pull ---

func cmdPull(ref string, strip bool, proxy string) int {
	proxyRef, err := convertImage(ref, proxy)
	if err != nil {
		util.Warn("%v", err)
		return 1
	}

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
