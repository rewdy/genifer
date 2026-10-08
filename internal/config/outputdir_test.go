package config

import (
	"path/filepath"
	"testing"
)

func TestResolveOutputDir(t *testing.T) {
	fakeWd := func() (string, error) { return "/work/here", nil }
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
		{"relative anchored to pwd", "out/images", filepath.Join("/work/here", "out/images")},
		{"absolute as-is", "/tmp/g", "/tmp/g"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveOutputDir(tc.raw, fakeWd, def)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ResolveOutputDir(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
