package apikey

import (
	"context"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/apikey"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) GetByKey(ctx context.Context, key []byte) (*apikey.APIKey, error) {
	return database.Row[apikey.APIKey](ctx, repo.db, `SELECT k.id AS id, k.name AS name, k.key AS key,
		k.user_id AS user_id, k.permissions AS permissions, k.created_at AS created_at,
		k.updated_at AS updated_at, k.update_id AS update_id
		FROM api_keys k JOIN users u ON u.id = k.user_id AND u.deleted_at IS NULL
		WHERE k.key = $1`,
		key,
	)
}

func (repo *Repository) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*apikey.APIKey, error) {
	return database.Row[apikey.APIKey](ctx, repo.db, `SELECT `+columns+` FROM api_keys
		WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
}

func (repo *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]apikey.APIKey, error) {
	return database.Rows[apikey.APIKey](ctx, repo.db, `SELECT `+columns+` FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
}
