package pypi

import "testing"

func TestExtractIndexURL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "standard config",
			content: `[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple/

[install]
trusted-host = pypi.tuna.tsinghua.edu.cn
`,
			want: "https://pypi.tuna.tsinghua.edu.cn/simple/",
		},
		{
			name: "no global section",
			content: `[install]
trusted-host = example.com
`,
			want: "",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name: "index-url without spaces",
			content: `[global]
index-url=https://mirrors.aliyun.com/pypi/simple/
`,
			want: "https://mirrors.aliyun.com/pypi/simple/",
		},
		{
			name: "other section after global",
			content: `[global]
index-url = https://example.com/simple/
timeout = 60

[install]
trusted-host = example.com
`,
			want: "https://example.com/simple/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIndexURL(tt.content)
			if got != tt.want {
				t.Errorf("extractIndexURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindMirror(t *testing.T) {
	m := findMirror("tuna")
	if m == nil {
		t.Fatal("findMirror(tuna) = nil")
	}
	if m.Desc != "Tsinghua University TUNA" {
		t.Errorf("tuna desc = %q", m.Desc)
	}

	if findMirror("nonexistent") != nil {
		t.Error("findMirror(nonexistent) should be nil")
	}
}

func TestCommand(t *testing.T) {
	cmd := Command()

	if cmd.Use != "pypi" {
		t.Errorf("Command().Use = %q, want %q", cmd.Use, "pypi")
	}

	// Verify all subcommands are registered
	subs := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}

	expected := []string{"config", "list", "set", "unset"}
	for _, name := range expected {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}
