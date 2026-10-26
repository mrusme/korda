package asset

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Marker struct {
	ID      uuid.UUID
	Lat     float64
	Lon     float64
	City    sql.Null[string]
	State   sql.Null[string]
	Country sql.Null[string]
}

type MarkerFilter struct {
	Caller            uuid.UUID
	OwnerIDs          []uuid.UUID
	WithSharedAlbums  bool
	IsArchived        *bool
	IsFavorite        *bool
	FileCreatedAfter  *time.Time
	FileCreatedBefore *time.Time
}

func (repo *Repository) MapMarkers(ctx context.Context, f MarkerFilter) ([]Marker, error) {
	b := &builder{}
	if f.IsArchived != nil && *f.IsArchived {
		b.where = append(b.where, `(a.visibility = 'timeline' OR (a.owner_id = `+b.arg(f.Caller)+` AND a.visibility = 'archive'))`)
	} else {
		b.where = append(b.where, `a.visibility = 'timeline'`)
	}
	if f.IsFavorite != nil {
		b.where = append(b.where, `a.is_favorite = `+b.arg(*f.IsFavorite))
	}
	if f.FileCreatedAfter != nil {
		b.where = append(b.where, `a.file_created_at >= `+b.arg(f.FileCreatedAfter.UTC()))
	}
	if f.FileCreatedBefore != nil {
		b.where = append(b.where, `a.file_created_at <= `+b.arg(f.FileCreatedBefore.UTC()))
	}

	owners := `a.owner_id IN ` + b.list(anyList(f.OwnerIDs))
	if f.WithSharedAlbums {
		owners = `(` + owners + ` OR EXISTS (SELECT 1 FROM album_assets x JOIN album_users au ON au.album_id = x.album_id
			JOIN albums al ON al.id = x.album_id AND al.deleted_at IS NULL
			WHERE x.asset_id = a.id AND au.user_id = ` + b.arg(f.Caller) + `))`
	}
	b.where = append(b.where, owners)

	return database.Rows[Marker](ctx, repo.db, `SELECT a.id, e.latitude AS lat, e.longitude AS lon, e.city, e.state, e.country
		FROM assets a JOIN asset_exif e ON e.asset_id = a.id AND e.latitude IS NOT NULL AND e.longitude IS NOT NULL
		WHERE a.deleted_at IS NULL AND `+strings.Join(b.where, " AND ")+`
		ORDER BY a.file_created_at DESC, a.rowid`, b.args...)
}
