package jsontime

import (
	"database/sql"
	"strconv"
	"time"
)

const Layout = "2006-01-02T15:04:05.000Z"

type Time struct {
	time.Time
}

func New(t time.Time) Time {
	return Time{Time: t.UTC()}
}

func Nullable(t sql.Null[time.Time]) *Time {
	if !t.Valid {
		return nil
	}

	value := New(t.V)
	return &value
}

func Format(t time.Time) string {
	return t.UTC().Format(Layout)
}

func (t Time) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, Format(t.Time)), nil
}

func (t *Time) UnmarshalJSON(b []byte) error {
	raw, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	parsed, err := Parse(raw)
	if err != nil {
		return err
	}

	t.Time = parsed
	return nil
}

func Parse(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, err
	}

	return parsed.UTC(), nil
}
