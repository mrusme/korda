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
	"time"
	"unicode/utf16"

	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/helpers/sanitize"
	"tty.fail/mrus/korda/models/geo"
)

const maxSafeInteger int64 = 1<<53 - 1

type mode int

const (
	modeBody mode = iota
	modeQuery
)

type pattern struct {
	re      *regexp.Regexp
	source  string
	message string
}

func newPattern(expr string, flags string) pattern {
	return pattern{re: regexp.MustCompile(expr), source: "/" + expr + "/" + flags}
}

var (
	emailPattern    = newPattern(`^[\p{L}\p{M}\p{N}.!#$%&'*+/=?^_`+"`"+`{|}~-]+@[\p{L}\p{N}](?:[\p{L}\p{M}\p{N}-]{0,61}[\p{L}\p{M}\p{N}])?(?:\.[\p{L}\p{N}](?:[\p{L}\p{M}\p{N}-]{0,61}[\p{L}\p{M}\p{N}])?)*$`, "u")
	uuidPattern     = newPattern(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`, "")
	uuidV7Pattern   = newPattern(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`, "")
	anyUUIDPattern  = newPattern(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$`, "")
	datePattern     = newPattern(`^(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))$`, "")
	datetimePattern = newPattern(`^(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))T(?:(?:[01]\d|2[0-3]):[0-5]\d(?::[0-5]\d(?:\.\d+)?)?(?:Z|([+-](?:[01]\d|2[0-3]):[0-5]\d)))$`, "")
	namedPatterns   = map[string]pattern{
		"pin":     newPattern(`^\d{6}$`, ""),
		"license": newPattern(`^IM(SV|CL)(-[\dA-Za-z]{4}){8}$`, ""),
		"hlsSegment": {
			re:      regexp.MustCompile(`^(init\.mp4|seg_\d+\.m4s)$`),
			source:  `/^(init\.mp4|seg_\d+\.m4s)$/`,
			message: "Invalid HLS segment filename",
		},
		"filename": {
			re:      regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`),
			source:  `/^[a-zA-Z0-9_\-.]+$/`,
			message: "Filename contains invalid characters",
		},
		"hexColor": newPattern(`^#?([0-9A-Fa-f]{3}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$`, ""),
		"tagName": {
			re:      regexp.MustCompile(`^[^/]*$`),
			source:  `/^[^/]*$/`,
			message: `Tag name cannot contain slash characters ("/")`,
		},
	}
	datetimeParts = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d+))?)?(Z|[+-]\d{2}:\d{2})$`)

	fieldRefinements = map[string]func(raw any) string{
		"albumUserId": func(raw any) string {
			if s, _ := raw.(string); s == "me" || uuidPattern.re.MatchString(s) {
				return ""
			}
			return `Must be a UUID v4 or "me"`
		},
		"angle":          angle,
		"birthDate":      birthDate,
		"uniqueIds":      uniqueIDs,
		"cron":           cronExpression,
		"issuerUrl":      emptyOrURL("Issuer URL must be an empty string or a valid URL"),
		"accountUrl":     emptyOrURL("Account management URL must be an empty string or a valid URL"),
		"endSessionUrl":  emptyOrURL("endSessionEndpoint must be an empty string or a valid URL"),
		"externalDomain": emptyOrURL("External domain must be an empty string or a valid URL"),
		"nonEmptyItems": func(raw any) string {
			list, _ := raw.([]any)
			for _, item := range list {
				if s, ok := item.(string); ok && jsTrim(s) == "" {
					return "Array items must not be empty"
				}
			}
			return ""
		},
		"uniqueItems": func(raw any) string {
			if uniqueIDs(raw) != "" {
				return "Array must have unique items"
			}
			return ""
		},
		"rating": func(raw any) string {
			if n, ok := raw.(json.Number); ok {
				if f, err := strconv.ParseFloat(string(n), 64); err == nil && f == 0 {
					return "Rating must be -1 (rejected), 1–5 (starred), or null (unrated); 0 is not valid"
				}
			}
			return ""
		},
	}
)

var (
	ErrUnsupportedType = errors.New("unsupported type in request schema")
	ErrUnknownRule     = errors.New("unknown validation rule")

	optionalPkgPath = reflect.TypeFor[optional.Value[int]]().PkgPath()
	enumType        = reflect.TypeFor[enum]()
	timeType        = reflect.TypeFor[time.Time]()
	recordType      = reflect.TypeFor[json.RawMessage]()
	strictType      = reflect.TypeFor[refine.Strict]()
	refinerType     = reflect.TypeFor[refine.Refiner]()
	piperType       = reflect.TypeFor[refine.Piper]()
	bboxType        = reflect.TypeFor[geo.BBox]()
	numberEnumType  = reflect.TypeFor[numberEnum]()
)

type numberEnum interface {
	NumberEnum() []int64
}

type enum interface {
	Enum() []string
}

type rules struct {
	email    bool
	uuid     bool
	anyUUID  bool
	uuidV7   bool
	positive bool
	date     bool
	datetime bool
	nullable bool
	optional bool
	sanitize bool
	coerce   bool
	trim     bool
	json     bool
	boolean  bool
	config   bool
	split    bool
	time     bool
	url      bool
	required bool
	minF     *float64
	maxF     *float64
	min      *int64
	max      *int64
	regex    *pattern
	refine   []string
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
		case "anyuuid":
			r.anyUUID = true
		case "uuidv7":
			r.uuidV7 = true
		case "positive":
			r.positive = true
		case "date":
			r.date = true
		case "datetime":
			r.datetime = true
		case "nullable":
			r.nullable = true
		case "optional":
			r.optional = true
		case "sanitize":
			r.sanitize = true
		case "coerce":
			r.coerce = true
		case "trim":
			r.trim = true
		case "configbool":
			r.config = true
		case "split":
			r.split = true
		case "json":
			r.json = true
		case "boolean":
			r.boolean = true
		case "min", "max":
			if strings.Contains(arg, ".") {
				limit, err := strconv.ParseFloat(arg, 64)
				if err != nil {
					return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
				}
				if name == "min" {
					r.minF = &limit
				} else {
					r.maxF = &limit
				}
				continue
			}
			limit, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
			}
			if name == "min" {
				r.min = &limit
			} else {
				r.max = &limit
			}
		case "time":
			r.time = true
		case "url":
			r.url = true
		case "required":
			r.required = true
		case "regex":
			p, ok := namedPatterns[arg]
			if !ok {
				return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
			}
			r.regex = &p
		case "refine":
			for name := range strings.SplitSeq(arg, "|") {
				if _, ok := fieldRefinements[name]; !ok {
					return r, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
				}
				r.refine = append(r.refine, name)
			}
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

func (w *walker) add(prefix []any, issues ...refine.Issue) {
	for _, iss := range issues {
		path := append(append([]any{}, prefix...), iss.Path...)
		if iss.MessageFirst {
			w.issues = append(w.issues, newIssue("message", iss.Message, "code", "custom", "path", path))
			continue
		}
		w.issues = append(w.issues, Custom(path, iss.Message))
	}
}

func (w *walker) abortedSince(start int) bool {
	for _, iss := range w.issues[start:] {
		if iss.abort {
			return true
		}
	}

	return false
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

func (w *walker) value(path []any, raw any, present bool, t reflect.Type, r rules, items rules) (any, bool) {
	switch {
	case isOptional(t):
		inner, _ := t.FieldByName("V")
		if !present && r.required {
			return w.value(path, raw, present, inner.Type, r, items)
		}
		if !present {
			return nil, false
		}
		if raw == nil && r.nullable {
			return nil, true
		}
		return w.value(path, raw, present, inner.Type, r, items)

	case t.Kind() == reflect.Pointer:
		if !present {
			return nil, false
		}
		if raw == nil && r.nullable {
			return nil, true
		}
		return w.value(path, raw, present, t.Elem(), r, items)
	}

	if r.json && present {
		if s, ok := raw.(string); ok {
			parsed, err := decodeOrdered([]byte(s))
			if err != nil {
				w.issues = append(w.issues, invalidJSON(path))
				return nil, false
			}
			saved := w.mode
			w.mode = modeBody
			defer func() { w.mode = saved }()
			raw = parsed
		}
	}

	switch {
	case t == timeType && r.datetime:
		return w.datetime(path, raw, present)

	case t == recordType:
		return w.record(path, raw, present)

	case t.Implements(numberEnumType):
		return w.numberEnum(path, raw, present, reflect.Zero(t).Interface().(numberEnum).NumberEnum())

	case t.Implements(enumType):
		return w.enum(path, raw, present, reflect.Zero(t).Interface().(enum).Enum())

	case t == editsType:
		return w.edits(path, raw, present, r)

	case t.Kind() == reflect.Slice:
		if !present && r.optional {
			return nil, false
		}
		start := len(w.issues)
		v, keep := w.slice(path, raw, present, t, r, items)
		if keep && !w.abortedSince(start) {
			for _, name := range r.refine {
				if message := fieldRefinements[name](raw); message != "" {
					w.issues = append(w.issues, Custom(path, message))
				}
			}
		}
		return v, keep

	case t == bboxType:
		return w.bbox(path, raw, present)

	case t == editType:
		return w.edit(path, raw, present)

	case t.Kind() == reflect.Struct:
		return w.object(path, raw, present, t)

	case t.Kind() == reflect.String:
		start := len(w.issues)
		v, keep := w.string(path, raw, present, r)
		if !w.abortedSince(start) && len(w.issues) == start {
			for _, name := range r.refine {
				if message := fieldRefinements[name](raw); message != "" {
					if name == "cron" {
						w.issues = append(w.issues, newIssue("code", "custom", "message", message, "path", append([]any{}, path...)))
						return nil, false
					}
					w.issues = append(w.issues, Custom(path, message))
					return nil, false
				}
			}
		}
		return v, keep

	case t.Kind() == reflect.Bool:
		if s, ok := raw.(string); ok && present && r.config && (s == "true" || s == "false") {
			return s == "true", true
		}
		if w.mode == modeQuery && !r.boolean {
			return w.stringBool(path, raw, present)
		}
		if v, ok := raw.(bool); ok && present {
			return v, true
		}
		iss, message := invalidType("boolean", received(raw, present))
		w.report(path, iss, message)
		return nil, false

	case t.Kind() >= reflect.Int && t.Kind() <= reflect.Int64:
		start := len(w.issues)
		v, keep := w.integer(path, raw, present, r)
		if !w.abortedSince(start) {
			for _, name := range r.refine {
				if message := fieldRefinements[name](raw); message != "" {
					w.issues = append(w.issues, Custom(path, message))
					return nil, false
				}
			}
		}
		return v, keep

	case t.Kind() == reflect.Float64:
		return w.number(path, raw, present, r)
	}

	w.fail(fmt.Errorf("%w: %s", ErrUnsupportedType, t))
	return nil, false
}

func (w *walker) object(path []any, raw any, present bool, t reflect.Type) (any, bool) {
	m, ok := raw.(*Object)
	if !ok || !present {
		iss, message := invalidType("object", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	start := len(w.issues)
	out := NewObject()
	known := make(map[string]bool)
	strict := false
	w.fields(path, m, t, out, known, &strict)

	if strict {
		var unknown []string
		for _, key := range jsKeyOrder(m.Keys) {
			if !known[key] {
				unknown = append(unknown, key)
			}
		}
		if len(unknown) > 0 {
			iss, message := unrecognizedKeys(unknown)
			w.report(path, iss, message)
		}
	}

	if !w.abortedSince(start) && reflect.PointerTo(t).Implements(refinerType) {
		w.add(path, reflect.New(t).Interface().(refine.Refiner).Refine(m)...)
	}
	if len(w.issues) == start && reflect.PointerTo(t).Implements(piperType) {
		w.add(path, reflect.New(t).Interface().(refine.Piper).Pipe(m)...)
	}

	return out, true
}

func (w *walker) fields(path []any, m *Object, t reflect.Type, out *Object, known map[string]bool, strict *bool) {
	for f := range t.Fields() {
		if f.Type == strictType {
			*strict = true
			continue
		}

		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct && !isOptional(f.Type) {
			w.fields(path, m, f.Type, out, known, strict)
			continue
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		known[name] = true

		r, err := parseRules(f.Tag.Get("v"))
		if err != nil {
			w.fail(err)
			continue
		}
		items, err := parseRules(f.Tag.Get("vi"))
		if err != nil {
			w.fail(err)
			continue
		}

		child, present := m.Get(name)
		if v, keep := w.value(join(path, name), child, present, f.Type, r, items); keep {
			out.Set(name, v)
		}
	}
}

func (w *walker) slice(path []any, raw any, present bool, t reflect.Type, r rules, items rules) (any, bool) {
	if s, ok := raw.(string); ok && present && r.split {
		parts := strings.Split(s, ",")
		raw = make([]any, len(parts))
		for i, part := range parts {
			raw.([]any)[i] = part
		}
	}

	list, ok := raw.([]any)
	if !ok || !present {
		iss, message := invalidType("array", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	out := make([]any, 0, len(list))
	for i, item := range list {
		if v, keep := w.value(join(path, i), item, true, t.Elem(), items, rules{}); keep {
			out = append(out, v)
		}
	}

	if r.min != nil && int64(len(list)) < *r.min {
		iss, message := tooSmall("array", *r.min)
		w.report(path, iss, message)
	}
	if r.max != nil && int64(len(list)) > *r.max {
		iss, message := tooBig("array", *r.max)
		w.report(path, iss, message)
	}

	return out, true
}

func (w *walker) record(path []any, raw any, present bool) (any, bool) {
	if o, ok := raw.(*Object); ok && present {
		return o, true
	}

	iss, message := invalidType("record", received(raw, present))
	w.report(path, iss, message)
	return nil, false
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

func (w *walker) datetime(path []any, raw any, present bool) (any, bool) {
	const expected = "Invalid input: expected ISO 8601 datetime string, received "

	s, ok := raw.(string)
	if !ok || !present {
		iss, _ := invalidType("string", received(raw, present))
		w.report(path, iss, expected+typeOf(raw, present))
		return nil, false
	}

	if !datetimePattern.re.MatchString(s) {
		iss, _ := invalidFormat("datetime", datetimePattern.source, "")
		w.report(path, iss, expected+"string")
		return nil, false
	}

	t, err := ParseISODatetime(s)
	if err != nil {
		iss, _ := invalidFormat("datetime", datetimePattern.source, "")
		w.report(path, iss, expected+"string")
		return nil, false
	}

	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00"), true
}

func ParseISODatetime(s string) (time.Time, error) {
	m := datetimeParts.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, fmt.Errorf("%w: %q", ErrUnknownRule, s)
	}

	atoi := func(v string) int {
		n, _ := strconv.Atoi(v)
		return n
	}

	nanos := 0
	if m[7] != "" {
		fraction := (m[7] + "000")[:3]
		nanos = atoi(fraction) * int(time.Millisecond)
	}

	loc := time.UTC
	if m[8] != "Z" {
		sign := 1
		if m[8][0] == '-' {
			sign = -1
		}
		offset := sign * (atoi(m[8][1:3])*3600 + atoi(m[8][4:6])*60)
		loc = time.FixedZone("", offset)
	}

	return time.Date(atoi(m[1]), time.Month(atoi(m[2])), atoi(m[3]), atoi(m[4]), atoi(m[5]), atoi(m[6]), nanos, loc).UTC(), nil
}

func (w *walker) string(path []any, raw any, present bool, r rules) (any, bool) {
	s, ok := raw.(string)
	if !ok || !present {
		iss, message := invalidType("string", received(raw, present))
		switch {
		case r.time:
			message = "Invalid input: expected string in HH:MM format, received " + typeOf(raw, present)
		case r.email:
			message = "Invalid input: expected email, received " + typeOf(raw, present)
		case r.date:
			message = "Invalid input: expected ISO date string (YYYY-MM-DD), received " + typeOf(raw, present)
		}
		w.report(path, iss, message)
		return nil, false
	}

	if r.trim {
		s = jsTrim(s)
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
	if r.time && !timePattern.re.MatchString(s) {
		iss, message := invalidFormat("time", timePattern.source, "Invalid input: expected string in HH:MM format, received string")
		w.report(path, iss, message)
		valid = false
	}
	if r.url && !validURL(s) {
		w.report(path, newIssue("code", "invalid_format", "format", "url"), "Invalid URL")
		valid = false
	}
	if r.uuidV7 && !uuidV7Pattern.re.MatchString(s) {
		iss, message := invalidFormat("uuid", uuidV7Pattern.source, "Invalid UUID")
		w.report(path, iss, message)
		valid = false
	}
	if r.anyUUID && !anyUUIDPattern.re.MatchString(s) {
		iss, message := invalidFormat("uuid", anyUUIDPattern.source, "Invalid UUID")
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
		message := "Invalid string: must match pattern " + r.regex.source
		if r.regex.message != "" {
			message = r.regex.message
		}
		iss, message := invalidFormat("regex", r.regex.source, message)
		w.report(path, iss, message)
		valid = false
	}
	length := int64(len(utf16.Encode([]rune(s))))
	if r.min != nil && length < *r.min {
		iss, message := tooSmall("string", *r.min)
		w.report(path, iss, message)
		valid = false
	}
	if r.max != nil && length > *r.max {
		iss, message := tooBig("string", *r.max)
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

func (w *walker) bbox(path []any, raw any, present bool) (any, bool) {
	s, ok := raw.(string)
	if !ok || !present {
		iss, message := invalidType("string", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	custom := func(message string) (any, bool) {
		w.issues = append(w.issues, newIssue("code", "custom", "message", message, "path", append([]any{}, path...)))
		return nil, false
	}

	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return custom("bbox must have 4 comma-separated numbers: west,south,east,north")
	}
	values := make([]float64, len(parts))
	for i, part := range parts {
		if values[i] = jsStringToNumber(part); math.IsNaN(values[i]) {
			return custom("bbox parts must be valid numbers")
		}
	}

	names := []string{"west", "south", "east", "north"}
	limits := []int64{180, 90, 180, 90}
	start := len(w.issues)
	for i, v := range values {
		at := append(append([]any{}, path...), names[i])
		switch {
		case math.IsInf(v, 0):
			iss, message := invalidInfinity()
			w.report(at, iss, message)
		case v < float64(-limits[i]):
			iss, message := tooSmall("number", -limits[i])
			w.report(at, iss, message)
		case v > float64(limits[i]):
			iss, message := tooBig("number", limits[i])
			w.report(at, iss, message)
		}
	}
	if !w.abortedSince(start) && values[3] < values[1] {
		w.issues = append(w.issues, Custom(append(append([]any{}, path...), "north"),
			"North latitude must be greater than or equal to south latitude"))
	}
	if len(w.issues) > start {
		return nil, false
	}

	out := NewObject()
	for i, name := range names {
		out.Set(name, values[i])
	}

	return out, true
}

func (w *walker) finite(path []any, raw any, present bool, r rules) (float64, bool) {
	var f float64
	if r.coerce {
		f = jsNumber(raw, present)
	} else {
		n, ok := raw.(json.Number)
		if !ok || !present {
			iss, message := invalidType("number", received(raw, present))
			w.report(path, iss, message)
			return 0, false
		}

		var err error
		f, err = strconv.ParseFloat(string(n), 64)
		if err != nil && !math.IsInf(f, 0) {
			iss, message := invalidType("number", "string")
			w.report(path, iss, message)
			return 0, false
		}
	}

	switch {
	case math.IsNaN(f):
		iss, message := invalidNaN()
		w.report(path, iss, message)
		return 0, false
	case math.IsInf(f, 0):
		iss, message := invalidInfinity()
		w.report(path, iss, message)
		return 0, false
	}

	return f, true
}

func (w *walker) bounds(path []any, f float64, r rules) bool {
	valid := true
	if r.positive && f <= 0 {
		w.report(path, newIssue("origin", "number", "code", "too_small", "minimum", 0, "inclusive", false),
			"Too small: expected number to be >0")
		valid = false
	}
	if r.minF != nil && f < *r.minF {
		iss, message := tooSmallNumber(*r.minF)
		w.report(path, iss, message)
		valid = false
	}
	if r.maxF != nil && f > *r.maxF {
		iss, message := tooBigNumber(*r.maxF)
		w.report(path, iss, message)
		valid = false
	}
	if r.min != nil && f < float64(*r.min) {
		iss, message := tooSmall("number", *r.min)
		w.report(path, iss, message)
		valid = false
	}
	if r.max != nil && f > float64(*r.max) {
		iss, message := tooBig("number", *r.max)
		w.report(path, iss, message)
		valid = false
	}

	return valid
}

func (w *walker) number(path []any, raw any, present bool, r rules) (any, bool) {
	f, ok := w.finite(path, raw, present, r)
	if !ok {
		return nil, false
	}

	if !w.bounds(path, f, r) {
		return nil, false
	}

	return f, true
}

func (w *walker) integer(path []any, raw any, present bool, r rules) (any, bool) {
	f, ok := w.finite(path, raw, present, r)
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
	if !w.bounds(path, f, r) {
		valid = false
	}
	if !valid {
		return nil, false
	}

	return int64(f), true
}

func validate(raw any, present bool, t reflect.Type, m mode) (any, []Issue, error) {
	w := &walker{mode: m}

	var v any
	switch t.Kind() {
	case reflect.Struct:
		v, _ = w.object([]any{}, ordered(raw), present, t)
	case reflect.Slice:
		v, _ = w.slice([]any{}, ordered(raw), present, t, rules{}, rules{})
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedType, t)
	}
	if w.err != nil {
		return nil, nil, w.err
	}

	return v, w.issues, nil
}
