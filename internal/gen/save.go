package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

// extByMediaType maps an IANA media type to a file extension.
func extByMediaType(mt string) string {
	switch mt {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	default:
		return ".img"
	}
}

// now is overridable in tests for deterministic file names.
var now = time.Now

// Save writes the generated image into dir with a timestamped, collision-free
// name whose extension matches the media type. It creates dir if missing and
// returns the resolved path.
func Save(dir string, res provider.GenerateResult) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("gen: output directory is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("gen: creating output dir %s: %w", dir, err)
	}
	ext := extByMediaType(res.MediaType)
	base := now().Format("2006-01-02-150405")
	path := filepath.Join(dir, base+ext)
	// Avoid collisions when generating multiple images within one second.
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
	}
	if err := os.WriteFile(path, res.Data, 0o644); err != nil {
		return "", fmt.Errorf("gen: writing image to %s: %w", path, err)
	}
	return path, nil
}
