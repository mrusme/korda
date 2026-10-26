package request

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

type Issue struct {
	keys   []string
	values []any
	abort  bool
}

func newIssue(pairs ...any) Issue {
	var iss Issue
	for i := 0; i+1 < len(pairs); i += 2 {
		iss.keys = append(iss.keys, pairs[i].(string))
		iss.values = append(iss.values, pairs[i+1])
	}

	return iss
}

func (iss Issue) Get(key string) any {
	for i, k := range iss.keys {
		if k == key {
			return iss.values[i]
		}
	}

	return nil
}

func (iss Issue) withPath(path []any, message string) Issue {
	out := Issue{
		keys:   append(append([]string(nil), iss.keys...), "path", "message"),
		values: append(append([]any(nil), iss.values...), append([]any{}, path...), message),
		abort:  iss.abort,
	}

	return out
}

func (iss Issue) aborting() Issue {
	iss.abort = true
	return iss
}

func (iss Issue) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range iss.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		encoded, err := marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(encoded)
		buf.WriteByte(':')
		if encoded, err = marshal(iss.values[i]); err != nil {
			return nil, err
		}
		buf.Write(encoded)
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}

func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func invalidType(expected string, received string) (Issue, string) {
	return newIssue("expected", expected, "code", "invalid_type").aborting(),
		"Invalid input: expected " + expected + ", received " + received
}

func invalidInt() (Issue, string) {
	return newIssue("expected", "int", "format", "safeint", "code", "invalid_type").aborting(),
		"Invalid input: expected int, received number"
}

func invalidInfinity() (Issue, string) {
	return newIssue("expected", "number", "code", "invalid_type", "received", "Infinity").aborting(),
		"Invalid input: expected number, received number"
}

func invalidNaN() (Issue, string) {
	return newIssue("expected", "number", "code", "invalid_type", "received", "NaN").aborting(),
		"Invalid input: expected number, received NaN"
}

func unsafeInt(positive bool) (Issue, string) {
	const note = "Integers must be within the safe integer range."
	if positive {
		return newIssue("code", "too_big", "maximum", int64(maxSafeInteger), "note", note, "origin", "int", "inclusive", true),
			"Too big: expected int to be <=" + strconv.FormatInt(maxSafeInteger, 10)
	}

	return newIssue("code", "too_small", "minimum", int64(-maxSafeInteger), "note", note, "origin", "int", "inclusive", true),
		"Too small: expected int to be >=" + strconv.FormatInt(-maxSafeInteger, 10)
}

func tooBig(origin string, maximum int64) (Issue, string) {
	iss := newIssue("origin", origin, "code", "too_big", "maximum", maximum, "inclusive", true)
	limit := strconv.FormatInt(maximum, 10)

	switch origin {
	case "string":
		return iss, "Too big: expected string to have <=" + limit + " characters"
	case "array":
		return iss, "Too big: expected array to have <=" + limit + " items"
	}

	return iss, "Too big: expected number to be <=" + limit
}

func tooSmall(origin string, minimum int64) (Issue, string) {
	iss := newIssue("origin", origin, "code", "too_small", "minimum", minimum, "inclusive", true)
	limit := strconv.FormatInt(minimum, 10)

	switch origin {
	case "string":
		return iss, "Too small: expected string to have >=" + limit + " characters"
	case "array":
		return iss, "Too small: expected array to have >=" + limit + " items"
	}

	return iss, "Too small: expected number to be >=" + limit
}

func tooBigNumber(maximum float64) (Issue, string) {
	return newIssue("origin", "number", "code", "too_big", "maximum", maximum, "inclusive", true),
		"Too big: expected number to be <=" + strconv.FormatFloat(maximum, 'g', -1, 64)
}

func tooSmallNumber(minimum float64) (Issue, string) {
	return newIssue("origin", "number", "code", "too_small", "minimum", minimum, "inclusive", true),
		"Too small: expected number to be >=" + strconv.FormatFloat(minimum, 'g', -1, 64)
}

func invalidFormat(format string, pattern string, message string) (Issue, string) {
	return newIssue("origin", "string", "code", "invalid_format", "format", format, "pattern", pattern), message
}

func invalidValue(values []string) (Issue, string) {
	message := "Invalid option: expected one of " + quoted(values)
	if len(values) == 1 {
		message = "Invalid input: expected " + quoted(values)
	}

	return newIssue("code", "invalid_value", "values", values).aborting(), message
}

func invalidStringBool() (Issue, string) {
	values := []string{"true", "false"}
	return newIssue("code", "invalid_value", "expected", "stringbool", "values", values).aborting(),
		"Invalid option: expected one of " + quoted(values)
}

func unrecognizedKeys(keys []string) (Issue, string) {
	message := "Unrecognized key: " + quoted(keys)
	if len(keys) > 1 {
		parts := make([]string, len(keys))
		for i, key := range keys {
			encoded, _ := marshal(key)
			parts[i] = string(encoded)
		}
		message = "Unrecognized keys: " + strings.Join(parts, ", ")
	}

	return newIssue("code", "unrecognized_keys", "keys", keys).aborting(), message
}

func invalidJSON(path []any) Issue {
	return newIssue("code", "custom", "message", "Invalid input: expected JSON string, received string", "path",
		append([]any{}, path...)).aborting()
}

func Custom(path []any, message string) Issue {
	return newIssue("code", "custom").withPath(path, message)
}

func quoted(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		encoded, _ := marshal(v)
		parts[i] = string(encoded)
	}

	return strings.Join(parts, "|")
}
