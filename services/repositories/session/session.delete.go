package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (repo *Repository) DeleteAllOfUser(ctx context.Context, userID uuid.UUID, except uuid.NullUUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM sessions
		WHERE user_id = $1 AND ($2 IS NULL OR id <> $2)`,
		userID, except,
	)

	return err
}

func (repo *Repository) Cleanup(ctx context.Context, now time.Time) (int64, error) {
	res, err := database.Exec(ctx, repo.db, `DELETE FROM sessions
		WHERE updated_at <= $1 OR (expires_at IS NOT NULL AND expires_at <= $2)`,
		now.AddDate(0, 0, -90), now,
	)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected(), nil
}
