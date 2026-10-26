package assetfile

import (
	"context"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/services/repositories/query"
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

type Filter struct {
	AssetID       uuid.UUID
	Type          *assetfile.Type
	IsEdited      *bool
	IsProgressive *bool
	IsTransparent *bool
}

func (repo *Repository) ByID(ctx context.Context, id uuid.UUID) (*assetfile.File, error) {
	return database.Row[assetfile.File](ctx, repo.db, `SELECT `+columns+` FROM asset_files WHERE id = $1`, id)
}

func (repo *Repository) Search(ctx context.Context, f Filter) ([]assetfile.File, error) {
	args := []any{f.AssetID}
	where := `asset_id = $1`
	add := func(column string, v any) {
		args = append(args, v)
		where += ` AND ` + column + ` = $` + strconv.Itoa(len(args))
	}
	if f.Type != nil {
		add("type", *f.Type)
	}
	if f.IsEdited != nil {
		add("is_edited", *f.IsEdited)
	}
	if f.IsProgressive != nil {
		add("is_progressive", *f.IsProgressive)
	}
	if f.IsTransparent != nil {
		add("is_transparent", *f.IsTransparent)
	}

	return database.Rows[assetfile.File](ctx, repo.db, `SELECT `+columns+` FROM asset_files WHERE `+where+` ORDER BY rowid`, args...)
}

func (repo *Repository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM asset_files WHERE id = $1`, id)
	return err
}

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, elevated bool) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID, elevated}
		in := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT f.id FROM asset_files f JOIN assets a ON a.id = f.asset_id
			WHERE a.owner_id = $1 AND ($2 OR a.visibility != 'locked') AND f.id IN (`+in+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
