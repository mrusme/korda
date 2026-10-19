package assetexif

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Locked int

const (
	Override Locked = iota
	Append
	Skip
)

var Lockable = []string{"description", "dateTimeOriginal", "latitude", "longitude", "rating", "timeZone", "tags"}

var properties = map[string]string{
	"description":      "description",
	"dateTimeOriginal": "date_time_original",
	"latitude":         "latitude",
	"longitude":        "longitude",
	"rating":           "rating",
	"timeZone":         "time_zone",
	"tags":             "tags",
}

type Write struct {
	Column string
	Value  any
}

func property(column string) string {
	for name, col := range properties {
		if col == column {
			return name
		}
	}

	return ""
}

func Names(writes []Write) []string {
	out := make([]string, 0, len(writes))
	for _, w := range writes {
		if name := property(w.Column); name != "" {
			out = append(out, name)
		}
	}

	return out
}

func LockedFor(writes []Write) []string {
	var out []string
	for _, name := range Lockable {
		if slices.ContainsFunc(writes, func(w Write) bool { return w.Column == properties[name] }) {
			out = append(out, name)
		}
	}

	return out
}

func (repo *Repository) Write(ctx context.Context, ex database.Executor, assetID uuid.UUID, writes []Write, behavior Locked, lock []string, now time.Time) error {
	if ex == nil {
		ex = repo.db
	}

	columns := []string{"asset_id"}
	args := []any{assetID}
	marks := []string{"$1"}
	var set []string

	for _, w := range writes {
		args = append(args, w.Value)
		columns = append(columns, w.Column)
		marks = append(marks, "$"+strconv.Itoa(len(args)))

		value := "excluded." + w.Column
		if name := property(w.Column); behavior == Skip && name != "" {
			value = `CASE WHEN EXISTS (SELECT 1 FROM json_each(COALESCE(asset_exif.locked_properties, '[]'))
				WHERE value = '` + name + `') THEN asset_exif.` + w.Column + ` ELSE excluded.` + w.Column + ` END`
		}
		set = append(set, w.Column+" = "+value)
	}

	if behavior == Append && len(lock) > 0 {
		encoded, err := json.Marshal(lock)
		if err != nil {
			return err
		}
		args = append(args, string(encoded))
		columns = append(columns, "locked_properties")
		marks = append(marks, "$"+strconv.Itoa(len(args)))
		set = append(set, `locked_properties = (SELECT json_group_array(DISTINCT value) FROM (
			SELECT value FROM json_each(COALESCE(asset_exif.locked_properties, '[]'))
			UNION ALL SELECT value FROM json_each(excluded.locked_properties)))`)
	}

	args = append(args, now)
	columns = append(columns, "updated_at")
	marks = append(marks, "$"+strconv.Itoa(len(args)))
	args = append(args, uuid.Must(uuid.NewV7()))
	columns = append(columns, "update_id")
	marks = append(marks, "$"+strconv.Itoa(len(args)))
	set = append(set, "updated_at = excluded.updated_at", "update_id = excluded.update_id")

	_, err := database.Exec(ctx, ex, `INSERT INTO asset_exif (`+strings.Join(columns, ", ")+`)
		VALUES (`+strings.Join(marks, ", ")+`)
		ON CONFLICT (asset_id) DO UPDATE SET `+strings.Join(set, ", "),
		args...,
	)

	return err
}

func (repo *Repository) UpdateAll(ctx context.Context, assetIDs []uuid.UUID, writes []Write, lock []string, now time.Time) error {
	if len(assetIDs) == 0 || len(writes) == 0 {
		return nil
	}

	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	encoded, err := json.Marshal(lock)
	if err != nil {
		return err
	}

	var set []string
	args := []any{}
	for _, w := range writes {
		args = append(args, w.Value)
		set = append(set, w.Column+" = $"+strconv.Itoa(len(args)))
	}
	args = append(args, string(encoded))
	set = append(set, `locked_properties = (SELECT json_group_array(DISTINCT value) FROM (
		SELECT value FROM json_each(COALESCE(locked_properties, '[]'))
		UNION ALL SELECT value FROM json_each($`+strconv.Itoa(len(args))+`)))`)
	args = append(args, now)
	set = append(set, "updated_at = $"+strconv.Itoa(len(args)))
	args = append(args, nil, nil)
	set = append(set, "update_id = $"+strconv.Itoa(len(args)-1))

	query := `UPDATE asset_exif SET ` + strings.Join(set, ", ") + ` WHERE asset_id = $` + strconv.Itoa(len(args))
	for _, id := range assetIDs {
		args[len(args)-2] = uuid.Must(uuid.NewV7())
		args[len(args)-1] = id
		if _, err = database.Exec(ctx, tx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) ShiftDateTimeOriginal(ctx context.Context, assetIDs []uuid.UUID, minutes int64, timeZone *string, now time.Time) error {
	if len(assetIDs) == 0 {
		return nil
	}

	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, id := range assetIDs {
		original, err := database.Value[sql.Null[time.Time]](ctx, tx, `SELECT date_time_original FROM asset_exif WHERE asset_id = $1`, id)
		if errors.Is(err, errs.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if original.Valid {
			original.V = original.V.Add(time.Duration(minutes) * time.Minute)
		}

		set := `date_time_original = $1`
		args := []any{original}
		if timeZone != nil {
			args = append(args, *timeZone)
			set += `, time_zone = $2`
		}
		args = append(args, now, uuid.Must(uuid.NewV7()), id)
		n := len(args)
		if _, err = database.Exec(ctx, tx, `UPDATE asset_exif SET `+set+`,
			locked_properties = (SELECT json_group_array(DISTINCT value) FROM (
				SELECT value FROM json_each(COALESCE(locked_properties, '[]'))
				UNION ALL SELECT value FROM json_each('["dateTimeOriginal","timeZone"]'))),
			updated_at = $`+strconv.Itoa(n-2)+`, update_id = $`+strconv.Itoa(n-1)+`
			WHERE asset_id = $`+strconv.Itoa(n),
			args...,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
