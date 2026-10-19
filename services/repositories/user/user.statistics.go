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
	return database.Rows[Statistics](ctx, repo.db, `SELECT u.id AS user_id, u.name AS user_name, u.quota_size_in_bytes,
		COUNT(*) FILTER (WHERE a.type = 'IMAGE' AND a.visibility != 'hidden') AS photos,
		COUNT(*) FILTER (WHERE a.type = 'VIDEO' AND a.visibility != 'hidden') AS videos,
		COALESCE(SUM(e.file_size_in_byte) FILTER (WHERE a.library_id IS NULL), 0) AS usage,
		COALESCE(SUM(e.file_size_in_byte) FILTER (WHERE a.library_id IS NULL AND a.type = 'IMAGE'), 0) AS usage_photos,
		COALESCE(SUM(e.file_size_in_byte) FILTER (WHERE a.library_id IS NULL AND a.type = 'VIDEO'), 0) AS usage_videos
		FROM users u
		LEFT JOIN assets a ON a.owner_id = u.id AND a.deleted_at IS NULL
		LEFT JOIN asset_exif e ON e.asset_id = a.id
		GROUP BY u.id
		ORDER BY u.created_at ASC`,
	)
}
