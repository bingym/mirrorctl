package docker

import "testing"

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
		{"https://gh-proxy.org/docker/nginx", "https://gh-proxy.org/docker/nginx"},
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
	for _, name := range []string{"convert", "pull"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}

	if _, err := cmd.PersistentFlags().GetString("proxy"); err != nil {
		t.Error("missing persistent --proxy flag")
	}
}
