package testmedia

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

type GPS struct {
	Latitude  float64
	Longitude float64
}

type Tags struct {
	Make               string
	Model              string
	LensModel          string
	Description        string
	Orientation        uint16
	DateTimeOriginal   string
	SubSecTimeOriginal string
	OffsetTimeOriginal string
	CreateDate         string
	ModifyDate         string
	GPS                *GPS
	FNumber            [2]uint32
	FocalLength        [2]uint32
	ExposureTime       [2]uint32
	ISO                uint16
	Rating             uint16
	ColorSpace         uint16
	PixelXDimension    uint32
	PixelYDimension    uint32
	ExposureBiasNum    int32
	ExposureBiasDen    int32
	BigEndian          bool
}

func (t Tags) TIFF() []byte {
	var order binary.ByteOrder = binary.LittleEndian
	if t.BigEndian {
		order = binary.BigEndian
	}

	var ifd0, exif, gps ifd
	if t.Make != "" {
		ifd0 = append(ifd0, ascii(0x010f, t.Make))
	}
	if t.Model != "" {
		ifd0 = append(ifd0, ascii(0x0110, t.Model))
	}
	if t.Description != "" {
		ifd0 = append(ifd0, ascii(0x010e, t.Description))
	}
	if t.Orientation != 0 {
		ifd0 = append(ifd0, short(order, 0x0112, t.Orientation))
	}
	if t.ModifyDate != "" {
		ifd0 = append(ifd0, ascii(0x0132, t.ModifyDate))
	}
	if t.Rating != 0 {
		ifd0 = append(ifd0, short(order, 0x4746, t.Rating))
	}

	if t.DateTimeOriginal != "" {
		exif = append(exif, ascii(0x9003, t.DateTimeOriginal))
	}
	if t.CreateDate != "" {
		exif = append(exif, ascii(0x9004, t.CreateDate))
	}
	if t.SubSecTimeOriginal != "" {
		exif = append(exif, ascii(0x9291, t.SubSecTimeOriginal))
	}
	if t.OffsetTimeOriginal != "" {
		exif = append(exif, ascii(0x9011, t.OffsetTimeOriginal))
	}
	if t.LensModel != "" {
		exif = append(exif, ascii(0xa434, t.LensModel))
	}
	if t.FNumber[1] != 0 {
		exif = append(exif, rational(order, 0x829d, t.FNumber))
	}
	if t.FocalLength[1] != 0 {
		exif = append(exif, rational(order, 0x920a, t.FocalLength))
	}
	if t.ExposureTime[1] != 0 {
		exif = append(exif, rational(order, 0x829a, t.ExposureTime))
	}
	if t.ExposureBiasDen != 0 {
		exif = append(exif, srational(order, 0x9204, t.ExposureBiasNum, t.ExposureBiasDen))
	}
	if t.ISO != 0 {
		exif = append(exif, short(order, 0x8827, t.ISO))
	}
	if t.ColorSpace != 0 {
		exif = append(exif, short(order, 0xa001, t.ColorSpace))
	}
	if t.PixelXDimension != 0 {
		exif = append(exif, long(order, 0xa002, t.PixelXDimension), long(order, 0xa003, t.PixelYDimension))
	}

	if t.GPS != nil {
		latRef, lonRef := "N", "E"
		if t.GPS.Latitude < 0 {
			latRef = "S"
		}
		if t.GPS.Longitude < 0 {
			lonRef = "W"
		}
		gps = append(gps,
			entry{tag: 0x0000, typ: typeByte, count: 4, data: []byte{2, 3, 0, 0}},
			ascii(0x0001, latRef),
			rational(order, 0x0002, degrees(t.GPS.Latitude)...),
			ascii(0x0003, lonRef),
			rational(order, 0x0004, degrees(t.GPS.Longitude)...),
		)
	}

	l := &layout{order: order}
	if order == binary.LittleEndian {
		l.buf.WriteString("II")
	} else {
		l.buf.WriteString("MM")
	}
	b2 := make([]byte, 2)
	order.PutUint16(b2, 42)
	l.buf.Write(b2)
	b4 := make([]byte, 4)
	order.PutUint32(b4, 8)
	l.buf.Write(b4)

	pointers := 0
	if len(exif) > 0 {
		pointers++
	}
	if len(gps) > 0 {
		pointers++
	}
	at := 8 + l.size(ifd0) + 12*pointers
	exifAt := at
	gpsAt := at
	if len(exif) > 0 {
		gpsAt += l.size(exif)
		ifd0 = append(ifd0, long(order, 0x8769, uint32(exifAt)))
	}
	if len(gps) > 0 {
		ifd0 = append(ifd0, long(order, 0x8825, uint32(gpsAt)))
	}

	l.write(ifd0, 8, 0)
	if len(exif) > 0 {
		l.write(exif, l.buf.Len(), 0)
	}
	if len(gps) > 0 {
		l.write(gps, l.buf.Len(), 0)
	}

	return l.buf.Bytes()
}

func Pattern(width int, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			c := color.NRGBA{R: uint8(40 + x*160/width), G: uint8(60 + y*140/height), B: 120, A: 255}
			if x < width/4 && y < height/4 {
				c = color.NRGBA{R: 230, G: 20, B: 20, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func JPEG(img image.Image, exif []byte) ([]byte, error) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	if len(exif) == 0 {
		return encoded.Bytes(), nil
	}

	payload := append([]byte("Exif\x00\x00"), exif...)
	segment := []byte{0xff, 0xe1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}

	var out bytes.Buffer
	out.Write(encoded.Bytes()[:2])
	out.Write(segment)
	out.Write(payload)
	out.Write(encoded.Bytes()[2:])
	return out.Bytes(), nil
}

func PNG(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	err := png.Encode(&out, img)
	return out.Bytes(), err
}
