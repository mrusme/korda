package asset

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/services/repositories/syncstate"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Removed struct {
	Asset          *asset.Asset
	FileSizeInByte sql.Null[int64]
	Files          []assetfile.File
}

func (repo *Repository) Remove(ctx context.Context, id uuid.UUID, now time.Time) (*Removed, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	a, err := database.Row[asset.Asset](ctx, tx, `SELECT `+columns+` FROM assets WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}

	removed := &Removed{Asset: a}
	if removed.FileSizeInByte, err = database.Value[sql.Null[int64]](ctx, tx, `SELECT file_size_in_byte FROM asset_exif
		WHERE asset_id = $1`, id); err != nil && !errors.Is(err, errs.ErrNoRows) {
		return nil, err
	}

	if removed.Files, err = database.Rows[assetfile.File](ctx, tx, `SELECT id, asset_id, type, path, is_edited,
		is_progressive, is_transparent, created_at, updated_at, update_id FROM asset_files WHERE asset_id = $1`, id); err != nil {
		return nil, err
	}

	stills, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM assets WHERE live_photo_video_id = $1`, id)
	if err != nil {
		return nil, err
	}
	for _, still := range stills {
		if _, err = database.Exec(ctx, tx, `UPDATE assets SET live_photo_video_id = NULL, updated_at = $1, update_id = $2
			WHERE id = $3`,
			now, uuid.Must(uuid.NewV7()), still,
		); err != nil {
			return nil, err
		}
	}
	if a.StackID.Valid {
		if err = detachFromStack(ctx, tx, a, now); err != nil {
			return nil, err
		}
	}
	thumbnails, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM albums WHERE album_thumbnail_asset_id = $1`, id)
	if err != nil {
		return nil, err
	}
	if err = syncstate.Bump(ctx, tx, "albums", "album_thumbnail_asset_id = NULL", thumbnails, now); err != nil {
		return nil, err
	}
	if err = syncstate.Audit(ctx, tx, "album_asset_audit", "album_id, asset_id", `SELECT album_id AS a, asset_id AS b
		FROM album_assets WHERE asset_id = $1`, now, id); err != nil {
		return nil, err
	}
	if err = syncstate.Audit(ctx, tx, "memory_asset_audit", "memory_id, asset_id", `SELECT memory_id AS a, asset_id AS b
		FROM memory_assets WHERE asset_id = $1`, now, id); err != nil {
		return nil, err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM assets WHERE id = $1`, id); err != nil {
		return nil, err
	}
	if _, err = database.Exec(ctx, tx, `INSERT INTO asset_audit (id, asset_id, owner_id, deleted_at) VALUES ($1, $2, $3, $4)`,
		uuid.Must(uuid.NewV7()), a.ID, a.OwnerID, now,
	); err != nil {
		return nil, err
	}
	if !a.LibraryID.Valid {
		if _, err = database.Exec(ctx, tx, `UPDATE users
			SET quota_usage_in_bytes = quota_usage_in_bytes - $1, updated_at = $2, update_id = $3
			WHERE id = $4 AND deleted_at IS NULL`,
			removed.FileSizeInByte.V, now, uuid.Must(uuid.NewV7()), a.OwnerID,
		); err != nil {
			return nil, err
		}
	}

	return removed, tx.Commit(ctx)
}

func detachFromStack(ctx context.Context, tx database.Executor, a *asset.Asset, now time.Time) error {
	stackID := a.StackID.UUID
	primary, err := database.Value[uuid.UUID](ctx, tx, `SELECT primary_asset_id FROM stacks WHERE id = $1`, stackID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	remaining, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM assets
		WHERE stack_id = $1 AND id <> $2 AND id <> $3 AND visibility = 'timeline' AND status <> 'deleted'
		ORDER BY created_at, id`,
		stackID, primary, a.ID,
	)
	if err != nil {
		return err
	}
	count := len(remaining)
	if primary != a.ID {
		count++
	}

	switch {
	case count < 2:
		members, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM assets WHERE stack_id = $1`, stackID)
		if err != nil {
			return err
		}
		if err = syncstate.Bump(ctx, tx, "assets", "stack_id = NULL", members, now); err != nil {
			return err
		}
		if _, err = database.Exec(ctx, tx, `INSERT INTO stack_audit (id, stack_id, user_id, deleted_at)
			SELECT $1, id, owner_id, $2 FROM stacks WHERE id = $3`,
			uuid.Must(uuid.NewV7()), now, stackID,
		); err != nil {
			return err
		}
		_, err = database.Exec(ctx, tx, `DELETE FROM stacks WHERE id = $1`, stackID)
		return err
	case primary == a.ID:
		_, err = database.Exec(ctx, tx, `UPDATE stacks SET primary_asset_id = $1, updated_at = $2, update_id = $3
			WHERE id = $4`,
			remaining[0], now, uuid.Must(uuid.NewV7()), stackID,
		)
		return err
	}

	return nil
}
