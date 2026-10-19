package probe

import (
	"bytes"
	"encoding/binary"
	"io"
)

var canonUUID = []byte{0x85, 0xc0, 0xb6, 0x87, 0x82, 0x0f, 0x11, 0xe0, 0x81, 0x11, 0xf4, 0xce, 0x46, 0x2b, 0x6a, 0x48}

type box struct {
	kind    string
	start   int64
	content int64
	end     int64
}

func readBox(r io.ReaderAt, off int64, limit int64, size int64) (box, bool) {
	head, err := readAt(r, off, 16, size)
	if err != nil && len(head) < 8 {
		return box{}, false
	}

	n := int64(binary.BigEndian.Uint32(head[0:4]))
	b := box{kind: string(head[4:8]), start: off, content: off + 8}
	switch n {
	case 0:
		n = limit - off
	case 1:
		if len(head) < 16 {
			return box{}, false
		}
		n = int64(binary.BigEndian.Uint64(head[8:16]))
		b.content = off + 16
	}
	if n < b.content-off || off+n > limit {
		return box{}, false
	}
	b.end = off + n

	return b, true
}

func children(r io.ReaderAt, from int64, to int64, size int64) []box {
	var out []box
	for off := from; off+8 <= to; {
		b, ok := readBox(r, off, to, size)
		if !ok {
			break
		}
		out = append(out, b)
		off = b.end
	}

	return out
}

type reader struct {
	data []byte
	pos  int
	bad  bool
}

func (rd *reader) take(n int) []byte {
	if n < 0 || rd.pos+n > len(rd.data) {
		rd.bad = true
		return make([]byte, max(n, 0))
	}
	b := rd.data[rd.pos : rd.pos+n]
	rd.pos += n
	return b
}

func (rd *reader) u8() uint64  { return uint64(rd.take(1)[0]) }
func (rd *reader) u16() uint64 { return uint64(binary.BigEndian.Uint16(rd.take(2))) }
func (rd *reader) u32() uint64 { return uint64(binary.BigEndian.Uint32(rd.take(4))) }

func (rd *reader) sized(n int) uint64 {
	switch n {
	case 0:
		return 0
	case 4:
		return rd.u32()
	case 8:
		return binary.BigEndian.Uint64(rd.take(8))
	}
	rd.bad = true
	return 0
}

func (rd *reader) cstring() string {
	rest := rd.data[min(rd.pos, len(rd.data)):]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		rd.pos = len(rd.data)
		return string(rest)
	}
	rd.pos += i + 1
	return string(rest[:i])
}

type extent struct {
	offset uint64
	length uint64
}

type location struct {
	method  uint64
	base    uint64
	extents []extent
}

type item struct {
	kind        string
	contentType string
}

type heif struct {
	primary    uint64
	items      map[uint64]item
	locations  map[uint64]location
	properties []box
	assoc      map[uint64][]int
	idat       box
}

func probeISOBMFF(r io.ReaderAt, size int64, head []byte) (*Info, error) {
	brands := string(head[8:12])
	if ftyp, ok := readBox(r, 0, size, size); ok && ftyp.end-ftyp.content <= 256 {
		if data, err := readAt(r, ftyp.content, ftyp.end-ftyp.content, size); err == nil {
			brands = string(data)
		}
	}

	info := &Info{}
	switch {
	case bytes.Contains([]byte(brands), []byte("crx ")):
		info.Format = FormatCR3
		probeCR3(r, size, info)
		return info, nil
	case bytes.Contains([]byte(brands), []byte("avif")), bytes.Contains([]byte(brands), []byte("avis")):
		info.Format = FormatAVIF
	case containsAny(brands, "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1"):
		info.Format = FormatHEIF
	default:
		return info, nil
	}

	for _, b := range children(r, 0, size, size) {
		if b.kind == "meta" {
			probeHEIFMeta(r, size, b, info)
			break
		}
	}

	return info, nil
}

func containsAny(s string, brands ...string) bool {
	for _, brand := range brands {
		if bytes.Contains([]byte(s), []byte(brand)) {
			return true
		}
	}
	return false
}

func probeHEIFMeta(r io.ReaderAt, size int64, meta box, info *Info) {
	h := &heif{items: map[uint64]item{}, locations: map[uint64]location{}, assoc: map[uint64][]int{}}

	for _, b := range children(r, meta.content+4, meta.end, size) {
		if b.end-b.content > maxMetadata {
			continue
		}
		data, err := readAt(r, b.content, b.end-b.content, size)
		if err != nil {
			continue
		}
		rd := &reader{data: data}

		switch b.kind {
		case "pitm":
			if rd.u8() == 0 {
				rd.take(3)
				h.primary = rd.u16()
			} else {
				rd.take(3)
				h.primary = rd.u32()
			}
		case "iinf":
			version := rd.u8()
			rd.take(3)
			if version == 0 {
				rd.u16()
			} else {
				rd.u32()
			}
			for _, infe := range children(r, b.content+int64(rd.pos), b.end, size) {
				parseInfe(r, size, infe, h)
			}
		case "iloc":
			parseIloc(rd, h)
		case "iprp":
			for _, c := range children(r, b.content, b.end, size) {
				switch c.kind {
				case "ipco":
					h.properties = children(r, c.content, c.end, size)
				case "ipma":
					parseIpma(r, size, c, h)
				}
			}
		case "idat":
			h.idat = b
		}
	}

	for _, p := range h.assoc[h.primary] {
		if p <= 0 || p > len(h.properties) {
			continue
		}
		prop := h.properties[p-1]
		data, err := readAt(r, prop.content, min(prop.end-prop.content, 64), size)
		if err != nil {
			continue
		}
		switch prop.kind {
		case "ispe":
			if len(data) >= 12 {
				info.Width = int(binary.BigEndian.Uint32(data[4:8]))
				info.Height = int(binary.BigEndian.Uint32(data[8:12]))
			}
		case "irot":
			if len(data) >= 1 {
				info.Rotation = int(data[0]&3) * 90
				info.HasRotation = true
			}
		case "imir":
			if len(data) >= 1 {
				info.Mirror = int(data[0]&1) + 1
			}
		}
	}

	for id, it := range h.items {
		switch {
		case it.kind == "Exif" && info.EXIF == nil:
			if data := h.read(r, size, id); len(data) >= 4 {
				skip := int(binary.BigEndian.Uint32(data[0:4]))
				if 4+skip <= len(data) {
					info.EXIF = bytes.TrimPrefix(data[4+skip:], exifPrefix)
				}
			}
		case it.kind == "mime" && it.contentType == "application/rdf+xml" && info.XMP == nil:
			info.XMP = h.read(r, size, id)
		}
	}
}

