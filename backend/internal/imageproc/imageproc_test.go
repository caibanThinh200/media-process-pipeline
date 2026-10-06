package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"testing"

	_ "golang.org/x/image/webp"
)

func noisyImage(w, h int) *image.NRGBA {
	r := rand.New(rand.NewSource(1))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func decodeConfig(t *testing.T, data []byte) (image.Config, string) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a decodable image: %v", err)
	}
	return cfg, format
}

func TestOptimize_ResizesAndEncodesWebP(t *testing.T) {
	in := encodeJPEG(t, noisyImage(3000, 2000))
	out, ct, err := Optimize(in, Options{})
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if ct != ContentTypeWebP {
		t.Errorf("content type = %q, want %q", ct, ContentTypeWebP)
	}
	cfg, format := decodeConfig(t, out)
	if format != "webp" {
		t.Errorf("format = %q, want webp", format)
	}
	if cfg.Width != 1920 || cfg.Height != 1280 {
		t.Errorf("size = %dx%d, want 1920x1280", cfg.Width, cfg.Height)
	}
	if len(out) >= len(in) {
		t.Errorf("output (%d) not smaller than input (%d)", len(out), len(in))
	}
}

func TestOptimize_DoesNotUpscale(t *testing.T) {
	in := encodeJPEG(t, noisyImage(300, 200))
	out, _, err := Optimize(in, Options{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := decodeConfig(t, out)
	if cfg.Width != 300 || cfg.Height != 200 {
		t.Errorf("size = %dx%d, want 300x200", cfg.Width, cfg.Height)
	}
}

func TestOptimize_PNGInput(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, noisyImage(400, 400)); err != nil {
		t.Fatal(err)
	}
	out, ct, err := Optimize(b.Bytes(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if ct != ContentTypeWebP {
		t.Errorf("content type = %q", ct)
	}
	decodeConfig(t, out)
}

func TestOptimize_WatermarkChangesOutput(t *testing.T) {
	// Flat image so any difference comes from the watermark.
	flat := image.NewNRGBA(image.Rect(0, 0, 800, 600))
	for i := range flat.Pix {
		flat.Pix[i] = 40
	}
	in := encodeJPEG(t, flat)

	plain, _, err := Optimize(in, Options{Watermark: false})
	if err != nil {
		t.Fatal(err)
	}
	marked, _, err := Optimize(in, Options{Watermark: true, WatermarkText: "tAI"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(plain, marked) {
		t.Error("watermarked output is identical to unwatermarked output")
	}

	// Decode both and confirm differing pixels are in the bottom-right quadrant.
	pi, _, _ := image.Decode(bytes.NewReader(plain))
	mi, _, _ := image.Decode(bytes.NewReader(marked))
	diffBR, diffElsewhere := 0, 0
	b := pi.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if pi.At(x, y) != mi.At(x, y) {
				if x > b.Dx()/2 && y > b.Dy()/2 {
					diffBR++
				} else {
					diffElsewhere++
				}
			}
		}
	}
	if diffBR == 0 {
		t.Error("no pixel changes in bottom-right quadrant")
	}
	if diffElsewhere > diffBR/10 {
		t.Errorf("too many changes outside bottom-right: %d vs %d", diffElsewhere, diffBR)
	}
}

func TestOptimize_CorruptInput(t *testing.T) {
	if _, _, err := Optimize([]byte("not an image"), Options{}); err == nil {
		t.Fatal("expected error for corrupt input")
	}
}

func TestApplyOrientation_Rotate90(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	out := applyOrientation(src, 6)
	if b := out.Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Errorf("orientation 6 size = %dx%d, want 2x4", b.Dx(), b.Dy())
	}
}
