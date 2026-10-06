package imageproc

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	// watermarkWidthRatio is the approximate text height relative to image width.
	watermarkWidthRatio = 0.04
	minFontSize         = 12.0
	DefaultOpacity      = 0.5
)

var (
	fontOnce sync.Once
	parsed   *opentype.Font
	fontErr  error
)

func loadFont() (*opentype.Font, error) {
	fontOnce.Do(func() { parsed, fontErr = opentype.Parse(gobold.TTF) })
	return parsed, fontErr
}

// drawWatermark draws semi-transparent white text (with a dark shadow for
// contrast) in the bottom-right corner and returns a new image.
func drawWatermark(img image.Image, text string, opacity float64) (image.Image, error) {
	if opacity <= 0 || opacity > 1 {
		opacity = DefaultOpacity
	}
	f, err := loadFont()
	if err != nil {
		return nil, err
	}

	dst := toNRGBA(img)
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()

	size := float64(w) * watermarkWidthRatio
	if size < minFontSize {
		size = minFontSize
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	defer face.Close()

	margin := int(size * 0.6)
	adv := font.MeasureString(face, text)
	x := w - margin - adv.Ceil()
	if x < 0 {
		x = 0
	}
	y := h - margin - face.Metrics().Descent.Ceil()
	if y < face.Metrics().Ascent.Ceil() {
		y = face.Metrics().Ascent.Ceil()
	}

	alpha := func(a float64) uint8 { return uint8(a*255 + 0.5) }
	shadowOff := int(size/16) + 1

	drawText := func(c color.NRGBA, px, py int) {
		d := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(c),
			Face: face,
			Dot:  fixed.P(px, py),
		}
		d.DrawString(text)
	}
	drawText(color.NRGBA{0, 0, 0, alpha(opacity * 0.6)}, x+shadowOff, y+shadowOff)
	drawText(color.NRGBA{255, 255, 255, alpha(opacity)}, x, y)

	return dst, nil
}

var _ draw.Image = (*image.NRGBA)(nil)
