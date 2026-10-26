package probe

import (
	"encoding/binary"
	"io"
)

const dataUTF8 = 1

func child(r io.ReaderAt, parent box, kind string, size int64) (box, bool) {
	for _, b := range children(r, parent.content, parent.end, size) {
		if b.kind == kind {
			return b, true
		}
	}

	return box{}, false
}

func handlerType(r io.ReaderAt, trak box, size int64) string {
	mdia, ok := child(r, trak, "mdia", size)
	if !ok {
		return ""
	}
	hdlr, ok := child(r, mdia, "hdlr", size)
	if !ok {
		return ""
	}
	data, err := readAt(r, hdlr.content+8, 4, size)
	if err != nil {
		return ""
	}

	return string(data)
}

func VideoKeys(r io.ReaderAt, size int64) map[string]string {
	out := map[string]string{}
	moov, ok := box{}, false
	for _, b := range children(r, 0, size, size) {
		if b.kind == "moov" {
			moov, ok = b, true
			break
		}
	}
	if !ok {
		return out
	}

	for _, trak := range children(r, moov.content, moov.end, size) {
		if trak.kind != "trak" || handlerType(r, trak, size) != "vide" {
			continue
		}
		if meta, ok := child(r, trak, "meta", size); ok {
			readKeys(r, meta, size, out)
		}
	}

	return out
}

func readKeys(r io.ReaderAt, meta box, size int64, out map[string]string) {
	from := meta.content
	if head, err := readAt(r, from, 8, size); err == nil && len(head) == 8 {
		switch string(head[4:8]) {
		case "hdlr", "keys", "ilst":
		default:
			from += 4
		}
	}

	var keys, ilst box
	var hasKeys, hasIlst bool
	for _, b := range children(r, from, meta.end, size) {
		switch b.kind {
		case "keys":
			keys, hasKeys = b, true
		case "ilst":
			ilst, hasIlst = b, true
		}
	}
	if !hasKeys || !hasIlst || keys.end-keys.content > maxMetadata {
		return
	}

	data, err := readAt(r, keys.content, keys.end-keys.content, size)
	if err != nil {
		return
	}
	rd := &reader{data: data}
	rd.take(4)
	count := rd.u32()
	names := map[uint32]string{}
	for i := uint32(1); uint64(i) <= count && !rd.bad; i++ {
		n := int(rd.u32())
		rd.take(4)
		name := string(rd.take(n - 8))
		if rd.bad {
			break
		}
		names[i] = name
	}

	for _, entry := range children(r, ilst.content, ilst.end, size) {
		name, ok := names[binary.BigEndian.Uint32([]byte(entry.kind))]
		if !ok {
			continue
		}
		value, ok := child(r, entry, "data", size)
		if !ok || value.end-value.content < 8 || value.end-value.content > maxMetadata {
			continue
		}
		raw, err := readAt(r, value.content, value.end-value.content, size)
		if err != nil || binary.BigEndian.Uint32(raw[0:4]) != dataUTF8 {
			continue
		}
		if _, seen := out[name]; !seen {
			out[name] = string(raw[8:])
		}
	}
}
