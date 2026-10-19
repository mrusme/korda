package apikey

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/apikey"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Update(
	ctx context.Context,
	userID uuid.UUID,
	id uuid.UUID,
	name *string,
	permissions permission.List,
	key []byte,
	now time.Time,
) (*apikey.APIKey, error) {
	var nameArg, permissionsArg, keyArg any
	if name != nil {
		nameArg = *name
	}
	if permissions != nil {
		permissionsArg = permissions
	}
	if key != nil {
		keyArg = key
	}

	return database.Row[apikey.APIKey](ctx, repo.db, `UPDATE api_keys SET
		name = COALESCE($1, name),
		permissions = COALESCE($2, permissions),
		key = COALESCE($3, key),
		updated_at = $4,
		update_id = $5
		WHERE id = $6 AND user_id = $7
		RETURNING `+columns,
		nameArg, permissionsArg, keyArg, now, uuid.Must(uuid.NewV7()), id, userID,
	)
}

func (repo *Repository) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}
