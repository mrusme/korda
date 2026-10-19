package systemmetadata

import (
	"context"

	"tty.fail/mrus/korda/models/systemmetadata"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Get(ctx context.Context, key systemmetadata.Key) (string, error) {
	return database.Value[string](ctx, repo.db, `SELECT value FROM system_metadata WHERE key = $1`, key)
}

func (repo *Repository) Set(ctx context.Context, key systemmetadata.Key, value string) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO system_metadata (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`,
		key, value,
	)

	return err
}
