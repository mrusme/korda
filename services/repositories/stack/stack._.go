package stack

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/stack"
	"tty.fail/mrus/korda/services/repositories/query"
	"tty.fail/mrus/korda/services/repositories/syncstate"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, owner_id, primary_asset_id, created_at, updated_at, update_id`

type Repository struct {
	db *database.Database
}

func New(db *database.Database) (*Repository, error) {
	repo := new(Repository)
	repo.db = db

	return repo, nil
}

func (repo *Repository) Startup() error {
	return nil
}

func (repo *Repository) Shutdown() error {
	return nil
}

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*stack.Stack, error) {
	return database.Row[stack.Stack](ctx, repo.db, `SELECT `+columns+` FROM stacks WHERE id = $1`, id)
}

func (repo *Repository) Search(ctx context.Context, ownerID uuid.UUID, primaryAssetID uuid.NullUUID) ([]stack.Stack, error) {
	args := []any{ownerID}
	where := ``
	if primaryAssetID.Valid {
		args = append(args, primaryAssetID.UUID)
		where = ` AND primary_asset_id = $2`
	}

	return database.Rows[stack.Stack](ctx, repo.db, `SELECT `+columns+` FROM stacks WHERE owner_id = $1`+where+
		` ORDER BY update_id`, args...)
}

func (repo *Repository) OwnedBy(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{ownerID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM stacks
			WHERE owner_id = $1 AND id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func remove(ctx context.Context, tx database.Executor, ids []uuid.UUID, now time.Time) error {
	for _, id := range ids {
		members, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM assets WHERE stack_id = $1`, id)
		if err != nil {
			return err
		}
		if err = syncstate.Bump(ctx, tx, "assets", "stack_id = NULL", members, now); err != nil {
			return err
		}
		if _, err = database.Exec(ctx, tx, `INSERT INTO stack_audit (id, stack_id, user_id, deleted_at)
			SELECT $1, id, owner_id, $2 FROM stacks WHERE id = $3`,
			uuid.Must(uuid.NewV7()), now, id,
		); err != nil {
			return err
		}
		if _, err = database.Exec(ctx, tx, `DELETE FROM stacks WHERE id = $1`, id); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) Create(ctx context.Context, ownerID uuid.UUID, assetIDs []uuid.UUID, now time.Time) (*stack.Stack, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	args := []any{ownerID}
	marks := query.Placeholders(&args, assetIDs)
	merged, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM stacks
		WHERE owner_id = $1 AND primary_asset_id IN (`+marks+`)`, args...)
	if err != nil {
		return nil, err
	}

	members := slices.Clone(assetIDs)
	for _, id := range merged {
		children, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM assets
			WHERE stack_id = $1 AND deleted_at IS NULL`, id)
		if err != nil {
			return nil, err
		}
		members = append(members, children...)
	}
	if err = remove(ctx, tx, merged, now); err != nil {
		return nil, err
	}

	created, err := database.Row[stack.Stack](ctx, tx, `INSERT INTO stacks (`+columns+`)
		VALUES ($1, $2, $3, $4, $4, $5)
		RETURNING `+columns,
		uuid.New(), ownerID, assetIDs[0], now, uuid.Must(uuid.NewV7()),
	)
	if err != nil {
		return nil, err
	}

	seen := map[uuid.UUID]bool{}
	for _, id := range members {
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, err = database.Exec(ctx, tx, `UPDATE assets SET stack_id = $1, updated_at = $2, update_id = $3 WHERE id = $4`,
			created.ID, now, uuid.Must(uuid.NewV7()), id,
		); err != nil {
			return nil, err
		}
	}

	return created, tx.Commit(ctx)
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, primaryAssetID uuid.NullUUID, now time.Time) (*stack.Stack, error) {
	return database.Row[stack.Stack](ctx, repo.db, `UPDATE stacks
		SET primary_asset_id = COALESCE($1, primary_asset_id), updated_at = $2, update_id = $3
		WHERE id = $4
		RETURNING `+columns,
		primaryAssetID, now, uuid.Must(uuid.NewV7()), id,
	)
}

func (repo *Repository) Delete(ctx context.Context, ids []uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	if err = remove(ctx, tx, ids, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo *Repository) Merge(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	members, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM assets WHERE stack_id = $1`, sourceID)
	if err != nil {
		return err
	}
	for _, id := range members {
		if _, err = database.Exec(ctx, tx, `UPDATE assets SET stack_id = $1, updated_at = $2, update_id = $3 WHERE id = $4`,
			targetID, now, uuid.Must(uuid.NewV7()), id); err != nil {
			return err
		}
	}
	if err = remove(ctx, tx, []uuid.UUID{sourceID}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
