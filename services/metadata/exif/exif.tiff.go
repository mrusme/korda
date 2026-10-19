package exif

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
)

const (
	typeByte      uint16 = 1
	typeASCII     uint16 = 2
	typeShort     uint16 = 3
	typeLong      uint16 = 4
	typeRational  uint16 = 5
	typeSByte     uint16 = 6
	typeUndefined uint16 = 7
	typeSShort    uint16 = 8
	typeSLong     uint16 = 9
	typeSRational uint16 = 10
	typeFloat     uint16 = 11
	typeDouble    uint16 = 12
	typeIFD       uint16 = 13

	maxEntries = 1024
	maxIFDs    = 64
)

var (
	ErrNotTIFF = errors.New("not a TIFF block")

	typeSizes = map[uint16]uint32{
		typeByte: 1, typeASCII: 1, typeShort: 2, typeLong: 4, typeRational: 8,
		typeSByte: 1, typeUndefined: 1, typeSShort: 2, typeSLong: 4, typeSRational: 8,
		typeFloat: 4, typeDouble: 8, typeIFD: 4,
	}
)

type Entry struct {
	Tag   uint16
	Type  uint16
	Count uint32
	Data  []byte
	order binary.ByteOrder
}

type IFD map[uint16]Entry

type TIFF struct {
	Order   binary.ByteOrder
	IFD0    IFD
	IFD1    IFD
	Exif    IFD
	GPS     IFD
	SubIFDs []IFD
	Data    []byte
}

func Parse(data []byte) (*TIFF, error) {
	if len(data) < 8 {
		return nil, ErrNotTIFF
	}

	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return nil, ErrNotTIFF
	}
	if magic := order.Uint16(data[2:4]); magic != 42 && magic != 0x4f52 && magic != 0x5352 && magic != 0x55 {
		return nil, ErrNotTIFF
	}

	t := &TIFF{Order: order, Data: data}
	seen := map[uint32]bool{}

	first := order.Uint32(data[4:8])
	ifd0, next := t.readIFD(first, seen)
	if ifd0 == nil {
		return nil, ErrNotTIFF
	}
	t.IFD0 = ifd0
	if next != 0 {
		t.IFD1, _ = t.readIFD(next, seen)
	}

	if e, ok := ifd0[0x8769]; ok {
		t.Exif, _ = t.readIFD(e.uint32(0), seen)
	}
	if e, ok := ifd0[0x8825]; ok {
		t.GPS, _ = t.readIFD(e.uint32(0), seen)
	}
	if e, ok := ifd0[0x014a]; ok {
		for i := range min(int(e.Count), 8) {
			if sub, _ := t.readIFD(e.uint32(i), seen); sub != nil {
				t.SubIFDs = append(t.SubIFDs, sub)
			}
		}
	}

	return t, nil
}

func (t *TIFF) readIFD(offset uint32, seen map[uint32]bool) (IFD, uint32) {
	data := t.Data
	if offset < 8 || uint64(offset)+2 > uint64(len(data)) || seen[offset] || len(seen) >= maxIFDs {
		return nil, 0
	}
	seen[offset] = true

	count := int(t.Order.Uint16(data[offset:]))
	if count == 0 || count > maxEntries {
		return nil, 0
	}

	end := uint64(offset) + 2 + uint64(count)*12
	if end > uint64(len(data)) {
		count = (len(data) - int(offset) - 2) / 12
		end = uint64(offset) + 2 + uint64(count)*12
	}

	ifd := make(IFD, count)
	for i := range count {
		at := int(offset) + 2 + i*12
		raw := data[at : at+12]
		e := Entry{
			Tag:   t.Order.Uint16(raw[0:2]),
			Type:  t.Order.Uint16(raw[2:4]),
			Count: t.Order.Uint32(raw[4:8]),
			order: t.Order,
		}
		size, ok := typeSizes[e.Type]
		if !ok {
			continue
		}
		total := uint64(size) * uint64(e.Count)
		if total <= 4 {
			e.Data = raw[8 : 8+total]
		} else {
			valueAt := uint64(t.Order.Uint32(raw[8:12]))
			if valueAt+total > uint64(len(data)) {
				continue
			}
			e.Data = data[valueAt : valueAt+total]
		}
		if _, dup := ifd[e.Tag]; !dup {
			ifd[e.Tag] = e
		}
	}

	var next uint32
	if end+4 <= uint64(len(data)) {
		next = t.Order.Uint32(data[end:])
	}

	return ifd, next
}

func (e Entry) uint32(i int) uint32 {
	switch e.Type {
	case typeByte, typeUndefined:
		if i < len(e.Data) {
			return uint32(e.Data[i])
		}
	case typeShort:
		if 2*i+2 <= len(e.Data) {
			return uint32(e.order.Uint16(e.Data[2*i:]))
		}
	case typeLong, typeIFD:
		if 4*i+4 <= len(e.Data) {
			return e.order.Uint32(e.Data[4*i:])
		}
	}

	return 0
}

func (e Entry) Uint(i int) (uint64, bool) {
	switch e.Type {
	case typeByte, typeUndefined, typeShort, typeLong, typeIFD:
		if uint32(i) >= e.Count {
			return 0, false
		}
		return uint64(e.uint32(i)), true
	case typeSShort, typeSLong, typeSByte:
		v, ok := e.Int(i)
		if !ok || v < 0 {
			return 0, false
		}
		return uint64(v), true
	}

	return 0, false
}

func (e Entry) Int(i int) (int64, bool) {
	switch e.Type {
	case typeSByte:
		if i < len(e.Data) {
			return int64(int8(e.Data[i])), true
		}
	case typeSShort:
		if 2*i+2 <= len(e.Data) {
			return int64(int16(e.order.Uint16(e.Data[2*i:]))), true
		}
	case typeSLong:
		if 4*i+4 <= len(e.Data) {
			return int64(int32(e.order.Uint32(e.Data[4*i:]))), true
		}
	default:
		v, ok := e.Uint(i)
		return int64(v), ok
	}

	return 0, false
}

func (e Entry) Rational(i int) (num int64, den int64, ok bool) {
	if 8*i+8 > len(e.Data) {
		return 0, 0, false
	}

	switch e.Type {
	case typeRational:
		return int64(e.order.Uint32(e.Data[8*i:])), int64(e.order.Uint32(e.Data[8*i+4:])), true
	case typeSRational:
		return int64(int32(e.order.Uint32(e.Data[8*i:]))), int64(int32(e.order.Uint32(e.Data[8*i+4:]))), true
	}

	return 0, 0, false
}

func (e Entry) Float(i int) (float64, bool) {
	switch e.Type {
	case typeRational, typeSRational:
		num, den, ok := e.Rational(i)
		if !ok || den == 0 {
			return 0, false
		}
		return float64(num) / float64(den), true
	case typeFloat:
		if 4*i+4 <= len(e.Data) {
			return float64(math.Float32frombits(e.order.Uint32(e.Data[4*i:]))), true
		}
	case typeDouble:
		if 8*i+8 <= len(e.Data) {
			return math.Float64frombits(e.order.Uint64(e.Data[8*i:])), true
		}
	default:
		if v, ok := e.Int(i); ok {
			return float64(v), true
		}
	}

	return 0, false
}

func (e Entry) String() string {
	if e.Type != typeASCII && e.Type != typeUndefined && e.Type != typeByte {
		return ""
	}

	s := string(e.Data)
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}

	return strings.TrimSpace(s)
}
