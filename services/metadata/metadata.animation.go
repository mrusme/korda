package metadata

import (
	"bytes"
	"encoding/binary"
	"io"

	"tty.fail/mrus/korda/services/metadata/probe"
)

const maxAnimationScan = 64 << 20

func animationSeconds(r io.ReaderAt, size int64, info *probe.Info) (float64, bool) {
	switch info.Format {
	case probe.FormatWebP:
		return webpDuration(r, size)
	case probe.FormatPNG:
		return apngDuration(r, size)
	}

	return 0, false
}

func webpDuration(r io.ReaderAt, size int64) (float64, bool) {
	header := make([]byte, 8)
	total := 0
	animated := false
	for off := int64(12); off+8 <= size && off < maxAnimationScan; {
		if _, err := r.ReadAt(header, off); err != nil {
			return 0, false
		}
		length := int64(binary.LittleEndian.Uint32(header[4:8]))
		switch string(header[:4]) {
		case "ANIM":
			animated = true
		case "ANMF":
			frame := make([]byte, 16)
			if _, err := r.ReadAt(frame, off+8); err != nil {
				return 0, false
			}
			total += int(frame[12]) | int(frame[13])<<8 | int(frame[14])<<16
		}
		off += 8 + length + length%2
	}
	if !animated {
		return 0, false
	}

	return float64(total) / 1000, true
}

func apngDuration(r io.ReaderAt, size int64) (float64, bool) {
	header := make([]byte, 8)
	var total float64
	animated := false
	for off := int64(8); off+8 <= size && off < maxAnimationScan; {
		if _, err := r.ReadAt(header, off); err != nil {
			return 0, false
		}
		length := int64(binary.BigEndian.Uint32(header[:4]))
		kind := header[4:8]
		switch {
		case bytes.Equal(kind, []byte("acTL")):
			animated = true
		case bytes.Equal(kind, []byte("fcTL")) && length >= 26:
			fc := make([]byte, 26)
			if _, err := r.ReadAt(fc, off+8); err != nil {
				return 0, false
			}
			num := float64(binary.BigEndian.Uint16(fc[20:22]))
			den := float64(binary.BigEndian.Uint16(fc[22:24]))
			if den == 0 {
				den = 100
			}
			total += num / den
		case bytes.Equal(kind, []byte("IDAT")) && !animated, bytes.Equal(kind, []byte("IEND")):
			if !animated {
				return 0, false
			}
		}
		off += 12 + length
	}
	if !animated {
		return 0, false
	}

	return total, true
}
