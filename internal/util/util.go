// Package util provides common utilities: error reporting, path expansion,
// file I/O, and directory creation for mirrorctl.
package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Die prints a fatal error to stderr and exits with code 1.
func Die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mirrorctl: "+format+"\n", args...)
	os.Exit(1)
}

// DieErr prints a fatal error with an associated error value.
func DieErr(what string, err error) {
	Die("%s: %v", what, err)
}

// Warn prints a non-fatal warning to stderr.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mirrorctl: "+format+"\n", args...)
}

// ExpandHome expands a leading ~ to the user's home directory.
// If the path does not start with ~, it is returned as-is.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home := os.Getenv("HOME")
	if home == "" {
		u, err := user.Current()
		if err != nil {
			Die("cannot determine HOME directory: %v", err)
		}
		home = u.HomeDir
	}

	rest := path[1:] // skip ~
	if strings.HasPrefix(rest, "/") {
		rest = rest[1:] // skip optional /
	}
	return filepath.Join(home, rest)
}

// FileExists reports whether a file or directory exists at path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MkdirAll creates a directory and all parents, like mkdir -p.
func MkdirAll(path string) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		DieErr("mkdir -p "+path, err)
	}
}

// ReadFile reads the entire file at path. Returns nil if the file does not exist.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// WriteFileAtomic writes data to a temporary file then renames it to path,
// ensuring an atomic replacement.
func WriteFileAtomic(path string, data []byte) {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		DieErr("write "+tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		DieErr("rename "+tmp+" -> "+path, err)
	}
}

// BackupFile reads the file at path and writes a copy to path.bak.
// Returns the backup path.
func BackupFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	bak := path + ".bak"
	if err := os.WriteFile(bak, data, 0o644); err != nil {
		return "", err
	}
	return bak, nil
}
