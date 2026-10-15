package provider

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"testing"
)

func referencePNG(t *testing.T, w, h int) ReferenceImage {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return ReferenceImage{Data: b.Bytes(), MediaType: "image/png"}
}
func TestValidateReferences(t *testing.T) {
	min, max := 1, 2
	style := Capabilities{AcceptsReferenceImages: true, ReferenceMin: &min, ReferenceMax: &max, ReferenceMediaTypes: []string{"image/png", "image/jpeg", "image/webp"}, ReferenceMinWidth: 256, ReferenceMinHeight: 256}
	good := referencePNG(t, 256, 256)
	var b bytes.Buffer
	if err := gif.Encode(&b, image.NewPaletted(image.Rect(0, 0, 256, 256), color.Palette{color.Black, color.White}), nil); err != nil {
		t.Fatal(err)
	}
	gifRef := ReferenceImage{Data: b.Bytes(), MediaType: "image/gif"}
	webp, err := os.ReadFile("testdata/reference.webp")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		caps    Capabilities
		refs    []ReferenceImage
		minimum bool
		want    error
	}{
		{"valid boundary", style, []ReferenceImage{good}, true, nil},
		{"short width", style, []ReferenceImage{referencePNG(t, 255, 256)}, true, ErrInvalidReference},
		{"short height", style, []ReferenceImage{referencePNG(t, 256, 255)}, true, ErrInvalidReference},
		{"missing", style, nil, true, ErrInvalidReference},
		{"attachment may be below min", style, nil, false, nil},
		{"too many", style, []ReferenceImage{good, good, good}, false, ErrInvalidReference},
		{"maximum", style, []ReferenceImage{good, good}, true, nil},
		{"gif style", style, []ReferenceImage{gifRef}, true, ErrInvalidReference},
		{"gif other", Capabilities{AcceptsReferenceImages: true}, []ReferenceImage{gifRef}, true, nil},
		{"webp", style, []ReferenceImage{{Data: webp, MediaType: "image/webp"}}, true, nil},
		{"malformed", style, []ReferenceImage{{Data: []byte("bad")}}, true, ErrInvalidReference},
		{"mime mismatch", style, []ReferenceImage{{Data: good.Data, MediaType: "image/jpeg"}}, true, ErrInvalidReference},
		{"oversize", style, []ReferenceImage{{Data: make([]byte, MaxReferenceBytes+1)}}, true, ErrInvalidReference},
		{"unsupported", Capabilities{}, []ReferenceImage{good}, true, ErrReferenceImagesUnsupported},
		{"optional", Capabilities{AcceptsReferenceImages: true}, nil, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateReferences(context.Background(), tc.caps, tc.refs, tc.minimum)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ValidateReferences(ctx, style, []ReferenceImage{good}, true); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
