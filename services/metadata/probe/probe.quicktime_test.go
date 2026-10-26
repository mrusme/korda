package probe

import (
	"bytes"
	"encoding/binary"
	"maps"
	"slices"
	"testing"
)

func atom(kind string, payload ...[]byte) []byte {
	body := bytes.Join(payload, nil)
	return append(append(binary.BigEndian.AppendUint32(nil, uint32(8+len(body))), kind...), body...)
}

func handlerAtom(kind string) []byte {
	return atom("hdlr", make([]byte, 8), []byte(kind), make([]byte, 12))
}

func keysAtom(names ...string) []byte {
	out := binary.BigEndian.AppendUint32(make([]byte, 4), uint32(len(names)))
	for _, name := range names {
		out = append(binary.BigEndian.AppendUint32(out, uint32(8+len(name))), "mdta"+name...)
	}

	return atom("keys", out)
}

func itemAtom(index uint32, kind uint32, value []byte) []byte {
	return atom(string(binary.BigEndian.AppendUint32(nil, index)),
		atom("data", binary.BigEndian.AppendUint32(nil, kind), make([]byte, 4), value))
}

func TestVideoKeys(t *testing.T) {
	lens := "com.apple.quicktime.camera.lens_model"
	keys := keysAtom(lens, "com.apple.quicktime.apple-maker-note.74")
	ilst := atom("ilst", itemAtom(1, 1, []byte("iPhone Air back camera 5.96mm f/1.6")), itemAtom(2, 21, []byte{0, 0, 0, 2}))
	sound := atom("trak", atom("mdia", handlerAtom("soun")),
		atom("meta", handlerAtom("mdta"), keys, atom("ilst", itemAtom(1, 1, []byte("sound lens")))))
	ftyp := atom("ftyp", []byte("qt  "), make([]byte, 4))

	cases := []struct {
		name string
		file []byte
		want map[string]string
	}{
		{
			name: "quicktime meta",
			file: slices.Concat(ftyp, atom("moov", sound, atom("trak", atom("mdia", handlerAtom("vide")), atom("meta", handlerAtom("mdta"), keys, ilst)))),
			want: map[string]string{lens: "iPhone Air back camera 5.96mm f/1.6"},
		},
		{
			name: "iso meta",
			file: slices.Concat(ftyp, atom("moov", atom("trak", atom("mdia", handlerAtom("vide")), atom("meta", make([]byte, 4), handlerAtom("mdta"), keys, ilst)))),
			want: map[string]string{lens: "iPhone Air back camera 5.96mm f/1.6"},
		},
		{
			name: "sound track only",
			file: slices.Concat(ftyp, atom("moov", sound)),
			want: map[string]string{},
		},
		{
			name: "truncated keys",
			file: slices.Concat(ftyp, atom("moov", atom("trak", atom("mdia", handlerAtom("vide")), atom("meta", handlerAtom("mdta"), atom("keys", make([]byte, 4), []byte{0, 0, 0, 9}), ilst)))),
			want: map[string]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := VideoKeys(bytes.NewReader(c.file), int64(len(c.file))); !maps.Equal(got, c.want) {
				t.Errorf("VideoKeys = %v, want %v", got, c.want)
			}
		})
	}
}
