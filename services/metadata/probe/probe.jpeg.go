package probe

import (
	"bytes"
	"encoding/binary"
	"io"
)

var (
	exifPrefix = []byte("Exif\x00\x00")
	xmpPrefix  = []byte("http://ns.adobe.com/xap/1.0/\x00")
)

func probeJPEG(r io.ReaderAt, size int64) (*Info, error) {
	info := &Info{Format: FormatJPEG}

	off := int64(2)
	for off+4 <= size {
		marker, err := readAt(r, off, 4, size)
		if err != nil || len(marker) < 4 {
			return info, nil
		}
		if marker[0] != 0xff {
			return info, nil
		}
		code := marker[1]
		if code == 0xff {
			off++
			continue
		}
		if code == 0xd8 || code == 0x01 || (code >= 0xd0 && code <= 0xd7) {
			off += 2
			continue
		}
		if code == 0xd9 || code == 0xda {
			return info, nil
		}

		length := int64(binary.BigEndian.Uint16(marker[2:4]))
		if length < 2 {
			return info, nil
		}

		switch {
		case code == 0xe1 && length <= maxMetadata:
			payload, err := readAt(r, off+4, length-2, size)
			if err == nil {
				switch {
				case bytes.HasPrefix(payload, exifPrefix) && info.EXIF == nil:
					info.EXIF = payload[len(exifPrefix):]
				case bytes.HasPrefix(payload, xmpPrefix) && info.XMP == nil:
					info.XMP = payload[len(xmpPrefix):]
				}
			}
		case code >= 0xc0 && code <= 0xcf && code != 0xc4 && code != 0xc8 && code != 0xcc:
			sof, err := readAt(r, off+4, 5, size)
			if err == nil && info.Width == 0 {
				info.Height = int(binary.BigEndian.Uint16(sof[1:3]))
				info.Width = int(binary.BigEndian.Uint16(sof[3:5]))
			}
		}

		off += 2 + length
	}

	return info, nil
}
