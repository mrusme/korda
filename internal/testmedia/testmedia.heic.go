//go:build heic

package testmedia

import (
	"bytes"
	"image"

	"github.com/gen2brain/h265/heic"
)

const HEICSupported = true

func HEIC(img image.Image, exif []byte) ([]byte, error) {
	var out bytes.Buffer
	err := heic.Encode(&out, img, heic.EncodeOptions{Quality: 80, Exif: exif})
	return out.Bytes(), err
}
