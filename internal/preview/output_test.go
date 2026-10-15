package preview

import (
	"bytes"
	"errors"
	"github.com/charmbracelet/x/term"
	"image"
	"image/color"
	"io"
	"os"
	"strings"
	"testing"
)

func fixture() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 30, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 30; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 8), G: uint8(y * 12), B: 120, A: 255})
		}
	}
	return img
}
func writerFixture(protocol Protocol) (*Writer, *bytes.Buffer, Placement) {
	r := new(Registry)
	id := r.Store(fixture())
	var output bytes.Buffer
	return NewWriter(&output, protocol, r), &output, Placement{id, 4, 3, 2, 4}
}
func TestNativeFrameLifecycle(t *testing.T) {
	for _, protocol := range []Protocol{Kitty, ITerm2} {
		t.Run(map[Protocol]string{Kitty: "kitty", ITerm2: "iterm2"}[protocol], func(t *testing.T) {
			w, out, p := writerFixture(protocol)
			frame := "header" + Marker(p) + "\nreserved\nresult"
			if n, err := w.Write([]byte(frame)); err != nil || n != len(frame) {
				t.Fatalf("write %d %v", n, err)
			}
			if strings.Contains(out.String(), markerPrefix) || !strings.Contains(out.String(), "header\nreserved\nresult") {
				t.Fatal("marker leaked or text changed")
			}
			if !strings.Contains(out.String(), "\x1b7\x1b[4;3H") || !strings.HasSuffix(out.String(), "\x1b8") {
				t.Fatal("cursor not preserved")
			}
			if protocol == Kitty {
				for _, want := range []string{"a=T", "q=2", "i=31871", "C=1", "f=100"} {
					if !strings.Contains(out.String(), want) {
						t.Errorf("missing %s", want)
					}
				}
			} else {
				for _, want := range []string{"inline=1", "width=30px", "height=20px"} {
					if !strings.Contains(out.String(), want) {
						t.Errorf("missing %s", want)
					}
				}
				if strings.Contains(out.String(), "preserveAspectRatio=0") {
					t.Fatal("stretched")
				}
			}
			// An unchanged marker line can be skipped by Bubble Tea; later text writes
			// must still redraw the desired image, after clearing the old one.
			out.Reset()
			w.Write([]byte("status changed"))
			if !strings.Contains(out.String(), "status changed") || w.drawn.ID != p.ID {
				t.Fatal("lost placement on partial redraw")
			}
			if protocol == Kitty {
				if !strings.Contains(out.String(), "a=d") || !strings.Contains(out.String(), "d=I") {
					t.Fatal("missing owned resource deletion")
				}
				if strings.Contains(out.String(), "d=A") {
					t.Fatal("delete-all")
				}
			} else {
				if !strings.Contains(out.String(), "\x1b[4;3H\x1b[4X") {
					t.Fatal("inline image not erased")
				}
			}
			out.Reset()
			w.Write([]byte(Marker(Placement{}) + "compose"))
			if w.drawn.ID != 0 || strings.Contains(out.String(), "a=T") || strings.Contains(out.String(), "inline=1") {
				t.Fatal("navigation retained image")
			}
			out.Reset()
			w.Close()
			first := out.String()
			w.Close()
			if out.String() != first {
				t.Fatal("close not idempotent")
			}
		})
	}
}
func TestMarkersAcrossEverySplit(t *testing.T) {
	r := new(Registry)
	id := r.Store(fixture())
	frame := "prefix" + Marker(Placement{id, 2, 2, 2, 2}) + "suffix"
	for split := 0; split <= len(frame); split++ {
		var out bytes.Buffer
		w := NewWriter(&out, None, r)
		w.Write([]byte(frame[:split]))
		w.Write([]byte(frame[split:]))
		w.Close()
		if out.String() != "prefixsuffix" {
			t.Fatalf("split %d: %q", split, out.String())
		}
	}
}
func TestOrdinaryOutputPreserved(t *testing.T) {
	const data = "\x1b[?1049hhello\n\x1b[31mworld\x1b[0m\x1b]1337;other\a\x1b"
	var out bytes.Buffer
	w := NewWriter(&out, None, new(Registry))
	for _, b := range []byte(data) {
		if _, err := w.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	if out.String() != data {
		t.Fatalf("output changed: %q", out.String())
	}
}
func TestCleanupBeforeAltScreenExit(t *testing.T) {
	w, out, p := writerFixture(Kitty)
	w.Write([]byte(Marker(p) + "frame"))
	out.Reset()
	w.Write([]byte("\x1b[?1049l"))
	s := out.String()
	if strings.Index(s, "a=d") < 0 || strings.Index(s, "a=d") > strings.Index(s, "\x1b[?1049l") || strings.Contains(s, "a=T") {
		t.Fatalf("bad teardown %q", s)
	}
	if w.registry.lookup(p.ID) != nil {
		t.Fatal("registry not released")
	}
}
func TestRegistryBound(t *testing.T) {
	r := new(Registry)
	first := r.Store(fixture())
	second := r.Store(fixture())
	third := r.Store(fixture())
	if r.lookup(first) != nil || r.lookup(second) == nil || r.lookup(third) == nil {
		t.Fatal("registry not bounded")
	}
	if r.Store(nil) != 0 {
		t.Fatal("nil image accepted")
	}
}

type shortWriter struct{ bytes.Buffer }

func (w *shortWriter) Write(p []byte) (int, error) { return w.Buffer.Write(p[:min(3, len(p))]) }

type brokenWriter struct{ err error }

func (w brokenWriter) Write([]byte) (int, error) { return 0, w.err }
func TestPartialWritesAndErrors(t *testing.T) {
	short := new(shortWriter)
	w := NewWriter(short, None, new(Registry))
	if n, err := w.Write([]byte("hello world")); err != nil || n != 11 || short.String() != "hello world" {
		t.Fatalf("%d %v %q", n, err, short.String())
	}
	want := errors.New("broken output")
	w = NewWriter(brokenWriter{want}, None, new(Registry))
	if _, err := w.Write([]byte("hello")); !errors.Is(err, want) {
		t.Fatal(err)
	}
	w = NewWriter(brokenWriter{}, None, new(Registry))
	if _, err := w.Write([]byte("hello")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	w = NewWriter(io.Discard, None, new(Registry))
	w.Close()
	if _, err := w.Write([]byte("hello")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func TestWriterPreservesTerminalFileInterface(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "terminal-output")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := NewWriter(f, None, new(Registry))
	// Bubble Tea uses this exact interface assertion before asking for sizes.
	var output io.Writer = w
	terminal, ok := output.(term.File)
	if !ok || terminal.Fd() != f.Fd() {
		t.Fatal("wrapper hides output TTY")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stat(); err != nil {
		t.Fatal("wrapper closed caller's output")
	}
	if NewWriter(io.Discard, None, new(Registry)).Fd() != ^uintptr(0) {
		t.Fatal("non-file output claims a terminal descriptor")
	}
}
