package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// EditableSettings are the config.yaml values the in-tool config view can edit.
// Each is a top-level scalar key; the writer updates exactly these and leaves
// everything else (comments, the api_key directive, provider, etc.) untouched.
type EditableSettings struct {
	OutputDir string // literal output_dir value (not a directive)
	AutoOpen  bool
	History   bool
    SmartFilenames bool // smart_filenames directive
    FilenameModel  string // filename_model directive
}

// WriteSettings updates output_dir, auto_open, and history in the config.yaml
// at path, preserving the rest of the file (comments and all other settings),
// and writes atomically (temp file + rename). It performs a targeted,
// line-level edit of just those three top-level scalar keys: an existing key
// (even one commented out) is replaced in place; a missing key is appended.
//
// The caller is responsible for validating values (e.g. the save location)
// before calling. WriteSettings assumes the file already exists and parses.
func WriteSettings(path string, s EditableSettings) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: reading %s: %w", path, err)
	}

	updated := string(data)
	updated = setScalarKey(updated, "output_dir", quoteYAML(s.OutputDir))
	updated = setScalarKey(updated, "auto_open", strconv.FormatBool(s.AutoOpen))
	updated = setScalarKey(updated, "history", strconv.FormatBool(s.History))
	updated = setScalarKey(updated, "smart_filenames", strconv.FormatBool(s.SmartFilenames))
	updated = setScalarKey(updated, "filename_model", fmt.Sprintf("%s", s.FilenameModel))

	if err := writeFileAtomic(path, []byte(updated)); err != nil {
		return err
	}
	return nil
}

// topLevelKeyPattern matches a top-level scalar key line (optionally commented
// out), capturing nothing — it only tests whether a line sets `key`. It allows
// a leading "# " on an otherwise-unindented key so a commented-out default is
// treated as the slot to replace.
func topLevelKeyPattern(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(#\s*)?` + regexp.QuoteMeta(key) + `\s*:`)
}

// setScalarKey replaces the first top-level line that sets key (commented or
// not) with "key: value", or appends "key: value" when no such line exists.
// Indented occurrences (nested under another mapping) are not matched, so a
// nested key of the same name is never clobbered.
func setScalarKey(content, key, value string) string {
	re := topLevelKeyPattern(key)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Only consider unindented lines: a top-level key has no leading space
		// (an optional comment marker is handled by the pattern).
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if re.MatchString(line) {
			lines[i] = key + ": " + value
			return strings.Join(lines, "\n")
		}
	}
	return appendKey(content, key+": "+value)
}

// appendKey adds a "key: value" line to content, ensuring exactly one trailing
// newline before it and after it.
func appendKey(content, line string) string {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return line + "\n"
	}
	return trimmed + "\n" + line + "\n"
}

// quoteYAML renders a string as a double-quoted YAML scalar, escaping backslash
// and double-quote so an arbitrary path round-trips safely.
func quoteYAML(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by a rename, so an interrupted write cannot leave a corrupt file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: creating config dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("config: creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: writing temp file: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("config: setting temp file mode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: replacing %s: %w", path, err)
	}
	return nil
}
