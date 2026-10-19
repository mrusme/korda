package media

import (
	"bytes"
	"io"

	"tty.fail/mrus/korda/services/metadata/exif"
)

type span struct {
	offset int64
	length int64
}

func jpegSpans(t *exif.TIFF) []span {
	var spans []span
	consider := func(d exif.IFD) {
		if d == nil {
			return
		}
		if off, ok := d[0x0201]; ok {
			if n, ok := d[0x0202]; ok {
				o, _ := off.Uint(0)
				l, _ := n.Uint(0)
				spans = append(spans, span{offset: int64(o), length: int64(l)})
			}
		}
		compression, _ := d[0x0103].Uint(0)
		if compression != 6 && compression != 7 {
			return
		}
		offsets, okO := d[0x0111]
		counts, okC := d[0x0117]
		if !okO || !okC || offsets.Count != 1 || counts.Count != 1 {
			return
		}
		o, _ := offsets.Uint(0)
		l, _ := counts.Uint(0)
		spans = append(spans, span{offset: int64(o), length: int64(l)})
	}

	consider(t.IFD0)
	consider(t.IFD1)
	for _, sub := range t.SubIFDs {
		consider(sub)
	}

	return spans
}

func EmbeddedJPEG(r io.ReaderAt, size int64, t *exif.TIFF) []byte {
	var best []byte
	for _, s := range jpegSpans(t) {
		if s.length <= int64(len(best)) || s.length < 4 || s.offset+s.length > size || s.length > 64<<20 {
			continue
		}
		data := make([]byte, s.length)
		if _, err := r.ReadAt(data, s.offset); err != nil {
			continue
		}
		if !bytes.HasPrefix(data, []byte{0xff, 0xd8}) {
			continue
		}
		best = data
	}

	return best
}
