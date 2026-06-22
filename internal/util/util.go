// Package util provides common utilities: error reporting, path expansion,
// file I/O, and directory creation for mirrorctl.
package util

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MirrorTarget describes a mirror to test.
type MirrorTarget struct {
	Name string
	Desc string
	URL  string
}

// MirrorResult holds the latency test result for a single mirror.
type MirrorResult struct {
	Name    string
	Desc    string
	Latency time.Duration
	Err     error
}

// TestMirrors concurrently sends HEAD requests to all targets and returns
// results sorted by latency (successful first, errors last alphabetically).
func TestMirrors(targets []MirrorTarget) []MirrorResult {
	results := make([]MirrorResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target MirrorTarget) {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}
			start := time.Now()
			resp, err := client.Head(target.URL)
			elapsed := time.Since(start)
			if err != nil {
				results[idx] = MirrorResult{Name: target.Name, Desc: target.Desc, Err: err}
				return
			}
			resp.Body.Close()
			results[idx] = MirrorResult{Name: target.Name, Desc: target.Desc, Latency: elapsed}
		}(i, t)
	}
	wg.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		ei, ej := results[i].Err, results[j].Err
		if ei != nil && ej != nil {
			return results[i].Name < results[j].Name
		}
		if ei != nil {
			return false
		}
		if ej != nil {
			return true
		}
		return results[i].Latency < results[j].Latency
	})

	return results
}

// PrintTestResults prints a formatted latency test result table to stdout.
func PrintTestResults(title string, results []MirrorResult) {
	fmt.Println(title)
	fmt.Println()

	// Calculate column widths dynamically
	nameW := 8
	descW := 44
	for _, r := range results {
		if len(r.Name) > nameW {
			nameW = len(r.Name)
		}
		if len(r.Desc) > descW {
			descW = len(r.Desc)
		}
	}

	hdrFmt := fmt.Sprintf("  %%-%ds %%-%ds %%s\n", nameW, descW)
	fmt.Fprintf(os.Stdout, hdrFmt, "name", "description", "latency")
	fmt.Fprintf(os.Stdout, hdrFmt, strings.Repeat("-", nameW), strings.Repeat("-", descW), "-------")
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stdout, hdrFmt, r.Name, r.Desc, r.Err.Error())
		} else {
			ms := float64(r.Latency) / float64(time.Millisecond)
			fmt.Fprintf(os.Stdout, hdrFmt, r.Name, r.Desc, fmt.Sprintf("%.0f ms", ms))
		}
	}
	fmt.Println()
}

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
