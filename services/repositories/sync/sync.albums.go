package sync

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Album struct {
	ID                uuid.UUID
	Name              string
	Description       sql.Null[string]
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ThumbnailAssetID  uuid.NullUUID
	IsActivityEnabled bool
	Order             string
	UpdateID          string
}

type AlbumV1 struct {
	Album
	OwnerID uuid.UUID
}

type AlbumDelete struct {
	ID      string
	AlbumID uuid.UUID
}

type AlbumUser struct {
	AlbumID  uuid.UUID
	UserID   uuid.UUID
	Role     string
	UpdateID string
}

type AlbumUserDelete struct {
	ID      string
	UserID  uuid.UUID
	AlbumID uuid.UUID
}

type AlbumToAsset struct {
	AssetID  uuid.UUID
	AlbumID  uuid.UUID
	UpdateID string
}

type AlbumToAssetDelete struct {
	ID      string
	AssetID uuid.UUID
	AlbumID uuid.UUID
}

type AlbumAsset Asset

type AlbumAssetUpdate struct {
	AlbumAsset
	PageAlbumID uuid.UUID
}

type AlbumAssetExifUpdate struct {
	AssetExif
	PageAlbumID uuid.UUID
}

type Cursor struct {
	UpdateID string
	AlbumID  uuid.UUID
}

const memberOf = `SELECT album_id FROM album_users WHERE user_id = $`

const albumAssetColumns = `a.id, a.owner_id, a.original_file_name, a.thumbhash, a.checksum, a.file_created_at,
	a.file_modified_at, a.created_at, a.local_date_time, a.type, a.deleted_at, a.visibility, a.duration,
	a.live_photo_video_id, a.stack_id, a.library_id, a.width, a.height, a.is_edited`

func favorite(args *[]any, userID uuid.UUID) string {
	*args = append(*args, userID)

	return `CASE WHEN a.owner_id = $` + strconv.Itoa(len(*args)) + ` THEN a.is_favorite ELSE 0 END AS is_favorite`
}

func (repo *Repository) AlbumsCreatedAfter(ctx context.Context, userID uuid.UUID, after *string, before string) ([]Membership, error) {
	args := []any{userID, before}
	query := `SELECT album_id AS id, create_id FROM album_users WHERE user_id = $1 AND create_id < $2`
	if after != nil {
		args = append(args, *after)
		query += ` AND create_id >= $3`
	}

	return database.Rows[Membership](ctx, repo.db, query+` ORDER BY create_id`, args...)
}

func (repo *Repository) Albums(ctx context.Context, userID uuid.UUID, b Bounds) ([]Album, error) {
	args := []any{}
	where := b.where("al.update_id", &args)
	args = append(args, userID)

	return database.Rows[Album](ctx, repo.db, `SELECT al.id, al.album_name AS name, al.description, al.created_at,
		al.updated_at, al.album_thumbnail_asset_id AS thumbnail_asset_id, al.is_activity_enabled, al."order",
		al.update_id
		FROM albums al JOIN album_users au ON au.album_id = al.id
		WHERE `+where+` AND au.user_id = $`+strconv.Itoa(len(args))+b.tail("al.update_id"),
		args...,
	)
}

func (repo *Repository) AlbumOwner(ctx context.Context, albumID uuid.UUID) (uuid.UUID, error) {
	return database.Value[uuid.UUID](ctx, repo.db, `SELECT user_id FROM album_users
		WHERE album_id = $1 AND role = 'owner'`, albumID)
}

func (repo *Repository) AlbumDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]AlbumDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)

	return database.Rows[AlbumDelete](ctx, repo.db, `SELECT id, album_id FROM album_audit
		WHERE `+where+` AND user_id = $`+strconv.Itoa(len(args))+b.tail("id"),
		args...,
	)
}

