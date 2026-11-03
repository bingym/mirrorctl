package github

import (
	"fmt"
	"strings"
)

// Proxy describes a GitHub acceleration proxy line.
//
// Every gh-proxy.com line works the same way: the line's prefix (for example
// "https://gh-proxy.com/") is prepended to the raw GitHub URL, so adding a
// line only requires adding one entry to Proxies.
type Proxy struct {
	Name    string   // canonical name accepted by --proxy
	Aliases []string // alternative names accepted by --proxy
	Prefix  string   // full URL prefix prepended to raw GitHub URLs
	Desc    string   // human-readable description
	Note    string   // extra constraint, printed by the help text
}

// Proxies is the table of supported GitHub acceleration proxies.
//
// The gh-proxy.com service (same family as the docker module) exposes several
// lines that share the same prefix layout but differ in edge network: the
// main site (gh-proxy.com / gh-proxy.org), a China-optimised IPv4 node (v4.),
// an IPv4/IPv6 dual-stack node (v6.) and a Fastly CDN node (cdn.).
var Proxies = []Proxy{
	{
		Name:    "gh-proxy",
		Aliases: []string{"ghproxy", "main"},
		Prefix:  "https://gh-proxy.com/",
		Desc:    "GH-Proxy main site (Cloudflare v4), global distribution",
		Note:    "IPv4",
	},
	{
		Name:    "gh-proxy-org",
		Aliases: []string{"org"},
		Prefix:  "https://gh-proxy.org/",
		Desc:    "GH-Proxy main site alt domain (Cloudflare v4)",
		Note:    "IPv4",
	},
	{
		Name:    "gh-proxy-v4",
		Aliases: []string{"v4"},
		Prefix:  "https://v4.gh-proxy.org/",
		Desc:    "GH-Proxy china-optimised node (Cloudflare v4), recommended",
		Note:    "IPv4",
	},
	{
		Name:    "gh-proxy-v6",
		Aliases: []string{"v6"},
		Prefix:  "https://v6.gh-proxy.org/",
		Desc:    "GH-Proxy dual-stack node (Cloudflare v4/v6)",
		Note:    "IPv4/IPv6",
	},
	{
		Name:    "gh-proxy-fastly",
		Aliases: []string{"fastly", "cdn"},
		Prefix:  "https://cdn.gh-proxy.org/",
		Desc:    "GH-Proxy Fastly CDN node",
		Note:    "IPv4",
	},
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
	return Proxy{}, fmt.Errorf("unknown proxy %q (available: %s)", name, strings.Join(ProxyNames(), ", "))
}

// ResolveProxy looks up a proxy by name. A value that already looks like a
// URL prefix (e.g. "https://mirror.example.com/") is accepted as a custom
// proxy prefix.
func ResolveProxy(name string) (Proxy, error) {
	p, err := LookupProxy(name)
	if err == nil {
		return p, nil
	}

	prefix := strings.TrimSpace(name)
	if strings.HasPrefix(prefix, "https://") || strings.HasPrefix(prefix, "http://") {
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		return Proxy{Name: prefix, Prefix: prefix, Desc: "custom proxy prefix"}, nil
	}

	return Proxy{}, err
}

// knownPrefix returns the leading proxy prefix in raw, or "" when raw carries
// no known proxy prefix.
func knownPrefix(raw string) string {
	for _, p := range Proxies {
		if strings.HasPrefix(raw, p.Prefix) {
			return p.Prefix
		}
	}
	return ""
}
