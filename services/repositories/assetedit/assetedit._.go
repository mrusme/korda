package assetedit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/assetedit"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, asset_id, action, parameters, sequence, updated_at, update_id`

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

type Item struct {
	Action     assetedit.Action
	Parameters string
}

func (repo *Repository) ReplaceAll(ctx context.Context, assetID uuid.UUID, edits []Item, now time.Time) ([]assetedit.Edit, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	old, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM asset_edits WHERE asset_id = $1 ORDER BY sequence`, assetID)
	if err != nil {
		return nil, err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM asset_edits WHERE asset_id = $1`, assetID); err != nil {
		return nil, err
	}
	for _, id := range old {
		if _, err = database.Exec(ctx, tx, `INSERT INTO asset_edit_audit (id, edit_id, asset_id, deleted_at)
			VALUES ($1, $2, $3, $4)`, uuid.Must(uuid.NewV7()), id, assetID, now); err != nil {
			return nil, err
		}
	}

	out := make([]assetedit.Edit, 0, len(edits))
	for i, e := range edits {
		created, err := database.Row[assetedit.Edit](ctx, tx, `INSERT INTO asset_edits (`+columns+`)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+columns,
			uuid.New(), assetID, e.Action, e.Parameters, i, now, uuid.Must(uuid.NewV7()))
		if err != nil {
			return nil, err
		}
		out = append(out, *created)
	}

	if _, err = database.Exec(ctx, tx, `UPDATE assets SET is_edited = $1, updated_at = $2, update_id = $3
		WHERE id = $4 AND is_edited != $1`, len(edits) > 0, now, uuid.Must(uuid.NewV7()), assetID); err != nil {
		return nil, err
	}

	return out, tx.Commit(ctx)
}

func (repo *Repository) All(ctx context.Context, assetID uuid.UUID) ([]assetedit.Edit, error) {
	return database.Rows[assetedit.Edit](ctx, repo.db, `SELECT `+columns+` FROM asset_edits
		WHERE asset_id = $1 ORDER BY sequence`, assetID)
}
