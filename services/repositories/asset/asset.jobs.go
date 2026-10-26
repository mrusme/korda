package asset

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Insert(ctx context.Context, a *asset.Asset, usage int64) (*asset.Asset, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	created, err := insert(ctx, tx, a)
	if err != nil {
		return nil, err
	}

	if usage != 0 {
		if _, err = database.Exec(ctx, tx, `UPDATE users
			SET quota_usage_in_bytes = quota_usage_in_bytes + $1, updated_at = $2, update_id = $3
			WHERE id = $4 AND deleted_at IS NULL`,
			usage, a.CreatedAt, uuid.Must(uuid.NewV7()), a.OwnerID,
		); err != nil {
			return nil, err
		}
	}

	return created, tx.Commit(ctx)
}

func (repo *Repository) LivePhotoMatch(ctx context.Context, ownerID uuid.UUID, cid string, otherID uuid.UUID, kind asset.Type) (*asset.Asset, error) {
	return database.Row[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a
		JOIN asset_exif e ON e.asset_id = a.id
		WHERE a.id != $1 AND a.owner_id = $2 AND a.type = $3 AND e.live_photo_cid = $4 LIMIT 1`,
		otherID, ownerID, kind, cid,
	)
}

func (repo *Repository) MetadataExtracted(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO asset_job_status (asset_id, metadata_extracted_at) VALUES ($1, $2)
		ON CONFLICT (asset_id) DO UPDATE SET metadata_extracted_at = excluded.metadata_extracted_at`,
		id, at,
	)

	return err
}

func (repo *Repository) ThumbnailsGenerated(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO asset_job_status (asset_id, thumbnails_generated_at) VALUES ($1, $2)
		ON CONFLICT (asset_id) DO UPDATE SET thumbnails_generated_at = excluded.thumbnails_generated_at`,
		id, at,
	)

	return err
}

func (repo *Repository) MissingThumbnails(ctx context.Context) ([]uuid.UUID, error) {
	return database.Values[uuid.UUID](ctx, repo.db, `SELECT a.id FROM assets a
		WHERE a.visibility != 'hidden' AND a.deleted_at IS NULL AND (
			a.thumbhash IS NULL
			OR NOT EXISTS (SELECT 1 FROM asset_files f WHERE f.asset_id = a.id AND f.type = 'preview' AND f.is_edited = 0)
			OR NOT EXISTS (SELECT 1 FROM asset_files f WHERE f.asset_id = a.id AND f.type = 'thumbnail' AND f.is_edited = 0)
			OR (a.is_edited = 1 AND NOT EXISTS (SELECT 1 FROM asset_files f
				WHERE f.asset_id = a.id AND f.type = 'fullsize' AND f.is_edited = 1)))`)
}

func (repo *Repository) ForThumbnails(ctx context.Context, force bool) ([]uuid.UUID, error) {
	if !force {
		return repo.MissingThumbnails(ctx)
	}

	return database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM assets
		WHERE visibility != 'hidden' AND deleted_at IS NULL`)
}

func (repo *Repository) ForMetadata(ctx context.Context, force bool) ([]uuid.UUID, error) {
	if force {
		return database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM assets WHERE deleted_at IS NULL`)
	}

	return database.Values[uuid.UUID](ctx, repo.db, `SELECT a.id FROM assets a
		LEFT JOIN asset_job_status s ON s.asset_id = a.id
		WHERE a.deleted_at IS NULL AND s.metadata_extracted_at IS NULL`)
}

func (repo *Repository) ForTranscode(ctx context.Context, force bool) ([]uuid.UUID, error) {
	if force {
		return database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM assets
			WHERE type = 'VIDEO' AND deleted_at IS NULL`)
	}

	return database.Values[uuid.UUID](ctx, repo.db, `SELECT a.id FROM assets a
		WHERE a.type = 'VIDEO' AND a.deleted_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM asset_files f WHERE f.asset_id = a.id AND f.type = 'encoded_video')`)
}
