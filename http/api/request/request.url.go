package request

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

var timePattern = newPattern(`^(?:[01]\d|2[0-3]):[0-5]\d$`, "")

func validURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || strings.ContainsAny(s, " \t\n") {
		return false
	}

	return u.Host != "" || u.Opaque != ""
}

func emptyOrURL(message string) func(raw any) string {
	return func(raw any) string {
		if s, _ := raw.(string); s == "" || validURL(s) {
			return ""
		}
		return message
	}
}

func (w *walker) numberEnum(path []any, raw any, present bool, values []int64) (any, bool) {
	if n, ok := raw.(json.Number); ok && present {
		for _, v := range values {
			if string(n) == strconv.FormatInt(v, 10) {
				return v, true
			}
		}
	}

	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatInt(v, 10))
	}
	w.report(path, newIssue("code", "invalid_value", "values", values).aborting(),
		"Invalid option: expected one of "+strings.Join(parts, "|"))

	return nil, false
}
