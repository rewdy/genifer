package gen

import (
	"context"
	"fmt"
	"os/exec"
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
