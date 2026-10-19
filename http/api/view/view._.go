package view

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/jsontime"
)

func nullable[T any](v sql.Null[T]) *T {
	if !v.Valid {
		return nil
	}

	value := v.V
	return &value
}

func time3(t time.Time) jsontime.Time {
	return jsontime.New(t)
}

func id(u uuid.UUID) string {
	return u.String()
}
