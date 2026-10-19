package probe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type Format string

const (
	FormatUnknown Format = ""
	FormatJPEG    Format = "jpeg"
	FormatPNG     Format = "png"
	FormatGIF     Format = "gif"
	FormatWebP    Format = "webp"
	FormatBMP     Format = "bmp"
	FormatTIFF    Format = "tiff"
	FormatHEIF    Format = "heif"
	FormatAVIF    Format = "avif"
	FormatCR3     Format = "cr3"
	FormatJXL     Format = "jxl"

	maxHeader   = 64 << 10
	maxMetadata = 4 << 20
	maxTIFFHead = 8 << 20
)

var ErrTruncated = errors.New("truncated file")

type Info struct {
	Format      Format
	Width       int
	Height      int
	Rotation    int
	HasRotation bool
	Mirror      int
	EXIF        []byte
	XMP         []byte
	CR3Exif     []byte
	CR3GPS      []byte
}

func readAt(r io.ReaderAt, off int64, n int64, size int64) ([]byte, error) {
	if off < 0 || off >= size {
		return nil, ErrTruncated
	}
	n = min(n, size-off)
	buf := make([]byte, n)
	read, err := r.ReadAt(buf, off)
	if read == int(n) {
		return buf, nil
	}
	if err == nil {
		err = ErrTruncated
	}
	return buf[:read], err
}

func Probe(r io.ReaderAt, size int64) (*Info, error) {
	head, err := readAt(r, 0, 64, size)
	if err != nil && len(head) < 12 {
		return &Info{}, err
	}

	switch {
	case bytes.HasPrefix(head, []byte{0xff, 0xd8, 0xff}):
		return probeJPEG(r, size)
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return probePNG(r, size)
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return &Info{
			Format: FormatGIF,
			Width:  int(binary.LittleEndian.Uint16(head[6:8])),
			Height: int(binary.LittleEndian.Uint16(head[8:10])),
		}, nil
	case bytes.HasPrefix(head, []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return probeWebP(r, size)
	case bytes.HasPrefix(head, []byte("BM")) && len(head) >= 26:
		h := int(int32(binary.LittleEndian.Uint32(head[22:26])))
		if h < 0 {
			h = -h
		}
		return &Info{Format: FormatBMP, Width: int(int32(binary.LittleEndian.Uint32(head[18:22]))), Height: h}, nil
	case bytes.HasPrefix(head, []byte("II")) || bytes.HasPrefix(head, []byte("MM")):
		return probeTIFF(r, size)
	case bytes.Equal(head[4:8], []byte("ftyp")):
		return probeISOBMFF(r, size, head)
	case bytes.HasPrefix(head, []byte{0xff, 0x0a}), bytes.HasPrefix(head, []byte("\x00\x00\x00\x0cJXL \r\n\x87\n")):
		return &Info{Format: FormatJXL}, nil
	}

	return &Info{}, nil
}

func probeTIFF(r io.ReaderAt, size int64) (*Info, error) {
	data, err := readAt(r, 0, maxTIFFHead, size)
	if err != nil && len(data) == 0 {
		return &Info{Format: FormatTIFF}, err
	}

	return &Info{Format: FormatTIFF, EXIF: data}, nil
}
