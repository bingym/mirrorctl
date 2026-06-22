package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/etc/hosts", "/etc/hosts"},
		{"relative", "relative"},
	}

	for _, tt := range tests {
		got := ExpandHome(tt.input)
		if got != tt.want {
			t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileExists(t *testing.T) {
	// existing file
	tmp := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmp, []byte("hello"), 0o644)
	if !FileExists(tmp) {
		t.Errorf("FileExists(%q) = false, want true", tmp)
	}

	// non-existing file
	if FileExists(filepath.Join(t.TempDir(), "nope")) {
		t.Error("FileExists for missing file = true, want false")
	}
}

func TestReadWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "test.txt")
	MkdirAll(filepath.Dir(path))

	data := []byte("hello mirrorctl")
	WriteFileAtomic(path, data)

	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("ReadFile = %q, want %q", got, data)
	}

	// ReadFile on non-existent returns nil, nil
	got, err = ReadFile(filepath.Join(dir, "missing"))
	if got != nil || err != nil {
		t.Errorf("ReadFile missing = %v, %v; want nil, nil", got, err)
	}
}

func TestBackupFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.txt")
	os.WriteFile(path, []byte("original"), 0o644)

	bak, err := BackupFile(path)
	if err != nil {
		t.Fatalf("BackupFile: %v", err)
	}
	if bak != path+".bak" {
		t.Errorf("BackupFile path = %q, want %q", bak, path+".bak")
	}

	data, _ := os.ReadFile(bak)
	if string(data) != "original" {
		t.Errorf("backup content = %q, want %q", data, "original")
	}
}
