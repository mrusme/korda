package jsonb

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrNotJSON = errors.New("not a JSON value")

type object struct {
	keys   []string
	values map[string]any
}

func Normalize(raw []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	v, err := decode(dec)
	if err != nil {
		return nil, err
	}
	if _, err = dec.Token(); !errors.Is(err, io.EOF) {
		return nil, ErrNotJSON
	}

	var buf bytes.Buffer
	encode(&buf, v)

	return buf.Bytes(), nil
}

func decode(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}

	switch delim {
	case '{':
		o := &object{values: make(map[string]any)}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, _ := keyTok.(string)
			value, err := decode(dec)
			if err != nil {
				return nil, err
			}
			if _, exists := o.values[key]; !exists {
				o.keys = append(o.keys, key)
			}
			o.values[key] = value
		}
		_, err = dec.Token()
		return o, err

	case '[':
		items := []any{}
		for dec.More() {
			value, err := decode(dec)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		_, err = dec.Token()
		return items, err
	}

	return nil, ErrNotJSON
}

func isIndex(key string) bool {
	if key == "" || (len(key) > 1 && key[0] == '0') {
		return false
	}
	n, err := strconv.ParseUint(key, 10, 32)
	return err == nil && n < math.MaxUint32
}

func order(keys []string) []string {
	out := slices.Clone(keys)
	slices.SortFunc(out, func(a, b string) int {
		ai, bi := isIndex(a), isIndex(b)
		switch {
		case ai && bi:
			x, _ := strconv.ParseUint(a, 10, 32)
			y, _ := strconv.ParseUint(b, 10, 32)
			return int(int64(x) - int64(y))
		case ai:
			return -1
		case bi:
			return 1
		case len(a) != len(b):
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})

	return out
}

func encode(buf *bytes.Buffer, v any) {
	switch t := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		buf.WriteString(strconv.FormatBool(t))
	case json.Number:
		buf.WriteString(Number(t.String()))
	case string:
		writeString(buf, t)
	case []any:
		buf.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			encode(buf, item)
		}
		buf.WriteByte(']')
	case *object:
		buf.WriteByte('{')
		for i, key := range order(t.keys) {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeString(buf, key)
			buf.WriteByte(':')
			encode(buf, t.values[key])
		}
		buf.WriteByte('}')
	}
}

func Number(text string) string {
	f, err := strconv.ParseFloat(text, 64)
	if err != nil && !math.IsInf(f, 0) {
		return "null"
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "null"
	}
	if f == 0 {
		return "0"
	}

	sign := ""
	if f < 0 {
		sign = "-"
		f = -f
	}

	mantissa, exp, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	e, _ := strconv.Atoi(exp)
	k := len(digits)
	n := e + 1

	switch {
	case k <= n && n <= 21:
		return sign + digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return sign + digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return sign + "0." + strings.Repeat("0", -n) + digits
	}

	exponent := n - 1
	expSign := "+"
	if exponent < 0 {
		expSign = "-"
		exponent = -exponent
	}
	if k == 1 {
		return sign + digits + "e" + expSign + strconv.Itoa(exponent)
	}

	return sign + digits[:1] + "." + digits[1:] + "e" + expSign + strconv.Itoa(exponent)
}

func writeString(buf *bytes.Buffer, s string) {
	const hex = "0123456789abcdef"

	buf.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '"':
			buf.WriteString(`\"`)
		case r == '\\':
			buf.WriteString(`\\`)
		case r == '\b':
			buf.WriteString(`\b`)
		case r == '\f':
			buf.WriteString(`\f`)
		case r == '\n':
			buf.WriteString(`\n`)
		case r == '\r':
			buf.WriteString(`\r`)
		case r == '\t':
			buf.WriteString(`\t`)
		case r < 0x20:
			buf.WriteString(`\u00`)
			buf.WriteByte(hex[r>>4])
			buf.WriteByte(hex[r&0xf])
		default:
			buf.WriteString(s[i : i+size])
		}
		i += size
	}
	buf.WriteByte('"')
}
