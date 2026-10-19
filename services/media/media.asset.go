package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/kovidgoyal/imaging"
	"go.n16f.net/thumbhash"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/services/metadata/exif"
	"tty.fail/mrus/korda/services/metadata/probe"
)

const thumbhashSize = 100

type Source struct {
	Path        string
	Info        *probe.Info
	TIFF        *exif.TIFF
	Orientation int
	Raw         bool
}

type Rendition struct {
	Path   string
	Width  int
	Height int
}

func decodeWith(format probe.Format, r io.Reader) (image.Image, error) {
	switch format {
	case probe.FormatJPEG:
		return jpeg.Decode(r)
	case probe.FormatPNG:
		return png.Decode(r)
	case probe.FormatGIF:
		return gif.Decode(r)
	case probe.FormatWebP:
		return webp.Decode(r)
	case probe.FormatBMP:
		return bmp.Decode(r)
	case probe.FormatTIFF:
		return tiff.Decode(r)
	case probe.FormatHEIF:
		return decodeHEIF(r)
	}

	return nil, errs.ErrImageUndecodable
}

func (m *Media) DecodeSource(src Source) (image.Image, error) {
	f, err := os.Open(src.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}

	var img image.Image
	switch {
	case src.Raw && src.TIFF != nil:
		preview := EmbeddedJPEG(f, st.Size(), src.TIFF)
		if preview == nil {
			return nil, errs.ErrImageUndecodable
		}
		img, err = jpeg.Decode(bytes.NewReader(preview))
	case src.Info.Format == probe.FormatHEIF:
		img, err = decodeHEIF(f)
		if err == nil {
			return img, nil
		}
	default:
		img, err = decodeWith(src.Info.Format, f)
	}
	if err != nil {
		if errors.Is(err, errs.ErrImageUndecodable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", errs.ErrImageUndecodable, err)
	}

	return Orient(img, src.Orientation), nil
}

func Inside(img image.Image, size int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= size && h <= size {
		return img
	}

	if w >= h {
		return imaging.Resize(img, size, max(1, h*size/w), imaging.Lanczos)
	}

	return imaging.Resize(img, max(1, w*size/h), size, imaging.Lanczos)
}

func Thumbhash(img image.Image) []byte {
	return thumbhash.EncodeImage(Inside(img, thumbhashSize))
}

func (m *Media) Render(img image.Image, relative string, size int, quality int) (Rendition, error) {
	out := Outside(img, size)
	if err := m.writeJPEG(out, relative, quality); err != nil {
		return Rendition{}, err
	}

	b := out.Bounds()
	return Rendition{Path: relative, Width: b.Dx(), Height: b.Dy()}, nil
}
