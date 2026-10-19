package apikey

import (
	"context"

	"tty.fail/mrus/korda/models/apikey"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Create(ctx context.Context, k *apikey.APIKey) (*apikey.APIKey, error) {
	return database.Row[apikey.APIKey](ctx, repo.db, `INSERT INTO api_keys (`+columns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+columns,
		k.ID, k.Name, k.Key, k.UserID, k.Permissions, k.CreatedAt, k.UpdatedAt, k.UpdateID,
	)
}
