package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Canvas wraps an image.RGBA with helper drawing functions
type Canvas struct {
	Img    *image.RGBA
	Width  int
	Height int
}

// NewCanvas creates a new Canvas
func NewCanvas(w, h int) *Canvas {
	return &Canvas{
		Img:    image.NewRGBA(image.Rect(0, 0, w, h)),
		Width:  w,
		Height: h,
	}
}

// FillVerticalGradient fills the canvas with a vertical linear gradient
func (c *Canvas) FillVerticalGradient(top, bottom color.RGBA) {
	for y := 0; y < c.Height; y++ {
		t := float64(y) / float64(c.Height)
		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)
		a := uint8(float64(top.A)*(1-t) + float64(bottom.A)*t)
		lineColor := color.RGBA{R: r, G: g, B: b, A: a}

		for x := 0; x < c.Width; x++ {
			c.Img.Set(x, y, lineColor)
		}
	}
}

// FillRect fills an axis-aligned rectangle
func (c *Canvas) FillRect(x, y, w, h int, col color.Color) {
	draw.Draw(c.Img, image.Rect(x, y, x+w, y+h), &image.Uniform{C: col}, image.Point{}, draw.Over)
}

// FillRoundedRect draws a rounded rectangle with given corner radius
func (c *Canvas) FillRoundedRect(x, y, w, h, radius int, col color.RGBA) {
	if radius <= 0 {
		c.FillRect(x, y, w, h, col)
		return
	}

	x1, y1 := x, y
	x2, y2 := x+w, y+h
	r := float64(radius)

	for py := y1; py < y2; py++ {
		for px := x1; px < x2; px++ {
			inCorner := false
			var dx, dy float64

			if px < x1+radius && py < y1+radius { // Top-left
				dx, dy = float64(px-(x1+radius)), float64(py-(y1+radius))
				inCorner = true
			} else if px >= x2-radius && py < y1+radius { // Top-right
				dx, dy = float64(px-(x2-radius-1)), float64(py-(y1+radius))
				inCorner = true
			} else if px < x1+radius && py >= y2-radius { // Bottom-left
				dx, dy = float64(px-(x1+radius)), float64(py-(y2-radius-1))
				inCorner = true
			} else if px >= x2-radius && py >= y2-radius { // Bottom-right
				dx, dy = float64(px-(x2-radius-1)), float64(py-(y2-radius-1))
				inCorner = true
			}

			if inCorner {
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= r {
					c.Img.Set(px, py, col)
				}
			} else {
				c.Img.Set(px, py, col)
			}
		}
	}
}

// DrawText draws text at (x, y) with an integer scale factor (1 = 7x13, 2 = 14x26, etc.)
func (c *Canvas) DrawText(text string, x, y int, col color.Color, scale int) {
	if scale <= 1 {
		point := fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y + 11),
		}
		d := &font.Drawer{
			Dst:  c.Img,
			Src:  image.NewUniform(col),
			Face: basicfont.Face7x13,
			Dot:  point,
		}
		d.DrawString(text)
		return
	}

	textW := len(text) * 7
	textH := 13
	temp := image.NewRGBA(image.Rect(0, 0, textW, textH))
	d := &font.Drawer{
		Dst:  temp,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(0, 11),
	}
	d.DrawString(text)

	for ty := 0; ty < textH; ty++ {
		for tx := 0; tx < textW; tx++ {
			cPx := temp.At(tx, ty)
			_, _, _, a := cPx.RGBA()
			if a > 0 {
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						destX := x + tx*scale + sx
						destY := y + ty*scale + sy
						if destX < c.Width && destY < c.Height {
							c.Img.Set(destX, destY, cPx)
						}
					}
				}
			}
		}
	}
}

// DrawGridPattern adds subtle tech grid lines to background
func (c *Canvas) DrawGridPattern(step int, col color.RGBA) {
	for x := 0; x < c.Width; x += step {
		for y := 0; y < c.Height; y++ {
			c.Img.Set(x, y, col)
		}
	}
	for y := 0; y < c.Height; y += step {
		for x := 0; x < c.Width; x++ {
			c.Img.Set(x, y, col)
		}
	}
}

// TruncateString ensures string does not exceed maxLen, adding ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
