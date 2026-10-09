package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupConfigFresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backup != path+".bak" {
		t.Errorf("backup = %q, want %q", backup, path+".bak")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("original should have been renamed away")
	}
	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "legacy" {
		t.Errorf("backup content = %q, want legacy", got)
	}
}

func TestBackupConfigExistingBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("new-legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A prior backup already exists and must not be overwritten.
	if err := os.WriteFile(path+".bak", []byte("old-backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backup != path+".bak.1" {
		t.Errorf("backup = %q, want %q", backup, path+".bak.1")
	}
	// Existing backup preserved byte-for-byte.
	old, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old-backup" {
		t.Errorf("existing backup was clobbered: %q", old)
	}
	fresh, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(fresh) != "new-legacy" {
		t.Errorf("new backup content = %q, want new-legacy", fresh)
	}
}
