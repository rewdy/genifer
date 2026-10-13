package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveOutputDir turns a resolved output_dir string into an absolute
// directory path:
//
//   - ""            -> defaultDir (e.g. ~/Pictures/genifer)
//   - "." or "pwd"  -> the current working directory
//   - "~" or "~/..." -> the leading ~ expanded to the user's home directory
//   - relative path -> resolved against the current working directory
//   - absolute path -> used as-is
//
// getwd, homeDir, and defaultDir are injected so the behavior is testable; pass
// os.Getwd, os.UserHomeDir, and the computed default in production.
func ResolveOutputDir(raw string, getwd, homeDir func() (string, error), defaultDir string) (string, error) {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "":
		return defaultDir, nil
	case ".", "pwd":
		return getwd()
	case "~":
		return homeDir()
	}
	if strings.HasPrefix(raw, "~/") {
		home, err := homeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, raw[2:]), nil
	}
	if filepath.IsAbs(raw) {
		return raw, nil
	}
	wd, err := getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, raw), nil
}

// DefaultOutputDir returns the built-in default output directory,
// ~/Pictures/genifer.
func DefaultOutputDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Pictures", "genifer"), nil
}

// ValidateOutputDir checks that raw resolves to a directory genifer can write
// generated images into, returning the resolved absolute path on success. It
// resolves raw via ResolveOutputDir, then probes that the directory is
// creatable and writable by creating it (if needed) and writing a temporary
// file. getwd/homeDir/defaultDir are injected to match ResolveOutputDir.
func ValidateOutputDir(raw string, getwd, homeDir func() (string, error), defaultDir string) (resolved string, err error) {
	resolved, err = ResolveOutputDir(raw, getwd, homeDir, defaultDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return "", fmt.Errorf("config: save location %q is not creatable: %w", resolved, err)
	}
	probe, err := os.CreateTemp(resolved, ".genifer-write-probe-*")
	if err != nil {
		return "", fmt.Errorf("config: save location %q is not writable: %w", resolved, err)
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)
	return resolved, nil
}
