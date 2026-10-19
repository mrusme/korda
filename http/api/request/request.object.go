package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type Object struct {
	Keys   []string
	Values map[string]any
}

func NewObject() *Object {
	return &Object{Values: make(map[string]any)}
}

func (o *Object) Set(key string, v any) {
	if _, ok := o.Values[key]; !ok {
		o.Keys = append(o.Keys, key)
	}
	o.Values[key] = v
}

func (o *Object) Get(key string) (any, bool) {
	v, ok := o.Values[key]
	return v, ok
}

func (o *Object) Has(key string) bool {
	_, ok := o.Values[key]
	return ok
}

func (o *Object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.Keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		encoded, err := marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(encoded)
		buf.WriteByte(':')
		if encoded, err = marshal(o.Values[key]); err != nil {
			return nil, err
		}
		buf.Write(encoded)
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}

func ordered(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for key := range t {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		o := NewObject()
		for _, key := range keys {
			o.Set(key, ordered(t[key]))
		}
		return o
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = ordered(item)
		}
		return out
	}

	return v
}

func decodeOrdered(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err = dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errTrailingData
	}

	return v, nil
}

var errTrailingData = errors.New("trailing data after JSON value")

func decodeValue(dec *json.Decoder) (any, error) {
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
		o := NewObject()
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, _ := keyTok.(string)
			value, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			o.Set(key, value)
		}
		if _, err = dec.Token(); err != nil {
			return nil, err
		}
		return o, nil

	case '[':
		items := []any{}
		for dec.More() {
			value, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		if _, err = dec.Token(); err != nil {
			return nil, err
		}
		return items, nil
	}

	return nil, errTrailingData
}

func isArrayIndex(key string) bool {
	if key == "" || (len(key) > 1 && key[0] == '0') {
		return false
	}
	n, err := strconv.ParseUint(key, 10, 32)
	return err == nil && n < math.MaxUint32
}

func jsKeyOrder(keys []string) []string {
	var indexes, names []string
	for _, key := range keys {
		if isArrayIndex(key) {
			indexes = append(indexes, key)
		} else {
			names = append(names, key)
		}
	}
	slices.SortFunc(indexes, func(a, b string) int {
		x, _ := strconv.ParseUint(a, 10, 32)
		y, _ := strconv.ParseUint(b, 10, 32)
		return int(int64(x) - int64(y))
	})

	return append(indexes, names...)
}

func isJSSpace(r rune) bool {
	switch r {
	case 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x20, 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}

	return r >= 0x2000 && r <= 0x200a
}

func jsTrim(s string) string {
	return strings.TrimFunc(s, isJSSpace)
}

var (
	jsDecimal = regexp.MustCompile(`^[+-]?(?:(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?|Infinity)$`)
	jsRadix   = regexp.MustCompile(`^0([xXoObB])([0-9a-fA-F]+)$`)
)

func jsStringToNumber(s string) float64 {
	s = jsTrim(s)
	if s == "" {
		return 0
	}

	if m := jsRadix.FindStringSubmatch(s); m != nil {
		base := map[byte]int{'x': 16, 'X': 16, 'o': 8, 'O': 8, 'b': 2, 'B': 2}[m[1][0]]
		value := 0.0
		for _, digit := range strings.ToLower(m[2]) {
			d := strings.IndexRune("0123456789abcdef", digit)
			if d < 0 || d >= base {
				return math.NaN()
			}
			value = value*float64(base) + float64(d)
		}
		return value
	}

	if !jsDecimal.MatchString(s) {
		return math.NaN()
	}
	switch s {
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return math.NaN()
	}

	return f
}

func jsToString(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", true
	case string:
		return t, true
	case json.Number:
		return t.String(), true
	case bool:
		return strconv.FormatBool(t), true
	case []any:
		parts := make([]string, len(t))
		for i, item := range t {
			s, ok := jsToString(item)
			if !ok {
				return "", false
			}
			parts[i] = s
		}
		return strings.Join(parts, ","), true
	}

	return "", false
}

func jsNumber(v any, present bool) float64 {
	if !present {
		return math.NaN()
	}

	switch t := v.(type) {
	case nil:
		return 0
	case bool:
		if t {
			return 1
		}
		return 0
	case json.Number:
		return jsStringToNumber(t.String())
	case string:
		return jsStringToNumber(t)
	case []any:
		s, ok := jsToString(t)
		if !ok {
			return math.NaN()
		}
		return jsStringToNumber(s)
	}

	return math.NaN()
}
