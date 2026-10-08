package tui

import (
	"context"
	"errors"
	"testing"
)

// fakePaster is a test double for imagePaster.
type fakePaster struct {
	data []byte
	err  error
}

func (f fakePaster) ReadImage(context.Context) ([]byte, error) { return f.data, f.err }

func TestFakePasterPaths(t *testing.T) {
	cases := []struct {
		name    string
		paster  imagePaster
		wantLen int
		wantErr error
	}{
		{"success", fakePaster{data: []byte("png")}, 3, nil},
		{"empty / no image", fakePaster{err: ErrNoClipboardImage}, 0, ErrNoClipboardImage},
		{"unavailable", fakePaster{err: ErrNoClipboardImage}, 0, ErrNoClipboardImage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.paster.ReadImage(context.Background())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if len(data) != tc.wantLen {
				t.Errorf("len(data) = %d, want %d", len(data), tc.wantLen)
			}
		})
	}
}
