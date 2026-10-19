package assetmetadata

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/assetmetadata"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `asset_id, key, value, updated_at, update_id`

type Repository struct {
	db *database.Database
}

func New(db *database.Database) (*Repository, error) {
	return &Repository{db: db}, nil
}

func (repo *Repository) Startup() error {
	return nil
}

func (repo *Repository) Shutdown() error {
	return nil
}

func (repo *Repository) List(ctx context.Context, assetID uuid.UUID) ([]assetmetadata.Item, error) {
	return database.Rows[assetmetadata.Item](ctx, repo.db, `SELECT `+columns+` FROM asset_metadata
		WHERE asset_id = $1 ORDER BY key`,
		assetID,
	)
}

func (repo *Repository) Get(ctx context.Context, assetID uuid.UUID, key string) (*assetmetadata.Item, error) {
	return database.Row[assetmetadata.Item](ctx, repo.db, `SELECT `+columns+` FROM asset_metadata
		WHERE asset_id = $1 AND key = $2`,
		assetID, key,
	)
}

func (repo *Repository) Upsert(ctx context.Context, items []assetmetadata.Item, now time.Time) ([]assetmetadata.Item, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	out := make([]assetmetadata.Item, 0, len(items))
	for _, item := range items {
		saved, err := database.Row[assetmetadata.Item](ctx, tx, `INSERT INTO asset_metadata (`+columns+`)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (asset_id, key) DO UPDATE SET value = excluded.value,
				updated_at = excluded.updated_at, update_id = excluded.update_id
			RETURNING `+columns,
			item.AssetID, item.Key, item.Value, now, uuid.Must(uuid.NewV7()),
		)
		if err != nil {
			return nil, err
		}
		out = append(out, *saved)
	}

	return out, tx.Commit(ctx)
}

func (repo *Repository) Delete(ctx context.Context, items []assetmetadata.Item, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, item := range items {
		res, err := database.Exec(ctx, tx, `DELETE FROM asset_metadata WHERE asset_id = $1 AND key = $2`, item.AssetID, item.Key)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			continue
		}
		if _, err = database.Exec(ctx, tx, `INSERT INTO asset_metadata_audit (id, asset_id, key, deleted_at)
			VALUES ($1, $2, $3, $4)`,
			uuid.Must(uuid.NewV7()), item.AssetID, item.Key, now,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
