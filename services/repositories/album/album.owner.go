package album

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/services/repositories/syncstate"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) setDeletedOwned(ctx context.Context, userID uuid.UUID, deletedAt any, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	ids, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM albums WHERE id IN (`+owned+`)`, userID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = database.Exec(ctx, tx, `UPDATE albums SET deleted_at = $1, updated_at = $2, update_id = $3 WHERE id = $4`,
			deletedAt, now, uuid.Must(uuid.NewV7()), id,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) SoftDeleteOwned(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return repo.setDeletedOwned(ctx, userID, now, now)
}

func (repo *Repository) RestoreOwned(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return repo.setDeletedOwned(ctx, userID, nil, now)
}

func (repo *Repository) RemoveAssetsFromAll(ctx context.Context, assetIDs []uuid.UUID, now time.Time) error {
	if len(assetIDs) == 0 {
		return nil
	}

	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, id := range assetIDs {
		if err = syncstate.Audit(ctx, tx, "album_asset_audit", "album_id, asset_id", `SELECT album_id AS a, asset_id AS b
			FROM album_assets WHERE asset_id = $1`, now, id); err != nil {
			return err
		}
		if _, err = database.Exec(ctx, tx, `DELETE FROM album_assets WHERE asset_id = $1`, id); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
