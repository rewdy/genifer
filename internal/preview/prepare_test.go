package preview

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func imageFile(t *testing.T, format string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "image."+format)
	var b bytes.Buffer
	switch format {
	case "png":
		png.Encode(&b, fixture())
	case "jpg":
		jpeg.Encode(&b, fixture(), nil)
	case "webp":
		data, err := os.ReadFile("../provider/testdata/reference.webp")
		if err != nil {
			t.Fatal(err)
		}
		b.Write(data)
	}
	if err := os.WriteFile(path, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestPrepareFormatsAndPreservesFile(t *testing.T) {
	for _, format := range []string{"png", "jpg", "webp"} {
		t.Run(format, func(t *testing.T) {
			path := imageFile(t, format)
			before, _ := os.ReadFile(path)
			got, err := Prepare(context.Background(), path, 300, 150)
			if err != nil {
				t.Fatal(err)
			}
			if got.Bounds().Dx() > 300 || got.Bounds().Dy() > 150 || got.Bounds().Empty() {
				t.Fatal(got.Bounds())
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("saved file changed")
			}
		})
	}
}
func TestPrepareAlphaAndFit(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 40))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 80})
	path := filepath.Join(t.TempDir(), "alpha.png")
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
	got, err := Prepare(context.Background(), path, 200, 150)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 75 || got.Bounds().Dy() != 150 {
		t.Fatal(got.Bounds())
	}
	_, _, _, a := got.At(74, 149).RGBA()
	if a != 0 {
		t.Fatal("alpha lost")
	}
}
func TestPrepareFailures(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.png")
	os.WriteFile(bad, []byte("bad"), 0600)
	svg := filepath.Join(dir, "vector.svg")
	original := []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)
	os.WriteFile(svg, original, 0600)
	large := filepath.Join(dir, "large.png")
	f, _ := os.Create(large)
	f.Truncate(MaxFileBytes + 1)
	f.Close()
	for _, path := range []string{filepath.Join(dir, "missing"), bad, svg, large, dir} {
		if _, err := Prepare(context.Background(), path, 100, 100); err == nil {
			t.Fatal(path)
		}
	}
	after, _ := os.ReadFile(svg)
	if !bytes.Equal(after, original) {
		t.Fatal("SVG changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Prepare(ctx, imageFile(t, "png"), 100, 100); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := NewLoader(new(Registry)).Load(ctx, bad, 100, 100); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if _, err := Prepare(context.Background(), imageFile(t, "png"), 0, 100); err == nil {
		t.Fatal("zero width")
	}
}
func TestLoaderBoundsAndCancelsWaitingWork(t *testing.T) {
	r := new(Registry)
	l := NewLoader(r)
	path := imageFile(t, "png")
	item, err := l.Load(context.Background(), path, 100, 100)
	if err != nil || item.ID == 0 || r.lookup(item.ID) == nil {
		t.Fatalf("%+v %v", item, err)
	}
	l.gate <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Load(ctx, path, 100, 100); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	<-l.gate
}

func TestPrepareRejectsOversizedHeader(t *testing.T) {
	path := imageFile(t, "png")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// PNG IHDR dimensions can be checked without allocating the source bitmap.
	binary.BigEndian.PutUint32(data[16:20], 40000)
	binary.BigEndian.PutUint32(data[20:24], 40000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	os.WriteFile(path, data, 0600)
	if _, err := Prepare(context.Background(), path, 150, 150); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
