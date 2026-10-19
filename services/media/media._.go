package media

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/kovidgoyal/imaging"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/services/mediaroot"
)

type Media struct {
	root    *mediaroot.MediaRoot
	size    int
	quality int
}

func New(root *mediaroot.MediaRoot, size int, quality int) (*Media, error) {
	m := new(Media)
	m.root = root
	m.size = size
	m.quality = quality

	return m, nil
}

func (m *Media) Startup() error {
	return nil
}

func (m *Media) Shutdown() error {
	return nil
}

func (m *Media) Decode(r io.Reader) (image.Image, error) {
	img, err := imaging.Decode(r, imaging.Backends(imaging.GO_IMAGE))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errs.ErrImageUndecodable, err)
	}

	return img, nil
}

func Outside(img image.Image, size int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	shorter := min(w, h)
	if shorter <= size || shorter == 0 {
		return img
	}

	scale := float64(size) / float64(shorter)
	width := max(1, int(math.Round(float64(w)*scale)))
	height := max(1, int(math.Round(float64(h)*scale)))

	return imaging.Resize(img, width, height, imaging.Lanczos)
}

func (m *Media) WriteJPEG(img image.Image, relative string) (err error) {
	abs, err := m.root.Abs(relative)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return err
	}

	f, err := os.CreateTemp(filepath.Dir(abs), ".korda-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	if err = jpeg.Encode(f, img, &jpeg.Options{Quality: m.quality}); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	return os.Rename(f.Name(), abs)
}

func (m *Media) Thumbnail(src io.Reader, relative string) error {
	img, err := m.Decode(src)
	if err != nil {
		return err
	}

	return m.WriteJPEG(Outside(img, m.size), relative)
}
