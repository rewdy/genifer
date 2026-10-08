package tui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	// Register decoders so image.DecodeConfig recognizes the formats a user is
	// likely to paste or reference. Blank imports: used for their side effects.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/rewdy/genifer/internal/provider"
)

// maxRefImageBytes caps the size of a single reference image. Pasted or
// file-referenced data larger than this is rejected before it reaches memory
// buffers or the network.
const maxRefImageBytes = 20 << 20 // 20 MiB

// errRefImageTooLarge and errRefImageUndecodable are the two validation
// failures. They are not sentinels the UI branches on — any validation failure
// becomes the same transient, non-fatal status — but distinct messages aid
// debugging and tests.
var (
	errRefImageTooLarge    = errors.New("image is too large")
	errRefImageUndecodable = errors.New("data is not a supported image")
)

// imageFormatMediaType maps image/* format names (as returned by
// image.DecodeConfig) to IANA media types.
var imageFormatMediaType = map[string]string{
	"png":  "image/png",
	"jpeg": "image/jpeg",
	"gif":  "image/gif",
}

// validateRefImage checks that data is a supported image within the size cap
// and returns it as a provider.ReferenceImage with the derived media type. It
// does not fully decode the image — only the header, enough to confirm the
// format and derive the media type.
func validateRefImage(data []byte) (provider.ReferenceImage, error) {
	if len(data) == 0 {
		return provider.ReferenceImage{}, errRefImageUndecodable
	}
	if len(data) > maxRefImageBytes {
		return provider.ReferenceImage{}, fmt.Errorf("%w (%d bytes, max %d)", errRefImageTooLarge, len(data), maxRefImageBytes)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return provider.ReferenceImage{}, errRefImageUndecodable
	}
	mt, ok := imageFormatMediaType[format]
	if !ok {
		return provider.ReferenceImage{}, fmt.Errorf("%w (%s)", errRefImageUndecodable, format)
	}
	return provider.ReferenceImage{Data: data, MediaType: mt}, nil
}

// refPathSigil marks a prompt line as a reference-image file path. A line whose
// first non-space character is this sigil is treated as a path, not prompt
// text.
const refPathSigil = '@'

// extractRefPaths scans a prompt for sigil-prefixed path lines. It returns the
// resolved file paths (with a leading ~ expanded to the home directory) in
// order, and the prompt with those lines removed. Lines that are not
// sigil-prefixed are preserved verbatim, including surrounding blank lines.
func extractRefPaths(prompt string) (paths []string, cleaned string) {
	lines := strings.Split(prompt, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] == refPathSigil {
			raw := strings.TrimSpace(trimmed[1:])
			if raw == "" {
				// A lone sigil is not a path; keep the line as prompt text.
				kept = append(kept, line)
				continue
			}
			paths = append(paths, expandHome(raw))
			continue
		}
		kept = append(kept, line)
	}
	return paths, strings.TrimSpace(strings.Join(kept, "\n"))
}

// expandHome expands a leading ~ (optionally ~/...) to the user's home
// directory. On failure it returns the path unchanged.
func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// loadRefImageFile reads and validates an image at path, returning it as a
// provider.ReferenceImage. A missing/unreadable file or unsupported/oversized
// content yields an error; the size cap is enforced before full validation.
func loadRefImageFile(path string) (provider.ReferenceImage, error) {
	info, err := os.Stat(path)
	if err != nil {
		return provider.ReferenceImage{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if info.IsDir() {
		return provider.ReferenceImage{}, fmt.Errorf("%s is a directory, not an image", path)
	}
	if info.Size() > maxRefImageBytes {
		return provider.ReferenceImage{}, fmt.Errorf("%w: %s", errRefImageTooLarge, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return provider.ReferenceImage{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	return validateRefImage(data)
}
