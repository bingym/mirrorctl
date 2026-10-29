package docker

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestConvertImageGhProxy(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"nginx:latest", "gh-proxy.org/docker/nginx:latest"},
		{"ubuntu:22.04", "gh-proxy.org/docker/ubuntu:22.04"},
		{"ghcr.io/linuxserver/webtop", "gh-proxy.org/docker/ghcr.io/linuxserver/webtop"},
		{"gcr.io/kaniko-project/executor:debug", "gh-proxy.org/docker/gcr.io/kaniko-project/executor:debug"},
		// already-converted references are passed through unchanged
		{"gh-proxy.org/docker/nginx", "gh-proxy.org/docker/nginx"},
		// a scheme or a different proxy prefix is normalised away
		{"https://gh-proxy.org/docker/nginx", "gh-proxy.org/docker/nginx"},
		{"v4.gh-proxy.org/docker/nginx", "gh-proxy.org/docker/nginx"},
	}
	for _, c := range cases {
		got, err := convertImage(c.in, "gh-proxy")
		if err != nil {
			t.Errorf("convertImage(%q, gh-proxy) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("convertImage(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestConvert1ms(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Docker Hub: bare / namespaced / explicit docker.io
		{"nginx", "docker.1ms.run/nginx"},
		{"nginx:latest", "docker.1ms.run/nginx:latest"},
		{"library/nginx:latest", "docker.1ms.run/library/nginx:latest"},
		{"docker.io/nginx", "docker.1ms.run/nginx"},
		// other registries: host-only replacement
		{"ghcr.io/linuxserver/webtop:latest", "ghcr.1ms.run/linuxserver/webtop:latest"},
		{"ghcr.io/user/repo", "ghcr.1ms.run/user/repo"},
		{"gcr.io/kaniko-project/executor:debug", "gcr.1ms.run/kaniko-project/executor:debug"},
		{"registry.k8s.io/pause", "k8s.1ms.run/pause"},
		{"quay.io/prometheus/prometheus", "quay.1ms.run/prometheus/prometheus"},
		{"mcr.microsoft.com/dotnet/runtime:8.0", "mcr.1ms.run/dotnet/runtime:8.0"},
		{"docker.elastic.co/elasticsearch/elasticsearch:8.0.0", "elastic.1ms.run/elasticsearch/elasticsearch:8.0.0"},
		// already-converted references are passed through unchanged
		{"docker.1ms.run/nginx", "docker.1ms.run/nginx"},
		{"ghcr.1ms.run/user/repo", "ghcr.1ms.run/user/repo"},
	}
	for _, c := range cases {
		got, err := convert1ms(c.in)
		if err != nil {
			t.Errorf("convert1ms(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("convert1ms(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestConvert1msErrors(t *testing.T) {
	unsupported := []string{
		"gitlab.example.com/group/proj", // unknown registry host
		"registry.example.com:5000/img", // host:port unknown
		"",
	}
	for _, in := range unsupported {
		if _, err := convert1ms(in); err == nil {
			t.Errorf("convert1ms(%q) expected error, got none", in)
		}
	}
}

func TestConvertImageProxyFlag(t *testing.T) {
	got, err := convertImage("nginx:latest", "1ms")
	if err != nil {
		t.Fatalf("convertImage(1ms) error: %v", err)
	}
	if got != "docker.1ms.run/nginx:latest" {
		t.Errorf("convertImage(1ms) = %q", got)
	}
}

func TestCommand(t *testing.T) {
	cmd := Command()

	if cmd.Use != "docker" {
		t.Errorf("Command().Use = %q, want %q", cmd.Use, "docker")
	}

	subs := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, name := range []string{"convert", "pull", "proxy"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}

	if _, err := cmd.PersistentFlags().GetString("proxy"); err != nil {
		t.Error("missing persistent --proxy flag")
	}
}

func TestProxyList(t *testing.T) {
	cmd := Command()
	var proxyCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "proxy" {
			proxyCmd = sub
		}
	}
	if proxyCmd == nil {
		t.Fatal("missing proxy subcommand")
	}

	found := false
	for _, sub := range proxyCmd.Commands() {
		if sub.Name() == "list" {
			found = true
		}
	}
	if !found {
		t.Error("missing proxy list subcommand")
	}
}

func TestLookupProxy(t *testing.T) {
	cases := map[string]string{
		"gh-proxy":        "gh-proxy",
		"main":            "gh-proxy",
		"gh-proxy-v4":     "gh-proxy-v4",
		"v4":              "gh-proxy-v4",
		"v6":              "gh-proxy-v6",
		"gh-proxy-v6":     "gh-proxy-v6",
		"fastly":          "gh-proxy-fastly",
		"CDN":             "gh-proxy-fastly",
		"gh-proxy-fastly": "gh-proxy-fastly",
		"1ms":             "1ms",
		"1ms.run":         "1ms",
	}
	for in, want := range cases {
		p, err := LookupProxy(in)
		if err != nil {
			t.Errorf("LookupProxy(%q) error: %v", in, err)
			continue
		}
		if p.Name != want {
			t.Errorf("LookupProxy(%q).Name = %q, want %q", in, p.Name, want)
		}
	}

	if _, err := LookupProxy("nope"); err == nil {
		t.Error("LookupProxy(\"nope\") expected error, got none")
	}
}

func TestProxiesTable(t *testing.T) {
	// The four gh-proxy lines published on https://gh-proxy.com/docker, plus 1ms.
	want := []struct {
		name, prefix string
	}{
		{"gh-proxy", "gh-proxy.org/docker/"},
		{"gh-proxy-v4", "v4.gh-proxy.org/docker/"},
		{"gh-proxy-v6", "v6.gh-proxy.org/docker/"},
		{"gh-proxy-fastly", "cdn.gh-proxy.org/docker/"},
		{"1ms", ""},
	}

	if len(Proxies) != len(want) {
		t.Fatalf("got %d proxies, want %d", len(Proxies), len(want))
	}
	for i, w := range want {
		if Proxies[i].Name != w.name || Proxies[i].Prefix != w.prefix {
			t.Errorf("Proxies[%d] = {%q, %q}, want {%q, %q}",
				i, Proxies[i].Name, Proxies[i].Prefix, w.name, w.prefix)
		}
	}
}

func TestConvertImageProxies(t *testing.T) {
	cases := []struct {
		proxy, in, want string
	}{
		{"gh-proxy", "nginx:latest", "gh-proxy.org/docker/nginx:latest"},
		{"gh-proxy-v4", "nginx:latest", "v4.gh-proxy.org/docker/nginx:latest"},
		{"v4", "caddy:2.11.7", "v4.gh-proxy.org/docker/caddy:2.11.7"},
		{"gh-proxy-v6", "nginx:latest", "v6.gh-proxy.org/docker/nginx:latest"},
		{"gh-proxy-fastly", "nginx:latest", "cdn.gh-proxy.org/docker/nginx:latest"},
		{"fastly", "ghcr.io/user/repo:tag", "cdn.gh-proxy.org/docker/ghcr.io/user/repo:tag"},
		// switching proxies rewrites the prefix instead of stacking it
		{"gh-proxy-fastly", "gh-proxy.org/docker/nginx", "cdn.gh-proxy.org/docker/nginx"},
		// registry rewrite
		{"1ms", "nginx:latest", "docker.1ms.run/nginx:latest"},
		// custom host
		{"docker.example.com", "nginx:latest", "docker.example.com/docker/nginx:latest"},
	}
	for _, c := range cases {
		got, err := convertImage(c.in, c.proxy)
		if err != nil {
			t.Errorf("convertImage(%q, %q) error: %v", c.in, c.proxy, err)
			continue
		}
		if got != c.want {
			t.Errorf("convertImage(%q, %q) = %q, want %q", c.in, c.proxy, got, c.want)
		}
	}

	if _, err := convertImage("nginx", "nope"); err == nil {
		t.Error("convertImage with unknown proxy expected error, got none")
	}
}
