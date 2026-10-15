package preview

import "math"

// Size describes bitmap pixels and the whole terminal cells reserved for them.
type Size struct{ Width, Height, Columns, Rows int }

// FitPixels fits without cropping or stretching, with a preferred height of
// 150 pixels and a maximum edge of 2048. Tiny unusable results are omitted.
func FitPixels(width, height, maxWidth, maxHeight int) (int, int) {
	if width <= 0 || height <= 0 || maxWidth <= 0 || maxHeight <= 0 {
		return 0, 0
	}
	scale := min(150/float64(height), float64(maxWidth)/float64(width), float64(maxHeight)/float64(height), 2048/float64(width), 2048/float64(height))
	return int(math.Floor(float64(width)*scale + 1e-9)), int(math.Floor(float64(height)*scale + 1e-9))
}
func (g Geometry) Size(width, height int) Size {
	if !g.Valid(g.Columns, g.Rows) || width < 1 || height < 1 {
		return Size{}
	}
	cw, ch := float64(g.PixelWidth)/float64(g.Columns), float64(g.PixelHeight)/float64(g.Rows)
	return Size{width, height, int(math.Ceil(float64(width) / cw)), int(math.Ceil(float64(height) / ch))}
}

// PixelBox gives the available pixels after the caller reserves its text rows.
func (g Geometry) PixelBox(columns, rows int) (int, int) {
	if !g.Valid(g.Columns, g.Rows) || columns < 1 || rows < 1 {
		return 0, 0
	}
	return int(float64(min(columns, g.Columns)) * float64(g.PixelWidth) / float64(g.Columns)), int(float64(min(rows, g.Rows)) * float64(g.PixelHeight) / float64(g.Rows))
}

// FrameBox reserves every text row and a blank gap below the preview before
// calculating its pixel budget. Header/result/footer counts include wrapping.
func (g Geometry) FrameBox(contentColumns, headerRows, resultRows, footerRows int) (width, height, row int) {
	available := g.Rows - (headerRows + 1 + resultRows + 1 + footerRows + 1)
	width, height = g.PixelBox(contentColumns, available)
	return width, height, headerRows + 2
}
