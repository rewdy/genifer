package preview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/iterm2"
	"github.com/charmbracelet/x/ansi/kitty"
)

// Protocol is an explicitly selected native graphics transport.
type Protocol uint8

const (
	None Protocol = iota
	Kitty
	ITerm2
)

// Registry holds at most two prepared thumbnails (active and pending). Images
// passed to Store are immutable after publication; View never mutates this store.
type Registry struct {
	mu     sync.Mutex
	next   uint64
	images map[uint64]image.Image
}

func (r *Registry) Store(img image.Image) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if img == nil || img.Bounds().Empty() || img.Bounds().Dx() > 2048 || img.Bounds().Dy() > 2048 {
		return 0
	}
	if r.images == nil {
		r.images = make(map[uint64]image.Image)
	}
	r.next++
	if len(r.images) >= 2 {
		old := r.next
		for id := range r.images {
			if id < old {
				old = id
			}
		}
		delete(r.images, old)
	}
	r.images[r.next] = img
	return r.next
}
func (r *Registry) lookup(id uint64) image.Image {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.images[id]
}
func (r *Registry) Clear() { r.mu.Lock(); defer r.mu.Unlock(); clear(r.images) }

// Placement locates a thumbnail in terminal cells, using one-based coordinates.
// The bitmap's pixels determine its display size; Rows/Columns reserve its area.
type Placement struct {
	ID                         uint64
	Row, Column, Rows, Columns int
}

const markerPrefix = "\x1b]777;genifer-preview;"
const ownedImageID = 31871

// Marker is an internal zero-width control consumed by Writer. The zero value
// clears the active preview. Insert it after all text layout/styling, on row one.
func Marker(p Placement) string {
	return fmt.Sprintf("%s%d,%d,%d,%d,%d\a", markerPrefix, p.ID, p.Row, p.Column, p.Rows, p.Columns)
}
func parsePlacement(s string) Placement {
	fields := strings.Split(s, ",")
	if len(fields) != 5 {
		return Placement{}
	}
	id, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || id == 0 {
		return Placement{}
	}
	values := [4]int{}
	for i := range values {
		n, err := strconv.Atoi(fields[i+1])
		if err != nil || n < 1 || n > 65535 {
			return Placement{}
		}
		values[i] = n
	}
	return Placement{id, values[0], values[1], values[2], values[3]}
}

// Writer wraps Bubble Tea's sole output writer. It strips private markers,
// clears the old image before text repaint, then restores native graphics after
// text output. It never closes the underlying writer. Calls are serialized.
type Writer struct {
	mu             sync.Mutex
	output         io.Writer
	protocol       Protocol
	registry       *Registry
	pending        []byte
	desired, drawn Placement
	closed         bool
}

func NewWriter(output io.Writer, protocol Protocol, registry *Registry) *Writer {
	return &Writer{output: output, protocol: protocol, registry: registry}
}

// Fd preserves terminal identity for Bubble Tea's TTY detection and resize
// handling. Non-file outputs deliberately report an invalid descriptor.
func (w *Writer) Fd() uintptr {
	if f, ok := w.output.(interface{ Fd() uintptr }); ok {
		return f.Fd()
	}
	return ^uintptr(0)
}

// Read completes the terminal-file interface required by Bubble Tea. Rendering
// still uses Write exclusively; input remains owned by Bubble Tea's input TTY.
func (w *Writer) Read(p []byte) (int, error) {
	if r, ok := w.output.(io.Reader); ok {
		return r.Read(p)
	}
	return 0, fmt.Errorf("preview: output is not readable")
}

