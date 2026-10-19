package asset

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Statistics struct {
	Audio int64
	Image int64
	Video int64
	Other int64
}

type StatisticsFilter struct {
	Visibility *asset.Visibility
	IsFavorite *bool
	IsTrashed  *bool
}

func (repo *Repository) Statistics(ctx context.Context, ownerID uuid.UUID, f StatisticsFilter) (*Statistics, error) {
	query := `SELECT
		COALESCE(SUM(type = 'AUDIO'), 0) AS audio,
		COALESCE(SUM(type = 'IMAGE'), 0) AS image,
		COALESCE(SUM(type = 'VIDEO'), 0) AS video,
		COALESCE(SUM(type = 'OTHER'), 0) AS other
		FROM assets WHERE owner_id = $1`
	args := []any{ownerID}

	if f.Visibility == nil {
		query += ` AND visibility IN ('archive', 'timeline')`
	} else {
		args = append(args, *f.Visibility)
		query += ` AND visibility = $` + strconv.Itoa(len(args))
	}
	if f.IsFavorite != nil {
		args = append(args, *f.IsFavorite)
		query += ` AND is_favorite = $` + strconv.Itoa(len(args))
	}
	if f.IsTrashed != nil && *f.IsTrashed {
		query += ` AND status != 'deleted' AND deleted_at IS NOT NULL`
	} else {
		query += ` AND deleted_at IS NULL`
	}

	return database.Row[Statistics](ctx, repo.db, query, args...)
}

type DayCount struct {
	Date  string
	Count int64
}

func (repo *Repository) Heatmap(ctx context.Context, ownerID uuid.UUID, taken bool, from time.Time, to time.Time) ([]DayCount, error) {
	column := "created_at"
	if taken {
		column = "local_date_time"
	}

	return database.Rows[DayCount](ctx, repo.db, `SELECT date(`+column+`) AS date, COUNT(*) AS count FROM assets
		WHERE owner_id = $1 AND `+column+` >= $2 AND `+column+` < $3 AND deleted_at IS NULL
		GROUP BY date(`+column+`) ORDER BY date ASC`,
		ownerID, from, to,
	)
}
