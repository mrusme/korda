package album

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/models/user"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, album_name, description, album_thumbnail_asset_id, is_activity_enabled, "order", created_at,
	updated_at, deleted_at, update_id`

const prefixed = `al.id, al.album_name, al.description, al.album_thumbnail_asset_id, al.is_activity_enabled,
	al."order", al.created_at, al.updated_at, al.deleted_at, al.update_id`

type Filter struct {
	ID       uuid.NullUUID
	Name     string
	IsOwned  *bool
	IsShared *bool
}

type Member struct {
	AlbumID uuid.UUID
	Role    album.Role
	user.User
}

type Metadata struct {
	AlbumID      uuid.UUID
	StartDate    string
	EndDate      string
	LastModified time.Time
	AssetCount   int64
}

type Counted struct {
	LocalDateTime time.Time
	FileCreatedAt time.Time
}

type Contributor struct {
	UserID     uuid.UUID
	AssetCount int64
}

type Marker struct {
	ID      uuid.UUID
	Lat     float64
	Lon     float64
	City    sql.Null[string]
	State   sql.Null[string]
	Country sql.Null[string]
}

const isShared = `(EXISTS (SELECT 1 FROM album_users o WHERE o.album_id = al.id AND o.role <> 'owner')
	OR EXISTS (SELECT 1 FROM shared_links sl WHERE sl.album_id = al.id))`

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*album.Album, error) {
	return database.Row[album.Album](ctx, repo.db, `SELECT `+columns+` FROM albums WHERE id = $1 AND deleted_at IS NULL`, id)
}

func (repo *Repository) List(ctx context.Context, userID uuid.UUID, f Filter) ([]album.Album, error) {
	args := []any{userID}
	where := ``
	if f.IsOwned != nil {
		if *f.IsOwned {
			where += ` AND au.role = 'owner'`
		} else {
			where += ` AND au.role <> 'owner'`
		}
	}
	if f.IsShared != nil {
		if *f.IsShared {
			where += ` AND ` + isShared
		} else {
			where += ` AND NOT ` + isShared
		}
	}
	if f.ID.Valid {
		args = append(args, f.ID.UUID)
		where += ` AND al.id = $2`
	}
	if f.Name != "" {
		args = append(args, f.Name)
		where += ` AND al.album_name = $` + strconv.Itoa(len(args))
	}

	return database.Rows[album.Album](ctx, repo.db, `SELECT `+prefixed+` FROM albums al
		JOIN album_users au ON au.album_id = al.id AND au.user_id = $1
		WHERE al.deleted_at IS NULL`+where+`
		ORDER BY al.created_at DESC, al.rowid DESC`,
		args...,
	)
}

func (repo *Repository) ByAsset(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) ([]album.Album, error) {
	return database.Rows[album.Album](ctx, repo.db, `SELECT `+prefixed+` FROM albums al
		JOIN album_assets aa ON aa.album_id = al.id
		WHERE EXISTS (SELECT 1 FROM album_users au WHERE au.album_id = al.id AND au.user_id = $1)
			AND aa.asset_id = $2 AND al.deleted_at IS NULL
		ORDER BY al.created_at DESC, al.rowid DESC`,
		userID, assetID,
	)
}

func (repo *Repository) Members(ctx context.Context, albumIDs []uuid.UUID) (map[uuid.UUID][]Member, error) {
	out := map[uuid.UUID][]Member{}
	for chunk := range slices.Chunk(albumIDs, query.ChunkSize) {
		args := []any{}
		marks := query.Placeholders(&args, chunk)
		rows, err := database.Rows[Member](ctx, repo.db, `SELECT au.album_id, au.role, `+userColumns+`
			FROM album_users au JOIN users u ON u.id = au.user_id
			WHERE au.album_id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[r.AlbumID] = append(out[r.AlbumID], r)
		}
	}

	return out, nil
}

