package gen

import (
	"context"
	"strings"
	"unicode"

	"github.com/rewdy/genifer/internal/provider"
)

// NameOptions controls whether a descriptive file name is derived from a text
// model for each generated image, and which model is used.
type NameOptions struct {
	// Enabled requests a descriptive name via the text model. When false, or
	// when Model is empty, a timestamped name is used.
	Enabled bool
	// Model is the text model id used to generate the descriptive name. Empty
	// means naming is not possible and a timestamp is used.
	Model string
}

// maxBaseNameLen bounds the sanitized base name so a pathological model reply
// cannot produce an unusably long file name.
const maxBaseNameLen = 64

// sanitizeBaseName reduces arbitrary text to a filesystem-safe base name: only
// letters, digits, spaces, and hyphens survive; the result is trimmed, any
// path separators or file extensions are removed, whitespace runs collapse,
// and it is bounded in length. It returns "" for junk input so callers fall
// back to a timestamp.
func sanitizeBaseName(s string) string {
	// Drop a trailing file extension (e.g. ".png") before filtering.
	if i := strings.LastIndex(s, "."); i > 0 {
		s = s[:i]
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune(r)
		}
		if b.Len() >= maxBaseNameLen {
			break
		}
	}
	// Collapse whitespace runs and strip leading/trailing separators.
	return strings.Join(strings.Fields(strings.Trim(b.String(), " -_")), " ")
}

// nameFor attempts to derive a descriptive, sanitized base name for the draft
// via the provider's text endpoint. It returns "" when naming is disabled, no
// model is configured, the request fails or is cancelled, or the reply is
// empty/junk. Naming failures are deliberately silent: a fallback to the
// timestamped name must never block the image itself.
func nameFor(ctx context.Context, p provider.Provider, d Draft, opts NameOptions) string {
	if !opts.Enabled || opts.Model == "" {
		return ""
	}
	if ctx.Err() != nil {
		return ""
	}
	name, err := p.GenerateText(ctx, provider.TextRequest{
		Model:  opts.Model,
		Prompt: "Describe this image in a few words: " + d.Review(),
	})
	if err != nil || ctx.Err() != nil {
		return ""
	}
	return sanitizeBaseName(name)
}