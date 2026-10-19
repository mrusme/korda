//go:build heic

package media

import (
	"image"
	"io"

	"github.com/gen2brain/h265/heic"
)

const HEIC = true

func decodeHEIF(r io.Reader) (image.Image, error) {
	return heic.Decode(r, heic.Options{AutoRotate: true, Threads: 1})
}