func (w *Writer) strip(input []byte) []byte {
	data := append(w.pending, input...)
	w.pending = nil
	var text bytes.Buffer
	for len(data) > 0 {
		at := bytes.Index(data, []byte(markerPrefix))
		if at < 0 {
			// Keep only a possible partial prefix, forwarding all ordinary text intact.
			keep := 0
			for n := 1; n < len(markerPrefix) && n <= len(data); n++ {
				if bytes.Equal(data[len(data)-n:], []byte(markerPrefix[:n])) {
					keep = n
				}
			}
			text.Write(data[:len(data)-keep])
			w.pending = append([]byte(nil), data[len(data)-keep:]...)
			break
		}
		text.Write(data[:at])
		data = data[at+len(markerPrefix):]
		end := bytes.IndexByte(data, '\a')
		if end < 0 {
			// Markers are small. Never buffer unbounded input on a malformed marker.
			if len(data) <= 128 {
				w.pending = append([]byte(markerPrefix), data...)
			}
			break
		}
		w.desired = parsePlacement(string(data[:end]))
		data = data[end+1:]
	}
	return text.Bytes()
}
func writeFull(out io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := out.Write(data)
		if n < 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
func (w *Writer) clearImage() string {
	if w.drawn.ID == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\x1b7")
	if w.protocol == Kitty {
		options := kitty.Options{Action: kitty.Delete, Delete: kitty.DeleteID, ID: ownedImageID, DeleteResources: true, Quite: 2}
		b.WriteString(ansi.KittyGraphics(nil, options.Options()...))
	} else if w.protocol == ITerm2 {
		for row := 0; row < w.drawn.Rows; row++ {
			fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[%dX", w.drawn.Row+row, w.drawn.Column, w.drawn.Columns)
		}
	}
	b.WriteString("\x1b8")
	w.drawn = Placement{}
	return b.String()
}
func (w *Writer) drawImage() []byte {
	p := w.desired
	if w.protocol == None || p.ID == 0 || w.registry == nil {
		return nil
	}
	img := w.registry.lookup(p.ID)
	if img == nil {
		return nil
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "\x1b7\x1b[%d;%dH", p.Row, p.Column)
	switch w.protocol {
	case Kitty:
		err := kitty.EncodeGraphics(&b, img, &kitty.Options{Action: kitty.TransmitAndPut, ID: ownedImageID, PlacementID: 1, Format: kitty.PNG, Transmission: kitty.Direct, Chunk: true, Quite: 2, DoNotMoveCursor: true})
		if err != nil {
			return nil
		}
	case ITerm2:
		var raw bytes.Buffer
		if png.Encode(&raw, img) != nil {
			return nil
		}
		b.WriteString(ansi.ITerm2(iterm2.File{Inline: true, Width: iterm2.Pixels(img.Bounds().Dx()), Height: iterm2.Pixels(img.Bounds().Dy()), Content: []byte(base64.StdEncoding.EncodeToString(raw.Bytes()))}))
	default:
		return nil
	}
	b.WriteString("\x1b8")
	w.drawn = p
	return b.Bytes()
}
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	text := w.strip(p)
	if len(text) == 0 && len(w.pending) > 0 {
		return len(p), nil
	}
	var frame bytes.Buffer
	frame.WriteString(w.clearImage())
	frame.Write(text)
	// Cleanup must precede Bubble Tea's exit from the alternate screen.
	if bytes.Contains(text, []byte("\x1b[?1049l")) || bytes.Contains(text, []byte("\x1b[?1047l")) {
		w.desired = Placement{}
		if w.registry != nil {
			w.registry.Clear()
		}
	} else {
		frame.Write(w.drawImage())
	}
	if err := writeFull(w.output, frame.Bytes()); err != nil {
		w.desired = Placement{}
		return 0, err
	}
	return len(p), nil
}

// Close clears owned graphics and registry data without closing stdout.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	cleanup := w.clearImage()
	w.desired = Placement{}
	if w.registry != nil {
		w.registry.Clear()
	}
	// A truncated private marker is discarded. An ordinary partial escape is not.
	if len(w.pending) > 0 && !bytes.HasPrefix(w.pending, []byte(markerPrefix)) {
		cleanup += string(w.pending)
	}
	w.pending = nil
	return writeFull(w.output, []byte(cleanup))
}