func parseInfe(r io.ReaderAt, size int64, b box, h *heif) {
	if b.kind != "infe" || b.end-b.content > 4096 {
		return
	}
	data, err := readAt(r, b.content, b.end-b.content, size)
	if err != nil {
		return
	}

	rd := &reader{data: data}
	version := rd.u8()
	rd.take(3)
	if version < 2 {
		return
	}

	var id uint64
	if version == 2 {
		id = rd.u16()
	} else {
		id = rd.u32()
	}
	rd.u16()
	it := item{kind: string(rd.take(4))}
	rd.cstring()
	if it.kind == "mime" {
		it.contentType = rd.cstring()
	}
	if !rd.bad {
		h.items[id] = it
	}
}

func parseIloc(rd *reader, h *heif) {
	version := rd.u8()
	rd.take(3)
	sizes := rd.u8()
	offsetSize, lengthSize := int(sizes>>4), int(sizes&15)
	more := rd.u8()
	baseSize, indexSize := int(more>>4), 0
	if version == 1 || version == 2 {
		indexSize = int(more & 15)
	}

	var count uint64
	if version < 2 {
		count = rd.u16()
	} else {
		count = rd.u32()
	}

	for i := uint64(0); i < count && !rd.bad; i++ {
		var id uint64
		if version < 2 {
			id = rd.u16()
		} else {
			id = rd.u32()
		}
		loc := location{}
		if version == 1 || version == 2 {
			loc.method = rd.u16() & 15
		}
		rd.u16()
		loc.base = rd.sized(baseSize)
		extents := rd.u16()
		for e := uint64(0); e < extents && !rd.bad; e++ {
			if indexSize > 0 {
				rd.sized(indexSize)
			}
			loc.extents = append(loc.extents, extent{offset: rd.sized(offsetSize), length: rd.sized(lengthSize)})
		}
		if !rd.bad {
			h.locations[id] = loc
		}
	}
}

func parseIpma(r io.ReaderAt, size int64, b box, h *heif) {
	if b.end-b.content > maxMetadata {
		return
	}
	data, err := readAt(r, b.content, b.end-b.content, size)
	if err != nil {
		return
	}

	rd := &reader{data: data}
	version := rd.u8()
	flags := rd.take(3)
	wide := flags[2]&1 == 1
	count := rd.u32()

	for i := uint64(0); i < count && !rd.bad; i++ {
		var id uint64
		if version < 1 {
			id = rd.u16()
		} else {
			id = rd.u32()
		}
		n := rd.u8()
		for a := uint64(0); a < n && !rd.bad; a++ {
			if wide {
				h.assoc[id] = append(h.assoc[id], int(rd.u16()&0x7fff))
			} else {
				h.assoc[id] = append(h.assoc[id], int(rd.u8()&0x7f))
			}
		}
	}
}

func (h *heif) read(r io.ReaderAt, size int64, id uint64) []byte {
	loc, ok := h.locations[id]
	if !ok {
		return nil
	}

	var out []byte
	for _, e := range loc.extents {
		if e.length > maxMetadata || uint64(len(out))+e.length > maxMetadata {
			return nil
		}
		off := int64(loc.base + e.offset)
		if loc.method == 1 {
			off += h.idat.content
		} else if loc.method != 0 {
			return nil
		}
		length := int64(e.length)
		if length == 0 {
			length = size - off
		}
		data, err := readAt(r, off, length, size)
		if err != nil {
			return nil
		}
		out = append(out, data...)
	}

	return out
}

func probeCR3(r io.ReaderAt, size int64, info *Info) {
	for _, top := range children(r, 0, size, size) {
		if top.kind != "moov" {
			continue
		}
		for _, b := range children(r, top.content, top.end, size) {
			if b.kind != "uuid" {
				continue
			}
			id, err := readAt(r, b.content, 16, size)
			if err != nil || !bytes.Equal(id, canonUUID) {
				continue
			}
			for _, c := range children(r, b.content+16, b.end, size) {
				if c.end-c.content > maxMetadata {
					continue
				}
				data, err := readAt(r, c.content, c.end-c.content, size)
				if err != nil {
					continue
				}
				switch c.kind {
				case "CMT1":
					info.EXIF = data
				case "CMT2":
					info.CR3Exif = data
				case "CMT4":
					info.CR3GPS = data
				}
			}
		}
	}
}
