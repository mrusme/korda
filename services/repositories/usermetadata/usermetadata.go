package usermetadata

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/usermetadata"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `user_id, key, value, updated_at, update_id`

func (repo *Repository) Get(ctx context.Context, userID uuid.UUID, key usermetadata.Key) (*usermetadata.UserMetadata, error) {
	return database.Row[usermetadata.UserMetadata](ctx, repo.db, `SELECT `+columns+` FROM user_metadata
		WHERE user_id = $1 AND key = $2`,
		userID, key,
	)
}

func (repo *Repository) List(ctx context.Context, userID uuid.UUID) ([]usermetadata.UserMetadata, error) {
	return database.Rows[usermetadata.UserMetadata](ctx, repo.db, `SELECT `+columns+` FROM user_metadata
		WHERE user_id = $1
		ORDER BY key`,
		userID,
	)
}

func (repo *Repository) Upsert(ctx context.Context, userID uuid.UUID, key usermetadata.Key, value string, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO user_metadata (`+columns+`)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at,
			update_id = excluded.update_id`,
		userID, key, value, now, uuid.Must(uuid.NewV7()),
	)

	return err
}

func (repo *Repository) Delete(ctx context.Context, userID uuid.UUID, key usermetadata.Key, now time.Time) (err error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	res, err := database.Exec(ctx, tx, `DELETE FROM user_metadata WHERE user_id = $1 AND key = $2`, userID, key)
	if err != nil {
		return err
	}
	if res.RowsAffected() > 0 {
		if _, err = database.Exec(ctx, tx, `INSERT INTO user_metadata_audit (id, user_id, key, deleted_at)
			VALUES ($1, $2, $3, $4)`,
			uuid.Must(uuid.NewV7()), userID, key, now,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
