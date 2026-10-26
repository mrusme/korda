package media

import (
	"image"

	"github.com/kovidgoyal/imaging"
)

func Crop(img image.Image, x, y, width, height int) image.Image {
	min := img.Bounds().Min
	return imaging.Crop(img, image.Rect(min.X+x, min.Y+y, min.X+x+width, min.Y+y+height))
}

func RotateClockwise(img image.Image, angle int) image.Image {
	switch angle {
	case 90:
		return imaging.Rotate270(img)
	case 180:
		return imaging.Rotate180(img)
	case 270:
		return imaging.Rotate90(img)
	}

	return img
}

func Mirror(img image.Image, horizontal bool) image.Image {
	if horizontal {
		return imaging.FlipH(img)
	}

	return imaging.FlipV(img)
}
