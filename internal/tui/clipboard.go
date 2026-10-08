package tui

import (
	"context"
	"errors"
	"sync"

	"golang.design/x/clipboard"
)

// ErrNoClipboardImage is the single sentinel the TUI reacts to when a paste
// yields nothing usable: an empty clipboard for images, a clipboard that holds
// no image, or no reachable clipboard at all (no display, remote session).
// Paste is best-effort, so every such condition collapses to one non-fatal
// outcome rather than distinct errors the UI would have to tell apart.
var ErrNoClipboardImage = errors.New("no image in clipboard")

// imagePaster reads a single image off the operating-system clipboard. It is
// the narrow seam the TUI depends on so clipboard access stays mockable and
// out of Update/View. Implementations return raw image bytes, or
// ErrNoClipboardImage when nothing usable is available.
type imagePaster interface {
	ReadImage(ctx context.Context) ([]byte, error)
}

// clipboardPaster is the production imagePaster backed by
// golang.design/x/clipboard. It is cgo-free on desktop platforms.
type clipboardPaster struct {
	once    sync.Once
	initErr error
}

// newClipboardPaster returns a paster that initializes the clipboard lazily on
// first ReadImage. It must not touch the clipboard at construction time, so
// startup never depends on a reachable clipboard.
func newClipboardPaster() *clipboardPaster { return &clipboardPaster{} }

// NewClipboardPaster returns the production clipboard-backed paster for wiring
// in main. It is exported so the entry point can inject it into Deps; the
// imagePaster seam itself stays internal to the package.
func NewClipboardPaster() imagePaster { return newClipboardPaster() }

// ReadImage returns the PNG bytes on the clipboard, or ErrNoClipboardImage when
// the clipboard is unreachable or holds no image. clipboard.Init is run once,
// lazily; its failure (e.g. no display) maps to the same non-fatal sentinel.
func (c *clipboardPaster) ReadImage(ctx context.Context) ([]byte, error) {
	c.once.Do(func() { c.initErr = clipboard.Init() })
	if c.initErr != nil {
		return nil, ErrNoClipboardImage
	}
	data, err := clipboard.Read(ctx, clipboard.FmtImage)
	if err != nil || len(data) == 0 {
		return nil, ErrNoClipboardImage
	}
	return data, nil
}
