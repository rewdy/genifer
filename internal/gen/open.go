package gen

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// defaultOpenCommand returns the per-OS command used to open a file in the
// system viewer.
func defaultOpenCommand(goos string) string {
	switch goos {
	case "darwin":
		return "open"
	case "windows":
		return "start"
	default:
		return "xdg-open"
	}
}

// OpenCommand returns the command used to open images: override when non-empty,
// otherwise the default for the current OS.
func OpenCommand(override string) string {
	if override != "" {
		return override
	}
	return defaultOpenCommand(runtime.GOOS)
}

// Open launches the system viewer for path using the resolved open command.
func Open(ctx context.Context, override, path string) error {
	cmd := OpenCommand(override)
	c := exec.CommandContext(ctx, cmd, path)
	if err := c.Start(); err != nil {
		return fmt.Errorf("gen: opening %s with %q: %w", path, cmd, err)
	}
	// Don't wait: the viewer is a separate, long-lived process.
	go func() { _ = c.Wait() }()
	return nil
}

// OpenFolder opens the saved image's containing directory in the system file
// manager. The image-viewer override does not apply to directories.
func OpenFolder(ctx context.Context, imagePath string) error {
	path, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("gen: resolving image directory: %w", err)
	}
	override := ""
	if runtime.GOOS == "windows" {
		override = "explorer"
	}
	return Open(ctx, override, filepath.Dir(path))
}
