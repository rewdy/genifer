package preview

import "testing"

func TestFitPixels(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		w, h, mw, mh, ow, oh int
	}{
		{"landscape", 1600, 900, 800, 300, 266, 150},
		{"portrait", 900, 1600, 800, 300, 84, 150},
		{"square", 1000, 1000, 800, 300, 150, 150},
		{"wide", 4000, 100, 200, 300, 200, 5},
		{"short", 1000, 1000, 800, 45, 45, 45},
		{"no room", 100, 100, 0, 200, 0, 0},
		{"extreme", 1, 100000, 200, 200, 0, 150},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, h := FitPixels(tc.w, tc.h, tc.mw, tc.mh)
			if w != tc.ow || h != tc.oh {
				t.Fatalf("%d x %d", w, h)
			}
		})
	}
	g := Geometry{80, 24, 800, 480}
	if got := g.Size(151, 151); got != (Size{151, 151, 16, 8}) {
		t.Fatal(got)
	}
	if w, h := g.PixelBox(76, 5); w != 760 || h != 100 {
		t.Fatal(w, h)
	}
}

func TestFrameBoxReservesText(t *testing.T) {
	g := Geometry{80, 40, 800, 800}
	w, h, row := g.FrameBox(76, 10, 4, 3)
	if w != 760 || h != 400 || row != 12 {
		t.Fatal(w, h, row)
	}
	_, wrapped, _ := g.FrameBox(76, 10, 8, 5)
	if wrapped >= h {
		t.Fatal("wrapped details/footer not reserved")
	}
	_, none, _ := (Geometry{80, 16, 800, 320}).FrameBox(76, 10, 4, 3)
	if none != 0 {
		t.Fatal("short terminal has no preview room")
	}
}
