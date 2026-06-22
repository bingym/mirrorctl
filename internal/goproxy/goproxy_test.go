package goproxy

import "testing"

func TestFindMirror(t *testing.T) {
	m := findMirror("goproxy.cn")
	if m == nil {
		t.Fatal("findMirror(goproxy.cn) = nil")
	}
	if m.Desc != "goproxy.cn - China Go module proxy (by Qiniu)" {
		t.Errorf("goproxy.cn desc = %q", m.Desc)
	}

	if findMirror("nonexistent") != nil {
		t.Error("findMirror(nonexistent) should be nil")
	}
}

func TestMirrorsTable(t *testing.T) {
	if len(Mirrors) < 2 {
		t.Error("Mirrors table should have at least 2 entries")
	}

	seen := make(map[string]bool)
	for _, m := range Mirrors {
		if m.Name == "" {
			t.Error("mirror name should not be empty")
		}
		if m.URL == "" {
			t.Errorf("mirror %q URL should not be empty", m.Name)
		}
		if m.Desc == "" {
			t.Errorf("mirror %q desc should not be empty", m.Name)
		}
		if seen[m.Name] {
			t.Errorf("duplicate mirror name: %s", m.Name)
		}
		seen[m.Name] = true
	}
}

func TestCommand(t *testing.T) {
	cmd := Command()

	if cmd.Use != "goproxy" {
		t.Errorf("Command().Use = %q, want %q", cmd.Use, "goproxy")
	}

	// Verify all subcommands are registered
	subs := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}

	expected := []string{"config", "list", "set", "unset", "test"}
	for _, name := range expected {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}