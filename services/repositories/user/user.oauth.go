package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) UnlinkOAuth(ctx context.Context, id uuid.UUID, now time.Time) (*user.User, error) {
	return database.Row[user.User](ctx, repo.db, `UPDATE users SET oauth_id = NULL, updated_at = $2, update_id = $3
		WHERE id = $1 RETURNING `+columns, id, now, uuid.Must(uuid.NewV7()))
}

func (repo *Repository) UnlinkAllOAuth(ctx context.Context, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE users SET oauth_id = NULL, updated_at = $1, update_id = $2`,
		now, uuid.Must(uuid.NewV7()))

	return err
}
