package asset

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/geo"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Timeline struct {
	Caller      uuid.UUID
	UserIDs     []uuid.UUID
	AlbumID     uuid.NullUUID
	TagID       uuid.NullUUID
	IsFavorite  *bool
	IsTrashed   bool
	WithStacked bool
	Visibility  *asset.Visibility
	ByCreatedAt bool
	Ascending   bool
	BBox        *geo.BBox
}

type TimeBucket struct {
	TimeBucket string
	Count      int64
}

type TimelineAsset struct {
	asset.Asset
	ProjectionType sql.Null[string]
	City           sql.Null[string]
	Country        sql.Null[string]
	Latitude       sql.Null[float64]
	Longitude      sql.Null[float64]
	StackCount     int64
}

func (t *Timeline) column() string {
	if t.ByCreatedAt {
		return "a.created_at"
	}

	return "a.local_date_time"
}

func (t *Timeline) build(b *builder) {
	if t.IsTrashed {
		b.where = append(b.where, `a.status != 'deleted'`, `a.deleted_at IS NOT NULL`)
	} else {
		b.where = append(b.where, `a.deleted_at IS NULL`)
	}

	if box := t.BBox; box != nil {
		b.innerExif()
		b.where = append(b.where, `e.latitude >= `+b.arg(box.South), `e.latitude <= `+b.arg(box.North))
		if box.West <= box.East {
			b.where = append(b.where, `e.longitude >= `+b.arg(box.West), `e.longitude <= `+b.arg(box.East))
		} else {
			b.where = append(b.where, `(e.longitude >= `+b.arg(box.West)+` OR e.longitude <= `+b.arg(box.East)+`)`)
		}
	}

	switch {
	case t.Visibility != nil:
		b.where = append(b.where, `a.visibility = `+b.arg(*t.Visibility))
	case t.UserIDs != nil:
		b.where = append(b.where, `(a.visibility = 'timeline' OR (a.visibility = 'archive' AND a.owner_id = `+b.arg(t.Caller)+`))`)
	default:
		b.where = append(b.where, `a.visibility IN ('archive', 'timeline')`)
	}

	if t.AlbumID.Valid {
		b.where = append(b.where, `EXISTS (SELECT 1 FROM album_assets x WHERE x.asset_id = a.id AND x.album_id = `+b.arg(t.AlbumID.UUID)+`)`)
	}

	if t.UserIDs != nil {
		b.where = append(b.where, `a.owner_id IN `+b.list(anyList(t.UserIDs)))
	}

	if t.IsFavorite != nil {
		b.where = append(b.where, `a.is_favorite = `+b.arg(*t.IsFavorite))
	}

	if t.TagID.Valid {
		b.where = append(b.where, `a.id IN (`+tagClosure(b, []uuid.UUID{t.TagID.UUID})+`
			SELECT ta.asset_id FROM tag_assets ta JOIN closure c ON ta.tag_id = c.descendant)`)
	}
}

func (repo *Repository) TimeBuckets(ctx context.Context, t *Timeline) ([]TimeBucket, error) {
	b := &builder{}
	t.build(b)
	if t.WithStacked {
		b.where = append(b.where, `(a.stack_id IS NULL OR EXISTS (SELECT 1 FROM stacks s WHERE s.id = a.stack_id AND s.primary_asset_id = a.id))`)
	}

	direction := "DESC"
	if t.Ascending {
		direction = "ASC"
	}

	return database.Rows[TimeBucket](ctx, repo.db, `SELECT strftime('%Y-%m-01', `+t.column()+`) AS time_bucket, count(*) AS count
		FROM assets a `+b.sql()+` GROUP BY time_bucket ORDER BY time_bucket `+direction, b.args...)
}

func (repo *Repository) TimeBucket(ctx context.Context, t *Timeline, month string) ([]TimelineAsset, error) {
	b := &builder{}
	b.innerExif()
	t.build(b)
	b.where = append(b.where, `strftime('%Y-%m', `+t.column()+`) = `+b.arg(month))
	if t.WithStacked {
		b.where = append(b.where, `NOT EXISTS (SELECT 1 FROM stacks s WHERE s.id = a.stack_id AND s.primary_asset_id != a.id)`)
	}

	return database.Rows[TimelineAsset](ctx, repo.db, `SELECT `+prefixed("a")+`,
			e.projection_type, e.city, e.country, e.latitude, e.longitude,
			(SELECT count(*) FROM assets m WHERE a.stack_id IS NOT NULL AND m.stack_id = a.stack_id
				AND m.deleted_at IS NULL AND m.visibility = 'timeline') AS stack_count
		FROM assets a `+b.sql(), b.args...)
}