const userColumns = `u.id, u.email, u.password, u.pin_code, u.name, u.is_admin, u.should_change_password, u.avatar_color,
	u.profile_image_path, u.profile_changed_at, u.storage_label, u.quota_size_in_bytes, u.quota_usage_in_bytes, u.status,
	u.oauth_id, u.cluster_group_id, u.created_at, u.updated_at, u.deleted_at, u.update_id`

func (repo *Repository) WithSharedLinks(ctx context.Context, albumIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	for chunk := range slices.Chunk(albumIDs, query.ChunkSize) {
		args := []any{}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT DISTINCT album_id FROM shared_links
			WHERE album_id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		for _, id := range found {
			out[id] = true
		}
	}

	return out, nil
}

const counted = `a.deleted_at IS NULL AND a.visibility IN ('archive', 'timeline')`

func (repo *Repository) Metadata(ctx context.Context, albumIDs []uuid.UUID) (map[uuid.UUID]Metadata, error) {
	out := map[uuid.UUID]Metadata{}
	for chunk := range slices.Chunk(albumIDs, query.ChunkSize) {
		args := []any{}
		marks := query.Placeholders(&args, chunk)
		type raw struct {
			AlbumID      uuid.UUID
			StartDate    string
			EndDate      string
			LastModified string
			AssetCount   int64
		}
		rows, err := database.Rows[raw](ctx, repo.db, `SELECT aa.album_id,
			min(date(a.local_date_time)) AS start_date, max(date(a.local_date_time)) AS end_date,
			max(a.updated_at) AS last_modified, count(a.id) AS asset_count
			FROM assets a JOIN album_assets aa ON aa.asset_id = a.id
			WHERE `+counted+` AND aa.album_id IN (`+marks+`)
			GROUP BY aa.album_id`, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			modified, err := time.Parse(timeLayout, r.LastModified)
			if err != nil {
				return nil, err
			}
			out[r.AlbumID] = Metadata{AlbumID: r.AlbumID, StartDate: r.StartDate, EndDate: r.EndDate,
				LastModified: modified.UTC(), AssetCount: r.AssetCount}
		}
	}

	return out, nil
}

const timeLayout = "2006-01-02 15:04:05.999999999-07:00"

func (repo *Repository) CountedAssets(ctx context.Context, albumID uuid.UUID) ([]Counted, error) {
	return database.Rows[Counted](ctx, repo.db, `SELECT a.local_date_time, a.file_created_at
		FROM assets a JOIN album_assets aa ON aa.asset_id = a.id
		WHERE aa.album_id = $1 AND `+counted+`
		ORDER BY a.file_created_at DESC, aa.rowid`, albumID)
}

func (repo *Repository) Contributors(ctx context.Context, albumID uuid.UUID) ([]Contributor, error) {
	return database.Rows[Contributor](ctx, repo.db, `SELECT a.owner_id AS user_id, count(*) AS asset_count
		FROM album_assets aa JOIN assets a ON a.id = aa.asset_id
		WHERE a.deleted_at IS NULL AND aa.album_id = $1
		GROUP BY a.owner_id
		ORDER BY asset_count DESC, min(aa.rowid)`, albumID)
}

func (repo *Repository) AssetIDs(ctx context.Context, albumID uuid.UUID, assetIDs []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{albumID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT asset_id FROM album_assets
			WHERE album_id = $1 AND asset_id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) MapMarkers(ctx context.Context, albumID uuid.UUID) ([]Marker, error) {
	return database.Rows[Marker](ctx, repo.db, `SELECT a.id, e.latitude AS lat, e.longitude AS lon, e.city, e.state,
		e.country
		FROM assets a
		JOIN asset_exif e ON e.asset_id = a.id AND e.latitude IS NOT NULL AND e.longitude IS NOT NULL
		JOIN album_assets aa ON aa.asset_id = a.id
		WHERE aa.album_id = $1 AND a.deleted_at IS NULL
		ORDER BY a.file_created_at DESC, a.rowid`, albumID)
}
