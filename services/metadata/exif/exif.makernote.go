package exif

import (
	"bytes"
	"encoding/binary"
)

var appleHeader = []byte("Apple iOS\x00")

func (e Entry) Bytes() []byte {
	return e.Data
}

func AppleMakerNote(data []byte) IFD {
	if !bytes.HasPrefix(data, appleHeader) || len(data) < 16 || string(data[12:14]) != "MM" {
		return nil
	}

	t := &TIFF{Order: binary.BigEndian, Data: data}
	ifd, _ := t.readIFD(14, map[uint32]bool{})

	return ifd
}
