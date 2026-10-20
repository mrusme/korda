package syncstate

import (
	"context"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type pair struct {
	A uuid.UUID
	B uuid.UUID
}

func Audit(ctx context.Context, ex database.Executor, table string, columns string, query string, now time.Time, args ...any) error {
	rows, err := database.Rows[pair](ctx, ex, query, args...)
	if err != nil {
		return err
	}

	for _, r := range rows {
		if _, err = database.Exec(ctx, ex, `INSERT INTO `+table+` (id, `+columns+`, deleted_at) VALUES ($1, $2, $3, $4)`,
			uuid.Must(uuid.NewV7()), r.A, r.B, now,
		); err != nil {
			return err
		}
	}

	return nil
}

func Bump(ctx context.Context, ex database.Executor, table string, set string, ids []uuid.UUID, now time.Time) error {
	if set != "" {
		set += ", "
	}

	for _, id := range ids {
		if _, err := database.Exec(ctx, ex, `UPDATE `+table+` SET `+set+`updated_at = $1, update_id = $2 WHERE id = $3`,
			now, uuid.Must(uuid.NewV7()), id,
		); err != nil {
			return err
		}
	}

	return nil
}
