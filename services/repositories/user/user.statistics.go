package user

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Statistics struct {
	UserID           uuid.UUID
	UserName         string
	QuotaSizeInBytes sql.Null[int64]
	Photos           int64
	Videos           int64
	Usage            int64
	UsagePhotos      int64
	UsageVideos      int64
}

func (repo *Repository) Statistics(ctx context.Context) ([]Statistics, error) {
	return database.Rows[Statistics](ctx, repo.db, `SELECT id AS user_id, name AS user_name,
		quota_size_in_bytes, 0 AS photos, 0 AS videos, 0 AS usage,
		0 AS usage_photos, 0 AS usage_videos
		FROM users
		ORDER BY created_at ASC`,
	)
}
