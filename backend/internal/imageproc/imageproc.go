// Package imageproc optimizes raw images: orient, resize, watermark and
// re-encode as lossy WebP.
package imageproc

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	// ContentTypeWebP is the content type of every successful output.
	ContentTypeWebP = "image/webp"

	DefaultMaxDimension = 1920
	DefaultQuality      = 80
	// MaxPixels rejects decompression bombs before a full decode.
	MaxPixels = 50_000_000
)

// ErrTooLarge is returned when the image exceeds MaxPixels.
var ErrTooLarge = errors.New("image exceeds maximum pixel count")

// Options controls Optimize. Zero values for MaxDimension/Quality use defaults.
type Options struct {
	MaxDimension int
	Quality      int

	Watermark        bool
	WatermarkText    string
	WatermarkOpacity float64 // 0..1
}

// Optimize decodes data, applies EXIF orientation, resizes so the longest side
// is at most MaxDimension (never upscaling), optionally watermarks, and
// encodes as lossy WebP. It returns the encoded bytes and content type.
func Optimize(data []byte, opts Options) ([]byte, string, error) {
	if opts.MaxDimension <= 0 {
		opts.MaxDimension = DefaultMaxDimension
	}
	if opts.Quality <= 0 {
		opts.Quality = DefaultQuality
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode config: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return nil, "", ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode: %w", err)
	}

	img := applyOrientation(src, jpegOrientation(data))
	img = resize(img, opts.MaxDimension)

	if opts.Watermark && opts.WatermarkText != "" {
		img, err = drawWatermark(img, opts.WatermarkText, opts.WatermarkOpacity)
		if err != nil {
			return nil, "", fmt.Errorf("watermark: %w", err)
		}
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: opts.Quality}); err != nil {
		return nil, "", fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), ContentTypeWebP, nil
}

// resize scales img down so its longest side is at most maxDim.
func resize(img image.Image, maxDim int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}
	var nw, nh int
	if w >= h {
		nw = maxDim
		nh = h * maxDim / w
	} else {
		nh = maxDim
		nw = w * maxDim / h
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// toNRGBA returns a zero-origin NRGBA copy of img.
func toNRGBA(img image.Image) *image.NRGBA {
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

// jpegOrientation returns the EXIF orientation (1-8) of a JPEG, or 1.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		if segLen < 2 || i+2+segLen > len(data) {
			return 1
		}
		if marker == 0xE1 { // APP1
			seg := data[i+4 : i+2+segLen]
			if len(seg) > 6 && string(seg[:6]) == "Exif\x00\x00" {
				return parseTIFFOrientation(seg[6:])
			}
		}
		i += 2 + segLen
	}
	return 1
}

func parseTIFFOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var u16 func([]byte) int
	var u32 func([]byte) int
	switch string(t[:2]) {
	case "II":
		u16 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 }
		u32 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) int { return int(b[0])<<8 | int(b[1]) }
		u32 = func(b []byte) int { return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3]) }
	default:
		return 1
	}
	ifd := u32(t[4:8])
	if ifd < 8 || ifd+2 > len(t) {
		return 1
	}
	n := u16(t[ifd : ifd+2])
	for k := 0; k < n; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if u16(t[e:e+2]) == 0x0112 { // Orientation tag
			if v := u16(t[e+8 : e+10]); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation transforms img according to the EXIF orientation value.
func applyOrientation(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	src := toNRGBA(img)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	swap := o >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			dst.SetNRGBA(nx, ny, src.NRGBAAt(x, y))
		}
	}
	return dst
}
