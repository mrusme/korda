//go:build !heic

package testmedia

import (
	"errors"
	"image"
)

const HEICSupported = false

var ErrNoHEIC = errors.New("HEIC needs the build tag heic")

func HEIC(image.Image, []byte) ([]byte, error) {
	return nil, ErrNoHEIC
}
