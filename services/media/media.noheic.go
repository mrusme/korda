//go:build !heic

package media

import (
	"image"
	"io"

	"tty.fail/mrus/korda/errs"
)

const HEIC = false

func decodeHEIF(r io.Reader) (image.Image, error) {
	return nil, errs.ErrImageUndecodable
}
