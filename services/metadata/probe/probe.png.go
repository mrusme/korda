package probe

import (
	"bytes"
	"encoding/binary"
	"io"
)

var xmpKeyword = []byte("XML:com.adobe.xmp\x00")

func probePNG(r io.ReaderAt, size int64) (*Info, error) {
	info := &Info{Format: FormatPNG}

	off := int64(8)
	for off+8 <= size {
		head, err := readAt(r, off, 8, size)
		if err != nil || len(head) < 8 {
			return info, nil
		}
		length := int64(binary.BigEndian.Uint32(head[0:4]))
		kind := string(head[4:8])

		switch kind {
		case "IHDR":
			data, err := readAt(r, off+8, 8, size)
			if err == nil {
				info.Width = int(binary.BigEndian.Uint32(data[0:4]))
				info.Height = int(binary.BigEndian.Uint32(data[4:8]))
			}
		case "eXIf":
			if length <= maxMetadata {
				if data, err := readAt(r, off+8, length, size); err == nil {
					info.EXIF = bytes.TrimPrefix(data, exifPrefix)
				}
			}
		case "iTXt":
			if length <= maxMetadata {
				if data, err := readAt(r, off+8, length, size); err == nil && bytes.HasPrefix(data, xmpKeyword) {
					rest := data[len(xmpKeyword):]
					if len(rest) >= 2 && rest[0] == 0 {
						rest = rest[2:]
						if i := bytes.IndexByte(rest, 0); i >= 0 {
							rest = rest[i+1:]
							if i = bytes.IndexByte(rest, 0); i >= 0 {
								info.XMP = rest[i+1:]
							}
						}
					}
				}
			}
		case "IDAT", "IEND":
			if info.EXIF != nil || kind == "IEND" {
				return info, nil
			}
		}

		off += 12 + length
	}

	return info, nil
}
