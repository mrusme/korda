package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID, now time.Time) (err error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	if _, err = database.Exec(ctx, tx, `DELETE FROM users WHERE id = $1`, id); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `INSERT INTO user_audit (id, user_id, deleted_at)
		VALUES ($1, $2, $3)`,
		uuid.Must(uuid.NewV7()), id, now,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
