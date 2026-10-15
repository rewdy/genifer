package preview

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxFileBytes       = 64 << 20
	MaxPixels          = 32_000_000
	PreparationTimeout = 5 * time.Second
)

var ErrUnavailable = errors.New("thumbnail unavailable")

// Item describes a bounded, prepared bitmap already published to the registry.
type Item struct {
	ID            uint64
	Width, Height int
}

// Loader serializes decoding so cancellation and resize cannot accumulate large
// source images. Only a bounded thumbnail is retained after each load.
type Loader struct {
	registry *Registry
	gate     chan struct{}
}

func NewLoader(registry *Registry) *Loader { return &Loader{registry, make(chan struct{}, 1)} }
func (l *Loader) Load(ctx context.Context, path string, maxWidth, maxHeight int) (Item, error) {
	ctx, cancel := context.WithTimeout(ctx, PreparationTimeout)
	defer cancel()
	select {
	case l.gate <- struct{}{}:
		defer func() { <-l.gate }()
	case <-ctx.Done():
		return Item{}, ctx.Err()
	}
	img, err := Prepare(ctx, path, maxWidth, maxHeight)
	if err != nil {
		return Item{}, err
	}
	if err = ctx.Err(); err != nil {
		return Item{}, err
	}
	id := l.registry.Store(img)
	if id == 0 {
		return Item{}, ErrUnavailable
	}
	return Item{id, img.Bounds().Dx(), img.Bounds().Dy()}, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// Prepare reads only regular local files, checks header dimensions before full
// decoding, and fits into the pixel box. Files over 64 MiB, images over 32 MP,
// SVG, and unusable fits are omitted. It never modifies the saved output.
func Prepare(ctx context.Context, path string, maxWidth, maxHeight int) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxWidth < 1 || maxHeight < 1 || strings.EqualFold(filepath.Ext(path), ".svg") {
		return nil, ErrUnavailable
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("preview: opening image: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("preview: inspecting file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > MaxFileBytes {
		return nil, ErrUnavailable
	}
	reader := func() io.Reader { return contextReader{ctx, io.LimitReader(f, MaxFileBytes)} }
	cfg, _, err := image.DecodeConfig(reader())
	if err != nil {
		return nil, fmt.Errorf("preview: image header: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxPixels/cfg.Height {
		return nil, ErrUnavailable
	}
	width, height := FitPixels(cfg.Width, cfg.Height, maxWidth, maxHeight)
	if width < 1 || height < 1 {
		return nil, ErrUnavailable
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("preview: rewinding image: %w", err)
	}
	src, _, err := image.Decode(reader())
	if err != nil {
		return nil, fmt.Errorf("preview: decoding image: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return dst, nil
}
