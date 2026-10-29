// Package docker implements the "docker" subcommand for mirrorctl,
// providing Docker image reference acceleration through the proxies listed in
// "docker proxy list" (gh-proxy.org main / v4 / v6 / Fastly nodes and 1ms.run).
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/bingym/mirrorctl/internal/util"
	"github.com/spf13/cobra"
)

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

// convertImage rewrites a Docker image reference using the named proxy,
// as listed by "docker proxy list".
func convertImage(ref, proxy string) (string, error) {
	p, err := ResolveProxy(proxy)
	if err != nil {
		return "", err
	}
	return p.Rewrite(ref)
}

// Command returns the cobra command for the "docker" subcommand tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Accelerate Docker image pulls via proxy (gh-proxy.org lines, 1ms.run, ...)",
		Long: `Convert and pull Docker images through an acceleration proxy.

Available proxies (select with --proxy, see "docker proxy list"):
  - gh-proxy (default): prefix "gh-proxy.org/docker/"    (main site, Cloudflare v4)
  - gh-proxy-v4:        prefix "v4.gh-proxy.org/docker/" (china-optimised, recommended)
  - gh-proxy-v6:        prefix "v6.gh-proxy.org/docker/" (IPv4/IPv6 dual stack)
  - gh-proxy-fastly:    prefix "cdn.gh-proxy.org/docker/" (Fastly CDN)
  - 1ms:                replace the registry host with a 1ms.run mirror
                        (docker.io -> docker.1ms.run, ghcr.io -> ghcr.1ms.run, ...)

A custom proxy host may also be passed, e.g. --proxy docker.example.com,
which is used as "<host>/docker/<image>".

Supports all public registries: Docker Hub, Google GCR, GitHub GHCR,
Red Hat Quay, Microsoft MCR, Elastic, NVIDIA NGC, AWS ECR, GitLab,
Kubernetes (registry.k8s.io, k8s.gcr.io), and more.`,
		Example: `  mirrorctl docker proxy list
  mirrorctl docker convert nginx:latest
  mirrorctl docker convert --proxy v4 nginx:latest
  mirrorctl docker convert --proxy 1ms nginx:latest
  mirrorctl docker pull nginx:latest
  mirrorctl docker pull gcr.io/kaniko-project/executor:debug --proxy gh-proxy-v6`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "unknown command %q for %q\n\n", args[0], cmd.CommandPath())
				cmd.Help()
				os.Exit(1)
			}
			cmd.Help()
		},
	}

	cmd.PersistentFlags().String("proxy", DefaultProxyName, "Proxy to use (see `mirrorctl docker proxy list`)")

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
		Long: `Pull a Docker image through the proxy accelerator selected by --proxy.

Runs: docker pull <proxy-prefixed-image>, then retags the result to the
original image reference and removes the proxy-prefixed tag, so the local
image always carries its original name.

Any public image reference is supported:
  - Official images:  nginx, ubuntu:22.04
  - Other registries: gcr.io/..., ghcr.io/..., quay.io/..., etc.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			proxy, _ := cmd.Flags().GetString("proxy")
			os.Exit(cmdPull(args[0], proxy))
		},
	}
	cmd.AddCommand(pullCmd)

	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Inspect available Docker acceleration proxies",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "unknown command %q for %q\n\n", args[0], cmd.CommandPath())
				cmd.Help()
				os.Exit(1)
			}
			cmd.Help()
		},
	}
	proxyCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all supported proxies",
		Long: `List every supported acceleration proxy with its name, conversion mode,
description and any network constraint.

Names from the NAME column are valid values for --proxy.`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(cmdProxyList())
		},
	})
	cmd.AddCommand(proxyCmd)

	return cmd
}

// --- proxy list ---

func cmdProxyList() int {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMODE\tPREFIX / REWRITE\tDESCRIPTION\tNET")
	for _, p := range Proxies {
		target := p.Prefix
		if target == "" {
			target = "registry host -> mirror host"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.Name, p.Mode(), target, p.Desc, p.Note)
	}
	if err := w.Flush(); err != nil {
		util.Warn("write output: %v", err)
		return 1
	}

	fmt.Printf("\nDefault: %s\n", DefaultProxyName)
	fmt.Printf("Usage: docker pull <image> --proxy <name>, e.g. docker pull nginx:latest --proxy 1ms\n")
	return 0
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

func cmdPull(ref, proxy string) int {
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

	if proxyRef == ref {
		// Nothing to strip: the reference was already proxy-prefixed. Skip
		// retag/remove, which would otherwise delete the just-pulled image.
		fmt.Println("Image reference is already proxy-prefixed; nothing to strip.")
		return 0
	}

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

	return 0
}
