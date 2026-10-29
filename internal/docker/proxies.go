package docker

import (
	"fmt"
	"strings"
)

// Proxy describes a Docker image acceleration proxy.
//
// Two conversion strategies exist:
//   - prefix: the proxy host and path (e.g. "gh-proxy.org/docker/") are
//     prepended to the image reference;
//   - registry rewrite: the registry host inside the reference is replaced
//     with a mirror host (used by 1ms.run).
//
// Each table entry carries its own rewrite function so adding a proxy only
// requires adding one line to Proxies.
type Proxy struct {
	Name    string   // canonical name accepted by --proxy
	Aliases []string // alternative names accepted by --proxy
	Prefix  string   // image reference prefix; empty for registry-rewrite proxies
	Desc    string   // human-readable description, printed by "proxy list"
	Note    string   // extra constraint, printed by "proxy list"

	rewrite func(ref string) (string, error)
}

// Rewrite converts an image reference for this proxy.
func (p Proxy) Rewrite(ref string) (string, error) {
	if p.rewrite == nil {
		return "", fmt.Errorf("proxy %q has no conversion rule", p.Name)
	}
	return p.rewrite(ref)
}

// Mode returns a short label describing how the proxy rewrites references.
func (p Proxy) Mode() string {
	if p.Prefix == "" {
		return "registry"
	}
	return "prefix"
}

// Proxies is the table of supported Docker acceleration proxies.
//
// The gh-proxy family (https://gh-proxy.com/docker) exposes four lines that
// share the same "/docker/<image>" path layout but differ in edge network:
// the main site (gh-proxy.org), a China-optimised IPv4 node (v4.), an
// IPv4/IPv6 dual-stack node (v6.) and a Fastly CDN node (cdn.).
// 1ms.run instead replaces the registry host.
//
// The table is filled in init because the rewrite functions consult Proxies to
// normalise already-proxied references (a package-level literal would form an
// initialisation cycle).
var Proxies []Proxy

func init() {
	Proxies = []Proxy{
		{
			Name:    "gh-proxy",
			Aliases: []string{"ghproxy", "main"},
			Prefix:  "gh-proxy.org/docker/",
			Desc:    "GH-Proxy main site (Cloudflare v4), global distribution",
			Note:    "IPv4",
			rewrite: prefixRewriter("gh-proxy.org/docker/"),
		},
		{
			Name:    "gh-proxy-v4",
			Aliases: []string{"v4"},
			Prefix:  "v4.gh-proxy.org/docker/",
			Desc:    "GH-Proxy china-optimised node (Cloudflare v4), recommended",
			Note:    "IPv4",
			rewrite: prefixRewriter("v4.gh-proxy.org/docker/"),
		},
		{
			Name:    "gh-proxy-v6",
			Aliases: []string{"v6"},
			Prefix:  "v6.gh-proxy.org/docker/",
			Desc:    "GH-Proxy dual-stack node (Cloudflare v4/v6)",
			Note:    "IPv4/IPv6",
			rewrite: prefixRewriter("v6.gh-proxy.org/docker/"),
		},
		{
			Name:    "gh-proxy-fastly",
			Aliases: []string{"fastly", "cdn"},
			Prefix:  "cdn.gh-proxy.org/docker/",
			Desc:    "GH-Proxy Fastly CDN node",
			Note:    "IPv4",
			rewrite: prefixRewriter("cdn.gh-proxy.org/docker/"),
		},
		{
			Name:    "1ms",
			Aliases: []string{"1ms.run"},
			Desc:    "1ms.run: replace the registry host (docker.io -> docker.1ms.run, ghcr.io -> ghcr.1ms.run, ...)",
			Note:    "per-registry",
			rewrite: convert1ms,
		},
	}
}

// DefaultProxyName is the proxy used when --proxy is not given.
const DefaultProxyName = "gh-proxy"

// ProxyNames returns the canonical names of all supported proxies.
func ProxyNames() []string {
	names := make([]string, len(Proxies))
	for i, p := range Proxies {
		names[i] = p.Name
	}
	return names
}

// LookupProxy finds a proxy by canonical name or alias, case-insensitively.
func LookupProxy(name string) (Proxy, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, p := range Proxies {
		if p.Name == n {
			return p, nil
		}
		for _, a := range p.Aliases {
			if strings.ToLower(a) == n {
				return p, nil
			}
		}
	}
	return Proxy{}, fmt.Errorf("unknown proxy %q (available: %s); run `mirrorctl docker proxy list` for details",
		name, strings.Join(ProxyNames(), ", "))
}

// ResolveProxy looks up a proxy by name, falling back to treating a value that
// looks like a hostname (e.g. "docker.example.com") as a custom prefix proxy
// with the same "/docker/" path layout.
func ResolveProxy(name string) (Proxy, error) {
	p, err := LookupProxy(name)
	if err == nil {
		return p, nil
	}

	host := strings.TrimSuffix(strings.TrimSpace(name), "/")
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if host != "" && !strings.ContainsAny(host, " /") && strings.Contains(host, ".") {
		prefix := host + "/docker/"
		return Proxy{
			Name:    host,
			Prefix:  prefix,
			Desc:    "custom proxy host",
			rewrite: prefixRewriter(prefix),
		}, nil
	}

	return Proxy{}, err
}

// prefixRewriter builds a rewrite function that prepends prefix to the image
// reference. An already proxied reference is normalised first, so switching
// between prefix proxies (e.g. re-hosting a gh-proxy reference on Fastly)
// keeps a single, clean prefix.
func prefixRewriter(prefix string) func(string) (string, error) {
	return func(ref string) (string, error) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return "", fmt.Errorf("empty image reference")
		}

		ref = strings.TrimPrefix(ref, "https://")
		ref = strings.TrimPrefix(ref, "http://")
		if n := knownPrefixLen(ref); n > 0 {
			ref = ref[n:]
		}
		return prefix + ref, nil
	}
}

// knownPrefixLen returns the length of the leading proxy prefix in ref, or 0
// when ref carries no known proxy prefix.
func knownPrefixLen(ref string) int {
	for _, p := range Proxies {
		if p.Prefix != "" && strings.HasPrefix(ref, p.Prefix) {
			return len(p.Prefix)
		}
	}
	return 0
}
