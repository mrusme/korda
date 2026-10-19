package asset

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) ids(ctx context.Context, query string, args ...any) ([]uuid.UUID, error) {
	return database.Values[uuid.UUID](ctx, repo.db, query, args...)
}

func (repo *Repository) EmptyTrash(ctx context.Context, ownerID uuid.UUID, now time.Time) (int64, error) {
	ids, err := repo.ids(ctx, `SELECT id FROM assets WHERE owner_id = $1 AND status = 'trashed'`, ownerID)
	if err != nil {
		return 0, err
	}

	return int64(len(ids)), repo.UpdateAll(ctx, ids, Update{Status: optionalStatus(asset.StatusDeleted)}, now)
}

func (repo *Repository) RestoreTrash(ctx context.Context, ownerID uuid.UUID, now time.Time) (int64, error) {
	ids, err := repo.ids(ctx, `SELECT id FROM assets WHERE owner_id = $1 AND status = 'trashed'`, ownerID)
	if err != nil {
		return 0, err
	}

	return int64(len(ids)), repo.restore(ctx, ids, now)
}

func (repo *Repository) RestoreAll(ctx context.Context, ids []uuid.UUID, now time.Time) error {
	trashed := []uuid.UUID{}
	for _, id := range ids {
		found, err := repo.ids(ctx, `SELECT id FROM assets WHERE id = $1 AND status = 'trashed'`, id)
		if err != nil {
			return err
		}
		trashed = append(trashed, found...)
	}

	return repo.restore(ctx, trashed, now)
}

func (repo *Repository) restore(ctx context.Context, ids []uuid.UUID, now time.Time) error {
	u := Update{Status: optionalStatus(asset.StatusActive)}
	u.DeletedAt.Set, u.DeletedAt.Null = true, true

	return repo.UpdateAll(ctx, ids, u, now)
}

func (repo *Repository) Deleted(ctx context.Context) ([]uuid.UUID, error) {
	return repo.ids(ctx, `SELECT id FROM assets WHERE status = 'deleted'`)
}

type Expired struct {
	ID        uuid.UUID
	IsOffline bool
}

func (repo *Repository) Expired(ctx context.Context, before time.Time) ([]Expired, error) {
	return database.Rows[Expired](ctx, repo.db, `SELECT id, is_offline FROM assets
		WHERE deleted_at IS NOT NULL AND deleted_at <= $1`,
		before,
	)
}

func (repo *Repository) LivePhotoCount(ctx context.Context, motionID uuid.UUID) (int64, error) {
	return database.Value[int64](ctx, repo.db, `SELECT COUNT(*) FROM assets WHERE live_photo_video_id = $1`, motionID)
}
