// Package imageproc turns an uploaded product image into the three sizes the
// storefront serves: a thumbnail (cart/nav), a card (listing grids), and a
// full-size image (product detail). Variants preserve aspect ratio (fit within
// a box, no crop) and are re-encoded as JPEG for consistent, compact delivery.
package imageproc

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// Variant box sizes (longest edge). Images are fit within a square box so the
// whole product stays visible regardless of orientation.
const (
	ThumbBox = 200
	CardBox  = 600
	FullBox  = 1400
)

// maxDecodePixels guards against decompression-bomb uploads. ~40 megapixels is
// generous for product photos.
const maxDecodePixels = 40 * 1000 * 1000

// ErrNotAnImage is returned when the upload can't be decoded as an image.
var ErrNotAnImage = errors.New("file is not a valid image")

// Variants holds the public URLs of the three generated sizes.
type Variants struct {
	ThumbnailURL string
	CardURL      string
	FullURL      string
}

// Processor writes variants into baseDir and exposes them under publicPrefix
// (e.g. baseDir="./uploads", publicPrefix="/media").
type Processor struct {
	baseDir      string
	publicPrefix string
	jpegQuality  int
}

func New(baseDir, publicPrefix string) (*Processor, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Processor{baseDir: baseDir, publicPrefix: publicPrefix, jpegQuality: 82}, nil
}

// Process decodes src and writes <id>_thumb.jpg / _card.jpg / _full.jpg, then
// returns their public URLs. id must be a filesystem- and URL-safe token.
func (p *Processor) Process(id string, src io.Reader) (*Variants, error) {
	img, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrNotAnImage
	}
	if b := img.Bounds(); b.Dx()*b.Dy() > maxDecodePixels {
		return nil, fmt.Errorf("image is too large")
	}

	out := &Variants{}
	for _, s := range []struct {
		suffix string
		box    int
		dst    *string
	}{
		{"thumb", ThumbBox, &out.ThumbnailURL},
		{"card", CardBox, &out.CardURL},
		{"full", FullBox, &out.FullURL},
	} {
		resized := imaging.Fit(img, s.box, s.box, imaging.Lanczos)
		name := id + "_" + s.suffix + ".jpg"
		if err := imaging.Save(resized, filepath.Join(p.baseDir, name), imaging.JPEGQuality(p.jpegQuality)); err != nil {
			return nil, fmt.Errorf("save %s variant: %w", s.suffix, err)
		}
		*s.dst = p.publicPrefix + "/" + name
	}
	return out, nil
}
