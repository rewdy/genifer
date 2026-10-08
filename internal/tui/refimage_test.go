package tui

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makePNG returns the PNG encoding of a tiny solid image.
func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestValidateRefImage(t *testing.T) {
	validPNG := makePNG(t)

	cases := []struct {
		name    string
		data    []byte
		wantErr error
		wantMT  string
	}{
		{"valid png", validPNG, nil, "image/png"},
		{"oversized", make([]byte, maxRefImageBytes+1), errRefImageTooLarge, ""},
		{"non-image", []byte("this is not an image"), errRefImageUndecodable, ""},
		{"empty", nil, errRefImageUndecodable, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := validateRefImage(tc.data)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ref.MediaType != tc.wantMT {
				t.Errorf("MediaType = %q, want %q", ref.MediaType, tc.wantMT)
			}
			if !bytes.Equal(ref.Data, tc.data) {
				t.Error("Data should be the original bytes unchanged")
			}
		})
	}
}
