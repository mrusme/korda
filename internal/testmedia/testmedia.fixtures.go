package testmedia

import (
	"image"
	"image/color"
)

var ExifTags = Tags{
	Make: "korda", Model: "Fixture One", LensModel: "Fixture Lens 4.2mm f/2.8", Description: "a fixture",
	Orientation: 6, DateTimeOriginal: "2024:05:06 07:08:09", SubSecTimeOriginal: "123", OffsetTimeOriginal: "+02:00",
	ModifyDate: "2024:05:07 10:11:12", GPS: &GPS{Latitude: 52.516272, Longitude: 13.377722},
	FNumber: [2]uint32{28, 10}, FocalLength: [2]uint32{42, 10}, ExposureTime: [2]uint32{1, 250}, ISO: 100,
	Rating: 4, ColorSpace: 1, PixelXDimension: 640, PixelYDimension: 480,
}

func ExifJPEG() ([]byte, error) {
	return JPEG(Pattern(640, 480), ExifTags.TIFF())
}

func PlainJPEG() ([]byte, error) {
	return JPEG(Pattern(300, 200), nil)
}

func PlainPNG() ([]byte, error) {
	return PNG(Pattern(400, 300))
}

func ExifHEIC() ([]byte, error) {
	tags := ExifTags
	tags.Model = "Fixture HEIC"
	tags.Orientation = 1
	tags.OffsetTimeOriginal = "-05:00"
	tags.DateTimeOriginal = "2023:12:24 18:30:00"
	tags.SubSecTimeOriginal = ""
	tags.BigEndian = true

	return HEIC(Pattern(800, 600), tags.TIFF())
}

func Mutate(shade uint8) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 120, 90))
	for y := range 90 {
		for x := range 120 {
			img.SetNRGBA(x, y, color.NRGBA{R: shade, G: uint8(x * 2), B: uint8(y * 2), A: 255})
		}
	}

	return JPEG(img, Tags{Make: "korda", Model: "Mutate", DateTimeOriginal: "2019:07:08 09:10:11"}.TIFF())
}
