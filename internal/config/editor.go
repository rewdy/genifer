package config

import "strings"

// ErrNoEditor signals that no editor could be resolved: neither $VISUAL nor
// $EDITOR is set and the current OS has no known default. Callers should print
// the config path and an actionable message instead of launching an editor.
type noEditorError struct{}

func (noEditorError) Error() string {
	return "config: no editor found — set $VISUAL or $EDITOR"
}

// ErrNoEditor is the sentinel returned by ResolveEditor when no editor is
// available. Compare with errors.Is.
var ErrNoEditor error = noEditorError{}

// ResolveEditor determines the command (and any leading arguments) used to open
// the config file in a text editor, in order: $VISUAL, then $EDITOR, then an
// OS-appropriate default. getenv and goos are injected for testability; pass
// os.Getenv and runtime.GOOS in production.
//
// This is intentionally distinct from the image open_command: editing config is
// a text-editing task, so it honors the standard $VISUAL/$EDITOR convention
// rather than the image-viewer command.
//
// The returned argv is the editor command split on spaces (so `code -w` works),
// to which the caller appends the file path. When no editor resolves it returns
// ErrNoEditor.
func ResolveEditor(getenv func(string) string, goos string) ([]string, error) {
	for _, key := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return strings.Fields(v), nil
		}
	}
	if def := defaultEditor(goos); def != nil {
		return def, nil
	}
	return nil, ErrNoEditor
}

// defaultEditor returns the OS-appropriate default editor argv, or nil when the
// OS has no known default.
func defaultEditor(goos string) []string {
	switch goos {
	case "darwin":
		// -t opens in the default *text* editor rather than guessing by type.
		return []string{"open", "-t"}
	case "windows":
		return []string{"notepad"}
	case "linux":
		return []string{"xdg-open"}
	default:
		return nil
	}
}
