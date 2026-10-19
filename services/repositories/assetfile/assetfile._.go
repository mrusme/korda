package assetfile

import (
	"context"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/assetfile"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, asset_id, type, path, is_edited, is_progressive, is_transparent, created_at, updated_at, update_id`

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

func (repo *Repository) List(ctx context.Context, assetID uuid.UUID) ([]assetfile.File, error) {
	return database.Rows[assetfile.File](ctx, repo.db, `SELECT `+columns+` FROM asset_files
		WHERE asset_id = $1 ORDER BY type, is_edited`,
		assetID,
	)
}

func (repo *Repository) Get(ctx context.Context, assetID uuid.UUID, kind assetfile.Type, edited bool) (*assetfile.File, error) {
	return database.Row[assetfile.File](ctx, repo.db, `SELECT `+columns+` FROM asset_files
		WHERE asset_id = $1 AND type = $2 AND is_edited = $3`,
		assetID, kind, edited,
	)
}

func (repo *Repository) Upsert(ctx context.Context, f *assetfile.File) (string, error) {
	previous, err := database.Values[string](ctx, repo.db, `SELECT path FROM asset_files
		WHERE asset_id = $1 AND type = $2 AND is_edited = $3`,
		f.AssetID, f.Type, f.IsEdited,
	)
	if err != nil {
		return "", err
	}

	if _, err = database.Exec(ctx, repo.db, `INSERT INTO asset_files (`+columns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (asset_id, type, is_edited) DO UPDATE SET path = excluded.path,
			is_progressive = excluded.is_progressive, is_transparent = excluded.is_transparent,
			updated_at = excluded.updated_at, update_id = excluded.update_id`,
		f.ID, f.AssetID, f.Type, f.Path, f.IsEdited, f.IsProgressive, f.IsTransparent, f.CreatedAt, f.UpdatedAt, f.UpdateID,
	); err != nil {
		return "", err
	}

	for _, p := range previous {
		if p != f.Path {
			return p, nil
		}
	}

	return "", nil
}

func (repo *Repository) Delete(ctx context.Context, assetID uuid.UUID, kind assetfile.Type, edited bool) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM asset_files WHERE asset_id = $1 AND type = $2 AND is_edited = $3`,
		assetID, kind, edited,
	)

	return err
}
