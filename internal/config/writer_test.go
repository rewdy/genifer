package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 1.1: writing settings updates the three keys while preserving comments and
// the api_key directive, and the result still Loads.
func TestWriteSettingsPreservesCommentsAndOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	// Start from the commented starter template (output_dir commented out).
	if _, err := WriteStarter(path, DefaultAPIKey); err != nil {
		t.Fatal(err)
	}

	err := WriteSettings(path, EditableSettings{
		OutputDir: "/tmp/genifer-out",
		AutoOpen:  true,
		History:   false,
	})
	if err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)

	// Comments survive.
	if !strings.Contains(text, "# genifer configuration.") {
		t.Error("header comment did not survive the write")
	}
	// The api_key directive survives verbatim.
	if !strings.Contains(text, string(DefaultAPIKey)) {
		t.Errorf("api_key directive lost; file:\n%s", text)
	}

	// The file still parses, with the edited values in effect.
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load after write: %v", err)
	}
	if cfg.OutputDir != "/tmp/genifer-out" {
		t.Errorf("OutputDir = %q, want /tmp/genifer-out", cfg.OutputDir)
	}
	if !cfg.AutoOpen {
		t.Error("AutoOpen = false, want true")
	}
	if cfg.HistoryEnabled() {
		t.Error("HistoryEnabled() = true, want false")
	}
	if cfg.OpenRouter.APIKey != DefaultAPIKey {
		t.Errorf("APIKey = %q, want %q", cfg.OpenRouter.APIKey, DefaultAPIKey)
	}
}

// 1.1: an existing (uncommented) key is replaced in place, not duplicated.
// 1.1: an existing (uncommented) key is replaced in place, not duplicated.
func TestWriteSettingsReplacesExistingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `provider: openrouter
auto_open: false
output_dir: "/old"
history: true
smart_filenames: true
filename_model: "old-model"
openrouter:
  api_key: "{env:OPENROUTER_API_KEY}"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteSettings(path, EditableSettings{
		OutputDir:      "/new",
		AutoOpen:       true,
		History:        true,
		SmartFilenames: false,
		FilenameModel:  "new-model",
	}); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if strings.Count(text, "output_dir:") != 1 {
		t.Errorf("output_dir should appear once, got:\n%s", text)
	}
	if strings.Count(text, "auto_open:") != 1 {
		t.Errorf("auto_open should appear once, got:\n%s", text)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OutputDir != "/new" {
		t.Errorf("OutputDir = %q, want /new", cfg.OutputDir)
	}
	// The nested api_key under openrouter must be untouched.
	if cfg.OpenRouter.APIKey != "{env:OPENROUTER_API_KEY}" {
		t.Errorf("nested api_key clobbered: %q", cfg.OpenRouter.APIKey)
	}
	// The new keys are replaced in place, not duplicated.
	if strings.Count(text, "smart_filenames:") != 1 {
		t.Errorf("smart_filenames should appear once, got:\n%s", text)
	}
	if strings.Count(text, "filename_model:") != 1 {
		t.Errorf("filename_model should appear once, got:\n%s", text)
	}
	if cfg.SmartFilenames {
		t.Error("SmartFilenames = true, want false")
	}
	if cfg.FilenameModel != "new-model" {
		t.Errorf("FilenameModel = %q, want new-model", cfg.FilenameModel)
	}
}

// 1.1: a path containing characters needing escaping round-trips.
func TestWriteSettingsQuotesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if _, err := WriteStarter(path, DefaultAPIKey); err != nil {
		t.Fatal(err)
	}
	weird := `/tmp/with "quotes" and \backslash`
	if err := WriteSettings(path, EditableSettings{OutputDir: weird, AutoOpen: false, History: true}); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OutputDir != Value(weird) {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, weird)
	}
}

// 1.3: smart_filenames and filename_model are written and can be replaced.
func TestWriteSettingsAddsSmartFilenamesAndFilenameModel(t *testing.T) {
    var err error
    path := filepath.Join(t.TempDir(), "config.yaml")
    // Start from a starter config containing a header comment and default keys.
    if _, err := WriteStarter(path, DefaultAPIKey); err != nil {
        t.Fatal(err)
    }
    // Set both new fields.
    err = WriteSettings(path, EditableSettings{
        OutputDir:    "/tmp/test",
        AutoOpen:     true,
        History:      false,
        SmartFilenames: true,
        FilenameModel: "test-model",
    })
    if err != nil {
        t.Fatalf("WriteSettings failed: %v", err)
    }
    raw, readErr := os.ReadFile(path)
    if readErr != nil {
        t.Fatalf("Read file failed: %v", readErr)
    }
    text := string(raw)
    // Verify smart_filenames key is present with expected value.
    if !strings.Contains(text, "smart_filenames: true") {
        t.Errorf("smart_filenames key not found with value true in %s", text)
    }
    // Verify filename_model key is present with expected value.
    if !strings.Contains(text, "filename_model: test-model") {
        t.Errorf("filename_model key not found or incorrect value in %s", text)
    }
    // Ensure the original three keys are still updated correctly.
    if !strings.Contains(text, "output_dir: \"/tmp/test\"") {
        t.Error("output_dir not correctly updated")
    }
    if !strings.Contains(text, "auto_open: true") {
        t.Error("auto_open not correctly updated")
    }
    if !strings.Contains(text, "history: false") {
        t.Error("history not correctly updated")
    }
}
