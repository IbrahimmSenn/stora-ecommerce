package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/disintegration/imaging"
)

func sampleJPEG(t *testing.T, w, h int) *bytes.Buffer {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode sample: %v", err)
	}
	return &buf
}

func TestProcess_GeneratesThreeVariants(t *testing.T) {
	dir := t.TempDir()
	p, err := New(dir, "/media")
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}

	// A wide source so we can confirm the longest edge is clamped to the box.
	v, err := p.Process("abc123", sampleJPEG(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}

	if v.ThumbnailURL != "/media/abc123_thumb.jpg" ||
		v.CardURL != "/media/abc123_card.jpg" ||
		v.FullURL != "/media/abc123_full.jpg" {
		t.Fatalf("unexpected variant URLs: %+v", v)
	}

	for suffix, box := range map[string]int{"thumb": ThumbBox, "card": CardBox, "full": FullBox} {
		path := filepath.Join(dir, "abc123_"+suffix+".jpg")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("variant %s not written: %v", suffix, err)
		}
		img, err := imaging.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", suffix, err)
		}
		b := img.Bounds()
		if b.Dx() > box || b.Dy() > box {
			t.Fatalf("%s variant %dx%d exceeds box %d", suffix, b.Dx(), b.Dy(), box)
		}
		// Aspect ratio (2:1) preserved → wide variant hits the box on width.
		if b.Dx() != box {
			t.Errorf("%s width expected %d, got %d", suffix, box, b.Dx())
		}
	}
}

func TestProcess_RejectsNonImage(t *testing.T) {
	p, err := New(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}
	_, err = p.Process("x", strings.NewReader("this is not an image"))
	if err != ErrNotAnImage {
		t.Fatalf("expected ErrNotAnImage, got %v", err)
	}
}
