package provider

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"slices"
	"strings"

	_ "golang.org/x/image/webp"
)

// MaxReferenceBytes is the per-image input cap, enforced before decoding.
const MaxReferenceBytes = 20 << 20

// InspectReference validates an image header and derives its actual media type.
// It preserves the original bytes and never performs a full raster decode.
func InspectReference(data []byte) (ReferenceImage, image.Config, error) {
	if len(data) > MaxReferenceBytes {
		return ReferenceImage{}, image.Config{}, fmt.Errorf("%w: image is too large (max 20 MiB)", ErrInvalidReference)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	media := map[string]string{"png": "image/png", "jpeg": "image/jpeg", "gif": "image/gif", "webp": "image/webp"}[format]
	if err != nil || media == "" {
		return ReferenceImage{}, image.Config{}, fmt.Errorf("%w: data is not a supported image", ErrInvalidReference)
	}
	return ReferenceImage{Data: data, MediaType: media}, cfg, nil
}

// ValidateReferences checks input constraints. requireMinimum is false while
// attaching/removing images and true before submission. Unknown bounds impose
// no count restriction. Context cancellation is checked between image headers.
func ValidateReferences(ctx context.Context, caps Capabilities, refs []ReferenceImage, requireMinimum bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(refs) > 0 && !caps.AcceptsReferenceImages {
		return ErrReferenceImagesUnsupported
	}
	if requireMinimum && caps.ReferenceMin != nil && len(refs) < *caps.ReferenceMin {
		return fmt.Errorf("%w: attach at least %d reference image(s) (currently %d)", ErrInvalidReference, *caps.ReferenceMin, len(refs))
	}
	if caps.ReferenceMax != nil && len(refs) > *caps.ReferenceMax {
		return fmt.Errorf("%w: at most %d reference image(s) allowed (currently %d); remove extras", ErrInvalidReference, *caps.ReferenceMax, len(refs))
	}
	for i, ref := range refs {
		if err := ctx.Err(); err != nil {
			return err
		}
		actual, cfg, err := InspectReference(ref.Data)
		if err != nil {
			return fmt.Errorf("reference %d: %w", i+1, err)
		}
		if ref.MediaType != "" && ref.MediaType != actual.MediaType {
			return fmt.Errorf("%w: reference %d media type is %s, not %s", ErrInvalidReference, i+1, actual.MediaType, ref.MediaType)
		}
		if len(caps.ReferenceMediaTypes) > 0 && !slices.Contains(caps.ReferenceMediaTypes, actual.MediaType) {
			return fmt.Errorf("%w: reference %d must be %s", ErrInvalidReference, i+1, strings.Join(caps.ReferenceMediaTypes, ", "))
		}
		if cfg.Width < caps.ReferenceMinWidth || cfg.Height < caps.ReferenceMinHeight {
			return fmt.Errorf("%w: reference %d must be at least %d x %d pixels (got %d x %d)", ErrInvalidReference, i+1, caps.ReferenceMinWidth, caps.ReferenceMinHeight, cfg.Width, cfg.Height)
		}
	}
	return nil
}
