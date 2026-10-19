package probe

import (
	"bytes"
	"encoding/binary"
	"io"
)

func uint24(b []byte) int {
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16
}

func probeWebP(r io.ReaderAt, size int64) (*Info, error) {
	info := &Info{Format: FormatWebP}

	off := int64(12)
	for off+8 <= size {
		head, err := readAt(r, off, 8, size)
		if err != nil || len(head) < 8 {
			return info, nil
		}
		kind := string(head[0:4])
		length := int64(binary.LittleEndian.Uint32(head[4:8]))

		switch kind {
		case "VP8X":
			if data, err := readAt(r, off+8, 10, size); err == nil && len(data) == 10 {
				info.Width = uint24(data[4:7]) + 1
				info.Height = uint24(data[7:10]) + 1
			}
		case "VP8 ":
			if data, err := readAt(r, off+8, 10, size); err == nil && len(data) == 10 && info.Width == 0 {
				info.Width = int(binary.LittleEndian.Uint16(data[6:8]) & 0x3fff)
				info.Height = int(binary.LittleEndian.Uint16(data[8:10]) & 0x3fff)
			}
		case "VP8L":
			if data, err := readAt(r, off+8, 5, size); err == nil && len(data) == 5 && data[0] == 0x2f && info.Width == 0 {
				bits := binary.LittleEndian.Uint32(data[1:5])
				info.Width = int(bits&0x3fff) + 1
				info.Height = int((bits>>14)&0x3fff) + 1
			}
		case "EXIF":
			if length <= maxMetadata {
				if data, err := readAt(r, off+8, length, size); err == nil {
					info.EXIF = bytes.TrimPrefix(data, exifPrefix)
				}
			}
		case "XMP ":
			if length <= maxMetadata {
				if data, err := readAt(r, off+8, length, size); err == nil {
					info.XMP = data
				}
			}
		}

		off += 8 + length + length%2
	}

	return info, nil
}
