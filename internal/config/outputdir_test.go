package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputDir(t *testing.T) {
	fakeWd := func() (string, error) { return "/work/here", nil }
	fakeHome := func() (string, error) { return "/home/u", nil }
	const def = "/home/u/Pictures/genifer"

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"empty uses default", "", def},
		{"dot uses pwd", ".", "/work/here"},
		{"pwd keyword uses pwd", "pwd", "/work/here"},
		{"whitespace trimmed to pwd", "  .  ", "/work/here"},
		{"bare tilde uses home", "~", "/home/u"},
		{"tilde path expands to home", "~/Downloads/genifer", filepath.Join("/home/u", "Downloads/genifer")},
		{"relative anchored to pwd", "out/images", filepath.Join("/work/here", "out/images")},
		{"absolute as-is", "/tmp/g", "/tmp/g"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveOutputDir(tc.raw, fakeWd, fakeHome, def)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ResolveOutputDir(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// 1.2: ValidateOutputDir accepts a creatable/writable directory and rejects one
// that cannot be created (its parent is a regular file).
func TestValidateOutputDir(t *testing.T) {
	home := func() (string, error) { return t.TempDir(), nil }
	wd := func() (string, error) { return t.TempDir(), nil }

	// Accepted: a fresh absolute path under a temp dir is created and writable.
	good := filepath.Join(t.TempDir(), "images", "out")
	resolved, err := ValidateOutputDir(good, wd, home, good)
	if err != nil {
		t.Fatalf("expected %q to validate: %v", good, err)
	}
	if resolved != good {
		t.Errorf("resolved = %q, want %q", resolved, good)
	}

	// Rejected: a path whose parent is a regular file cannot be created.
	fileParent := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(fileParent, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(fileParent, "sub")
	if _, err := ValidateOutputDir(bad, wd, home, bad); err == nil {
		t.Errorf("expected %q to be rejected", bad)
	}
}
