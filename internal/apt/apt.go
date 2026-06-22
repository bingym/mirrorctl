// Package apt provides shared utilities for Debian/Ubuntu apt source management.
package apt

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bingym/mirrorctl/internal/util"
)

// Distro identifies a Debian-family distribution.
type Distro int

const (
	Ubuntu Distro = iota
	Debian
)

func (d Distro) String() string {
	switch d {
	case Ubuntu:
		return "ubuntu"
	case Debian:
		return "debian"
	}
	return ""
}

// Mirror represents an APT software mirror.
type Mirror struct {
	Name    string
	BaseURL string // e.g. "https://mirrors.ustc.edu.cn"
	Desc    string
}

// MirrorHost extracts the hostname from a mirror BaseURL.
func MirrorHost(m Mirror) string {
	raw := m.BaseURL
	if strings.HasPrefix(raw, "https://") {
		raw = strings.TrimPrefix(raw, "https://")
	} else if strings.HasPrefix(raw, "http://") {
		raw = strings.TrimPrefix(raw, "http://")
	}
	// Remove trailing path if any
	if idx := strings.Index(raw, "/"); idx >= 0 {
		raw = raw[:idx]
	}
	return raw
}

// DetectDistro reads /etc/os-release to detect the distribution and codename.
// Returns only Ubuntu or Debian; other distros return an error.
func DetectDistro() (Distro, string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return 0, "", fmt.Errorf("cannot read /etc/os-release: %w", err)
	}
	content := string(data)

	var id, codename string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			codename = strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), `"`)
		}
	}

	switch id {
	case "ubuntu":
		return Ubuntu, codename, nil
	case "debian":
		return Debian, codename, nil
	default:
		return 0, "", fmt.Errorf("unsupported distribution: %s (expected ubuntu or debian)", id)
	}
}

// SourcePaths returns the traditional and DEB822 config paths for the distro.
func SourcePaths(d Distro) (traditional, deb822 string) {
	switch d {
	case Ubuntu:
		return "/etc/apt/sources.list", "/etc/apt/sources.list.d/ubuntu.sources"
	case Debian:
		return "/etc/apt/sources.list", "/etc/apt/sources.list.d/debian.sources"
	}
	return "", ""
}

// DetectActiveSource detects which apt source config file is in use.
// Prefers DEB822 format if it exists, falls back to traditional sources.list.
func DetectActiveSource(d Distro) (path string, isDEB822 bool, err error) {
	trad, deb822 := SourcePaths(d)
	if util.FileExists(deb822) {
		return deb822, true, nil
	}
	if util.FileExists(trad) {
		return trad, false, nil
	}
	return "", false, fmt.Errorf("no apt source config found (looked for %s and %s)", trad, deb822)
}

// OldDomains returns the list of old domain patterns to replace for the distro.
func OldDomains(d Distro) []string {
	switch d {
	case Ubuntu:
		return []string{
			"archive.ubuntu.com",
			"security.ubuntu.com",
		}
	case Debian:
		return []string{
			"deb.debian.org",
		}
	}
	return nil
}

// ReplaceMirror reads an apt sources file, replaces old domains with the mirror
// hostname, and writes the result back atomically.
func ReplaceMirror(path string, d Distro, m Mirror) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	content := string(data)

	host := MirrorHost(m)
	for _, old := range OldDomains(d) {
		content = strings.ReplaceAll(content, old, host)
	}

	util.WriteFileAtomic(path, []byte(content))
	return nil
}

// BackupSources creates a .bak copy of the given file.
// Returns the backup path.
func BackupSources(path string) (string, error) {
	bak := path + ".bak"
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s for backup: %w", path, err)
	}
	if err := os.WriteFile(bak, data, 0o644); err != nil {
		return "", fmt.Errorf("cannot write backup %s: %w", bak, err)
	}
	return bak, nil
}

// RestoreSources restores the .bak backup of the given file path.
func RestoreSources(path string) error {
	bak := path + ".bak"
	if _, err := os.Stat(bak); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", bak)
	}
	data, err := os.ReadFile(bak)
	if err != nil {
		return fmt.Errorf("cannot read backup %s: %w", bak, err)
	}
	util.WriteFileAtomic(path, data)
	return nil
}

// RunUpdate runs apt-get update.
func RunUpdate() error {
	cmd := exec.Command("apt-get", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckRoot returns true if running as root (EUUID == 0).
func CheckRoot() bool {
	return os.Geteuid() == 0
}

// ShowConfig prints the current apt source config and detected mirror.
func ShowConfig(path string, _ Distro, mirrors []Mirror) {
	fmt.Printf("Config file: %s\n\n", path)

	data, err := util.ReadFile(path)
	if err != nil {
		util.Warn("cannot read %s: %v", path, err)
		return
	}
	if data == nil {
		fmt.Println("(empty)")
		return
	}

	content := string(data)
	fmt.Print(content)
	fmt.Println()

	// Reverse-lookup which mirror is in use
	for _, m := range mirrors {
		host := MirrorHost(m)
		if strings.Contains(content, host) {
			fmt.Printf("Mirror: %s (%s)\n", m.Name, m.Desc)
			return
		}
	}
	fmt.Println("(unknown or no mirror)")
}
