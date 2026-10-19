package testmedia

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
)

const (
	typeByte      uint16 = 1
	typeASCII     uint16 = 2
	typeShort     uint16 = 3
	typeLong      uint16 = 4
	typeRational  uint16 = 5
	typeSRational uint16 = 10
)

type entry struct {
	tag   uint16
	typ   uint16
	count uint32
	data  []byte
}

type ifd []entry

func ascii(tag uint16, s string) entry {
	b := append([]byte(s), 0)
	return entry{tag: tag, typ: typeASCII, count: uint32(len(b)), data: b}
}

func short(order binary.ByteOrder, tag uint16, v ...uint16) entry {
	b := make([]byte, 2*len(v))
	for i, x := range v {
		order.PutUint16(b[2*i:], x)
	}
	return entry{tag: tag, typ: typeShort, count: uint32(len(v)), data: b}
}

func long(order binary.ByteOrder, tag uint16, v ...uint32) entry {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		order.PutUint32(b[4*i:], x)
	}
	return entry{tag: tag, typ: typeLong, count: uint32(len(v)), data: b}
}

func rational(order binary.ByteOrder, tag uint16, v ...[2]uint32) entry {
	b := make([]byte, 8*len(v))
	for i, x := range v {
		order.PutUint32(b[8*i:], x[0])
		order.PutUint32(b[8*i+4:], x[1])
	}
	return entry{tag: tag, typ: typeRational, count: uint32(len(v)), data: b}
}

func srational(order binary.ByteOrder, tag uint16, num int32, den int32) entry {
	b := make([]byte, 8)
	order.PutUint32(b, uint32(num))
	order.PutUint32(b[4:], uint32(den))
	return entry{tag: tag, typ: typeSRational, count: 1, data: b}
}

func degrees(v float64) [][2]uint32 {
	v = math.Abs(v)
	d := math.Floor(v)
	m := math.Floor((v - d) * 60)
	s := ((v-d)*60 - m) * 60
	return [][2]uint32{{uint32(d), 1}, {uint32(m), 1}, {uint32(math.Round(s * 10000)), 10000}}
}

type layout struct {
	order binary.ByteOrder
	buf   bytes.Buffer
}

func (l *layout) size(d ifd) int {
	n := 2 + 12*len(d) + 4
	for _, e := range d {
		if len(e.data) > 4 {
			n += len(e.data) + len(e.data)%2
		}
	}
	return n
}

func (l *layout) write(d ifd, at int, next uint32) {
	sort.Slice(d, func(i, j int) bool { return d[i].tag < d[j].tag })

	data := at + 2 + 12*len(d) + 4
	var head, tail bytes.Buffer
	b2 := make([]byte, 2)
	b4 := make([]byte, 4)

	l.order.PutUint16(b2, uint16(len(d)))
	head.Write(b2)
	for _, e := range d {
		l.order.PutUint16(b2, e.tag)
		head.Write(b2)
		l.order.PutUint16(b2, e.typ)
		head.Write(b2)
		l.order.PutUint32(b4, e.count)
		head.Write(b4)
		if len(e.data) <= 4 {
			value := make([]byte, 4)
			copy(value, e.data)
			head.Write(value)
			continue
		}
		l.order.PutUint32(b4, uint32(data+tail.Len()))
		head.Write(b4)
		tail.Write(e.data)
		if len(e.data)%2 == 1 {
			tail.WriteByte(0)
		}
	}
	l.order.PutUint32(b4, next)
	head.Write(b4)

	l.buf.Write(head.Bytes())
	l.buf.Write(tail.Bytes())
}
