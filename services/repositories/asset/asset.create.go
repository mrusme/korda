package asset

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/assetmetadata"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Upload struct {
	Asset          *asset.Asset
	FileSizeInByte int64
	Metadata       []assetmetadata.Item
	Sidecar        string
}

func insert(ctx context.Context, ex database.Executor, a *asset.Asset) (*asset.Asset, error) {
	return database.Row[asset.Asset](ctx, ex, `INSERT INTO assets (`+columns+`) VALUES (
		 $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		,$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		,$21, $22, $23, $24, $25, $26, $27, $28
		) RETURNING `+columns,
		a.ID, a.OwnerID, a.Type, a.OriginalPath, a.OriginalFileName, a.Checksum, a.ChecksumAlgorithm,
		a.FileCreatedAt, a.FileModifiedAt, a.LocalDateTime, a.IsFavorite, a.Duration, a.LivePhotoVideoID,
		a.Thumbhash, a.Visibility, a.Status, a.DeletedAt, a.StackID, a.LibraryID, a.IsOffline, a.IsExternal,
		a.IsEdited, a.DuplicateID, a.Width, a.Height, a.CreatedAt, a.UpdatedAt, a.UpdateID,
	)
}

func (repo *Repository) Create(ctx context.Context, up Upload) (*asset.Asset, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	created, err := insert(ctx, tx, up.Asset)
	if err != nil {
		return nil, err
	}

	now := up.Asset.CreatedAt
	if _, err = database.Exec(ctx, tx, `INSERT INTO asset_exif (asset_id, file_size_in_byte, updated_at, update_id)
		VALUES ($1, $2, $3, $4)`,
		created.ID, up.FileSizeInByte, now, uuid.Must(uuid.NewV7()),
	); err != nil {
		return nil, err
	}

	for _, item := range up.Metadata {
		if _, err = database.Exec(ctx, tx, `INSERT INTO asset_metadata (asset_id, key, value, updated_at, update_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (asset_id, key) DO UPDATE SET value = excluded.value,
				updated_at = excluded.updated_at, update_id = excluded.update_id`,
			created.ID, item.Key, item.Value, now, uuid.Must(uuid.NewV7()),
		); err != nil {
			return nil, err
		}
	}

	if up.Sidecar != "" {
		if err = upsertFile(ctx, tx, created.ID, assetfile.TypeSidecar, up.Sidecar, false, now); err != nil {
			return nil, err
		}
	}

	if _, err = database.Exec(ctx, tx, `UPDATE users
		SET quota_usage_in_bytes = quota_usage_in_bytes + $1, updated_at = $2, update_id = $3
		WHERE id = $4 AND deleted_at IS NULL`,
		up.FileSizeInByte, now, uuid.Must(uuid.NewV7()), created.OwnerID,
	); err != nil {
		return nil, err
	}

	return created, tx.Commit(ctx)
}

func upsertFile(ctx context.Context, ex database.Executor, assetID uuid.UUID, kind assetfile.Type, path string, edited bool, now time.Time) error {
	_, err := database.Exec(ctx, ex, `INSERT INTO asset_files
		(id, asset_id, type, path, is_edited, is_progressive, is_transparent, created_at, updated_at, update_id)
		VALUES ($1, $2, $3, $4, $5, 0, 0, $6, $6, $7)
		ON CONFLICT (asset_id, type, is_edited) DO UPDATE SET path = excluded.path,
			updated_at = excluded.updated_at, update_id = excluded.update_id`,
		uuid.New(), assetID, kind, path, edited, now, uuid.Must(uuid.NewV7()),
	)

	return err
}
