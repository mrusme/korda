package media

import (
	"image"

	"github.com/kovidgoyal/imaging"
)

func Orient(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Transpose(img)
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Transverse(img)
	case 8:
		return imaging.Rotate90(img)
	}

	return img
}

func Swaps(orientation int) bool {
	return orientation >= 5 && orientation <= 8
}
