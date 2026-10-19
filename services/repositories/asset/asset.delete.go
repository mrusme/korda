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