func (repo *Repository) AlbumUsers(ctx context.Context, userID uuid.UUID, b Bounds) ([]AlbumUser, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, userID)

	return database.Rows[AlbumUser](ctx, repo.db, `SELECT album_id, user_id, role, update_id FROM album_users
		WHERE `+where+` AND album_id IN (`+memberOf+strconv.Itoa(len(args))+`)`+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) AlbumUsersBackfill(ctx context.Context, albumID uuid.UUID, b Backfill) ([]AlbumUser, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, albumID)

	return database.Rows[AlbumUser](ctx, repo.db, `SELECT album_id, user_id, role, update_id FROM album_users
		WHERE `+where+` AND album_id = $`+strconv.Itoa(len(args))+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) AlbumUserDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]AlbumUserDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)

	return database.Rows[AlbumUserDelete](ctx, repo.db, `SELECT id, user_id, album_id FROM album_user_audit
		WHERE `+where+` AND album_id IN (`+memberOf+strconv.Itoa(len(args))+`)`+b.tail("id"),
		args...,
	)
}

func (repo *Repository) AlbumToAssets(ctx context.Context, userID uuid.UUID, b Bounds) ([]AlbumToAsset, error) {
	args := []any{}
	where := b.where("aa.update_id", &args)
	args = append(args, userID)

	return database.Rows[AlbumToAsset](ctx, repo.db, `SELECT aa.asset_id, aa.album_id, aa.update_id
		FROM album_assets aa JOIN album_users au ON au.album_id = aa.album_id
		WHERE `+where+` AND au.user_id = $`+strconv.Itoa(len(args))+b.tail("aa.update_id"),
		args...,
	)
}

func (repo *Repository) AlbumToAssetsBackfill(ctx context.Context, albumID uuid.UUID, b Backfill) ([]AlbumToAsset, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, albumID)

	return database.Rows[AlbumToAsset](ctx, repo.db, `SELECT asset_id, album_id, update_id FROM album_assets
		WHERE `+where+` AND album_id = $`+strconv.Itoa(len(args))+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) AlbumToAssetDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]AlbumToAssetDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)

	return database.Rows[AlbumToAssetDelete](ctx, repo.db, `SELECT id, asset_id, album_id FROM album_asset_audit
		WHERE `+where+` AND album_id IN (`+memberOf+strconv.Itoa(len(args))+`)`+b.tail("id"),
		args...,
	)
}

func (repo *Repository) AlbumAssetsBackfill(ctx context.Context, userID uuid.UUID, albumID uuid.UUID, b Backfill) ([]AlbumAsset, error) {
	args := []any{}
	fav := favorite(&args, userID)
	where := b.where("aa.update_id", &args)
	args = append(args, albumID)

	return database.Rows[AlbumAsset](ctx, repo.db, `SELECT `+albumAssetColumns+`, `+fav+`, aa.update_id
		FROM album_assets aa JOIN assets a ON a.id = aa.asset_id
		WHERE `+where+` AND aa.album_id = $`+strconv.Itoa(len(args))+b.tail("aa.update_id"),
		args...,
	)
}

func pageAfter(c *Cursor, column string, args *[]any) string {
	if c == nil {
		return ""
	}
	*args = append(*args, c.UpdateID, c.AlbumID)
	n := len(*args)

	return ` AND (` + column + ` > $` + strconv.Itoa(n-1) + ` OR (` + column + ` = $` + strconv.Itoa(n-1) +
		` AND aa.album_id > $` + strconv.Itoa(n) + `))`
}

func (repo *Repository) AlbumAssetUpdates(ctx context.Context, userID uuid.UUID, created string, b Bounds, c *Cursor) ([]AlbumAssetUpdate, error) {
	args := []any{}
	fav := favorite(&args, userID)
	where := b.where("a.update_id", &args)
	args = append(args, created, userID)
	n := len(args)
	page := pageAfter(c, "a.update_id", &args)

	return database.Rows[AlbumAssetUpdate](ctx, repo.db, `SELECT `+albumAssetColumns+`, `+fav+`, a.update_id,
		aa.album_id AS page_album_id
		FROM assets a JOIN album_assets aa ON aa.asset_id = a.id JOIN album_users au ON au.album_id = aa.album_id
		WHERE `+where+` AND aa.update_id <= $`+strconv.Itoa(n-1)+` AND au.user_id = $`+strconv.Itoa(n)+page+
		` ORDER BY a.update_id, aa.album_id LIMIT `+strconv.Itoa(b.Limit),
		args...,
	)
}

