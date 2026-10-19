package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/sanitize"
)

const maxSafeInteger int64 = 1<<53 - 1

type mode int

const (
	modeBody mode = iota
	modeQuery
)

type pattern struct {
	re     *regexp.Regexp
	source string
}

var (
	emailPattern = pattern{
		re:     regexp.MustCompile(`^[\p{L}\p{M}\p{N}.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\p{L}\p{N}](?:[\p{L}\p{M}\p{N}-]{0,61}[\p{L}\p{M}\p{N}])?(?:\.[\p{L}\p{N}](?:[\p{L}\p{M}\p{N}-]{0,61}[\p{L}\p{M}\p{N}])?)*$`),
		source: `/^[\p{L}\p{M}\p{N}.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\p{L}\p{N}](?:[\p{L}\p{M}\p{N}-]{0,61}[\p{L}\p{M}\p{N}])?(?:\.[\p{L}\p{N}](?:[\p{L}\p{M}\p{N}-]{0,61}[\p{L}\p{M}\p{N}])?)*$/u`,
	}
	uuidPattern = pattern{
		re:     regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`),
		source: `/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/`,
	}
	datePattern = pattern{
		re:     regexp.MustCompile(`^(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))$`),
		source: `/^(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))$/`,
	}
	namedPatterns = map[string]pattern{
		"pin": {
			re:     regexp.MustCompile(`^\d{6}$`),
			source: `/^\d{6}$/`,
		},
		"license": {
			re:     regexp.MustCompile(`^IM(SV|CL)(-[\dA-Za-z]{4}){8}$`),
			source: `/^IM(SV|CL)(-[\dA-Za-z]{4}){8}$/`,
		},
	}
)

var (
	ErrUnsupportedType = errors.New("unsupported type in request schema")
	ErrUnknownRule     = errors.New("unknown validation rule")

	optionalPkgPath = reflect.TypeFor[optional.Value[int]]().PkgPath()
	enumType        = reflect.TypeFor[enum]()
)

type enum interface {
	Enum() []string
}

type rules struct {
	email    bool
	uuid     bool
	date     bool
	nullable bool
	optional bool
	sanitize bool
	min      *int64
	regex    *pattern
}

func parseRules(tag string) (r rules, err error) {
	if tag == "" {
		return r, nil
	}

	for rule := range strings.SplitSeq(tag, ",") {
		name, arg, _ := strings.Cut(rule, "=")
		switch name {
		case "email":
			r.email = true
		case "uuid":
			r.uuid = true
		case "date":
			r.date = true
		case "nullable":
			r.nullable = true
		case "optional":
			r.optional = true
		case "sanitize":
			r.sanitize = true
		case "min":
			minimum, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
			}
			r.min = &minimum
		case "regex":
			p, ok := namedPatterns[arg]
			if !ok {
				return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
			}
			r.regex = &p
		default:
			return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
		}
	}

	return r, nil
}

type walker struct {
	mode   mode
	issues []Issue
	err    error
}

func (w *walker) report(path []any, iss Issue, message string) {
	w.issues = append(w.issues, iss.withPath(path, message))
}

func (w *walker) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

func join(path []any, segment any) []any {
	return append(append(make([]any, 0, len(path)+1), path...), segment)
}

func received(raw any, present bool) string {
	if !present {
		return "undefined"
	}

	switch raw.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	}

	return "object"
}

func typeOf(raw any, present bool) string {
	switch received(raw, present) {
	case "null", "array":
		return "object"
	case "undefined":
		return "undefined"
	}

	return received(raw, present)
}

func isOptional(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t.PkgPath() == optionalPkgPath && strings.HasPrefix(t.Name(), "Value[")
}

func (w *walker) value(path []any, raw any, present bool, t reflect.Type, r rules) (any, bool) {
	switch {
	case isOptional(t):
		inner, _ := t.FieldByName("V")
		if !present {
			return nil, false
		}
		if raw == nil && r.nullable {
			return nil, true
		}
		return w.value(path, raw, present, inner.Type, r)

	case t.Kind() == reflect.Pointer:
		if !present {
			return nil, false
		}
		if raw == nil && r.nullable {
			return nil, true
		}
		return w.value(path, raw, present, t.Elem(), r)

	case t.Implements(enumType):
		return w.enum(path, raw, present, reflect.Zero(t).Interface().(enum).Enum())

	case t.Kind() == reflect.Slice:
		if !present && r.optional {
			return nil, false
		}
		return w.slice(path, raw, present, t, r)

	case t.Kind() == reflect.Struct:
		return w.object(path, raw, present, t)

	case t.Kind() == reflect.String:
		return w.string(path, raw, present, r)

	case t.Kind() == reflect.Bool:
		if w.mode == modeQuery {
			return w.stringBool(path, raw, present)
		}
		if v, ok := raw.(bool); ok && present {
			return v, true
		}
		iss, message := invalidType("boolean", received(raw, present))
		w.report(path, iss, message)
		return nil, false

	case t.Kind() >= reflect.Int && t.Kind() <= reflect.Int64:
		return w.integer(path, raw, present, r)

	case t.Kind() == reflect.Float64:
		return w.number(path, raw, present, r)
	}

	w.fail(fmt.Errorf("%w: %s", ErrUnsupportedType, t))
	return nil, false
}

func (w *walker) object(path []any, raw any, present bool, t reflect.Type) (any, bool) {
	m, ok := raw.(map[string]any)
	if !ok || !present {
		iss, message := invalidType("object", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	out := make(map[string]any, len(m))
	w.fields(path, m, t, out)

	return out, true
}

func (w *walker) fields(path []any, m map[string]any, t reflect.Type, out map[string]any) {
	for f := range t.Fields() {
		if !f.IsExported() {
			continue
		}

		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
			w.fields(path, m, f.Type, out)
			continue
		}
		if name == "" {
			name = f.Name
		}

		r, err := parseRules(f.Tag.Get("v"))
		if err != nil {
			w.fail(err)
			continue
		}

		child, present := m[name]
		if v, keep := w.value(join(path, name), child, present, f.Type, r); keep {
			out[name] = v
		}
	}
}

func (w *walker) slice(path []any, raw any, present bool, t reflect.Type, r rules) (any, bool) {
	items, ok := raw.([]any)
	if !ok || !present {
		iss, message := invalidType("array", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	itemRules := rules{}
	out := make([]any, 0, len(items))
	for i, item := range items {
		if v, keep := w.value(join(path, i), item, true, t.Elem(), itemRules); keep {
			out = append(out, v)
		}
	}

	if r.min != nil && int64(len(items)) < *r.min {
		iss, message := tooSmall("array", *r.min)
		w.report(path, iss, message)
	}

	return out, true
}

func (w *walker) enum(path []any, raw any, present bool, values []string) (any, bool) {
	if s, ok := raw.(string); ok && present {
		if slices.Contains(values, s) {
			return s, true
		}
	}

	iss, message := invalidValue(values)
	w.report(path, iss, message)
	return nil, false
}

func (w *walker) stringBool(path []any, raw any, present bool) (any, bool) {
	s, ok := raw.(string)
	if !ok || !present {
		iss, message := invalidType("string", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	switch s {
	case "true":
		return true, true
	case "false":
		return false, true
	}

	iss, message := invalidStringBool()
	w.report(path, iss, message)
	return nil, false
}

func (w *walker) string(path []any, raw any, present bool, r rules) (any, bool) {
	s, ok := raw.(string)
	if !ok || !present {
		iss, message := invalidType("string", received(raw, present))
		switch {
		case r.email:
			message = "Invalid input: expected email, received " + typeOf(raw, present)
		case r.date:
			message = "Invalid input: expected ISO date string (YYYY-MM-DD), received " + typeOf(raw, present)
		}
		w.report(path, iss, message)
		return nil, false
	}

	valid := true
	if r.email && !emailPattern.re.MatchString(s) {
		iss, message := invalidFormat("email", emailPattern.source, "Invalid input: expected email, received string")
		w.report(path, iss, message)
		valid = false
	}
	if r.uuid && !uuidPattern.re.MatchString(s) {
		iss, message := invalidFormat("uuid", uuidPattern.source, "Invalid UUID")
		w.report(path, iss, message)
		valid = false
	}
	if r.date && !datePattern.re.MatchString(s) {
		iss, message := invalidFormat("date", datePattern.source,
			"Invalid input: expected ISO date string (YYYY-MM-DD), received string")
		w.report(path, iss, message)
		valid = false
	}
	if r.regex != nil && !r.regex.re.MatchString(s) {
		iss, message := invalidFormat("regex", r.regex.source, "Invalid string: must match pattern "+r.regex.source)
		w.report(path, iss, message)
		valid = false
	}
	if r.min != nil && int64(len(utf16.Encode([]rune(s)))) < *r.min {
		iss, message := tooSmall("string", *r.min)
		w.report(path, iss, message)
		valid = false
	}
	if !valid {
		return nil, false
	}

	if r.email {
		s = strings.ToLower(s)
	}
	if r.sanitize {
		s = sanitize.Filename(s)
	}

	return s, true
}

func (w *walker) finite(path []any, raw any, present bool) (float64, bool) {
	n, ok := raw.(json.Number)
	if !ok || !present {
		iss, message := invalidType("number", received(raw, present))
		w.report(path, iss, message)
		return 0, false
	}

	f, err := strconv.ParseFloat(string(n), 64)
	if math.IsInf(f, 0) {
		iss, message := invalidInfinity()
		w.report(path, iss, message)
		return 0, false
	}
	if err != nil {
		iss, message := invalidType("number", "string")
		w.report(path, iss, message)
		return 0, false
	}

	return f, true
}

func (w *walker) number(path []any, raw any, present bool, r rules) (any, bool) {
	f, ok := w.finite(path, raw, present)
	if !ok {
		return nil, false
	}

	if r.min != nil && f < float64(*r.min) {
		iss, message := tooSmall("number", *r.min)
		w.report(path, iss, message)
		return nil, false
	}

	return f, true
}

func (w *walker) integer(path []any, raw any, present bool, r rules) (any, bool) {
	f, ok := w.finite(path, raw, present)
	if !ok {
		return nil, false
	}

	if f != math.Trunc(f) {
		iss, message := invalidInt()
		w.report(path, iss, message)
		return nil, false
	}

	valid := true
	if math.Abs(f) > float64(maxSafeInteger) {
		iss, message := unsafeInt(f > 0)
		w.report(path, iss, message)
		valid = false
	}
	if r.min != nil && f < float64(*r.min) {
		iss, message := tooSmall("number", *r.min)
		w.report(path, iss, message)
		valid = false
	}
	if !valid {
		return nil, false
	}

	return int64(f), true
}

func validate(raw any, present bool, t reflect.Type, m mode) (any, []Issue, error) {
	w := &walker{mode: m}

	if t.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedType, t)
	}

	v, _ := w.object([]any{}, raw, present, t)
	if w.err != nil {
		return nil, nil, w.err
	}

	return v, w.issues, nil
}