func (repo *Repository) AlbumAssetCreates(ctx context.Context, userID uuid.UUID, b Bounds) ([]AlbumAsset, error) {
	args := []any{}
	fav := favorite(&args, userID)
	where := b.where("aa.update_id", &args)
	args = append(args, userID)

	return database.Rows[AlbumAsset](ctx, repo.db, `SELECT `+albumAssetColumns+`, `+fav+`, aa.update_id
		FROM album_assets aa JOIN assets a ON a.id = aa.asset_id JOIN album_users au ON au.album_id = aa.album_id
		WHERE `+where+` AND au.user_id = $`+strconv.Itoa(len(args))+b.tail("aa.update_id"),
		args...,
	)
}

const albumExifColumns = `e.asset_id, e.description, e.exif_image_width, e.exif_image_height, e.file_size_in_byte,
	e.orientation, e.date_time_original, e.modify_date, e.time_zone, e.latitude, e.longitude, e.projection_type,
	e.city, e.state, e.country, e.make, e.model, e.lens_model, e.f_number, e.focal_length, e.iso, e.exposure_time,
	e.profile_description, e.rating, e.fps`

func (repo *Repository) AlbumAssetExifsBackfill(ctx context.Context, albumID uuid.UUID, b Backfill) ([]AssetExif, error) {
	args := []any{}
	where := b.where("aa.update_id", &args)
	args = append(args, albumID)

	return database.Rows[AssetExif](ctx, repo.db, `SELECT `+albumExifColumns+`, aa.update_id
		FROM album_assets aa JOIN asset_exif e ON e.asset_id = aa.asset_id
		WHERE `+where+` AND aa.album_id = $`+strconv.Itoa(len(args))+b.tail("aa.update_id"),
		args...,
	)
}

func (repo *Repository) AlbumAssetExifUpdates(ctx context.Context, userID uuid.UUID, created string, b Bounds, c *Cursor) ([]AlbumAssetExifUpdate, error) {
	args := []any{}
	where := b.where("e.update_id", &args)
	args = append(args, created, userID)
	n := len(args)
	page := pageAfter(c, "e.update_id", &args)

	return database.Rows[AlbumAssetExifUpdate](ctx, repo.db, `SELECT `+albumExifColumns+`, e.update_id,
		aa.album_id AS page_album_id
		FROM asset_exif e JOIN album_assets aa ON aa.asset_id = e.asset_id JOIN album_users au ON au.album_id = aa.album_id
		WHERE `+where+` AND aa.update_id <= $`+strconv.Itoa(n-1)+` AND au.user_id = $`+strconv.Itoa(n)+page+
		` ORDER BY e.update_id, aa.album_id LIMIT `+strconv.Itoa(b.Limit),
		args...,
	)
}

func (repo *Repository) AlbumAssetExifCreates(ctx context.Context, userID uuid.UUID, b Bounds) ([]AssetExif, error) {
	args := []any{}
	where := b.where("aa.update_id", &args)
	args = append(args, userID)

	return database.Rows[AssetExif](ctx, repo.db, `SELECT `+albumExifColumns+`, aa.update_id
		FROM album_assets aa JOIN asset_exif e ON e.asset_id = aa.asset_id JOIN albums al ON al.id = aa.album_id
		JOIN album_users au ON au.album_id = aa.album_id
		WHERE `+where+` AND au.user_id = $`+strconv.Itoa(len(args))+b.tail("aa.update_id"),
		args...,
	)
}
