package sync

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Asset struct {
	ID               uuid.UUID
	OwnerID          uuid.UUID
	OriginalFileName string
	Thumbhash        []byte
	Checksum         []byte
	FileCreatedAt    time.Time
	FileModifiedAt   time.Time
	CreatedAt        time.Time
	LocalDateTime    time.Time
	Type             string
	DeletedAt        sql.Null[time.Time]
	IsFavorite       bool
	Visibility       string
	Duration         sql.Null[int64]
	LivePhotoVideoID uuid.NullUUID
	StackID          uuid.NullUUID
	LibraryID        uuid.NullUUID
	Width            sql.Null[int64]
	Height           sql.Null[int64]
	IsEdited         bool
	UpdateID         string
}

type AssetDelete struct {
	ID      string
	AssetID uuid.UUID
}

func (repo *Repository) Assets(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]Asset, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, ownerID)

	return database.Rows[Asset](ctx, repo.db, `SELECT id, owner_id, original_file_name, thumbhash, checksum,
		file_created_at, file_modified_at, created_at, local_date_time, type, deleted_at, is_favorite, visibility,
		duration, live_photo_video_id, stack_id, library_id, width, height, is_edited, update_id
		FROM assets WHERE `+where+` AND owner_id = $`+strconv.Itoa(len(args))+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) AssetDeletes(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]AssetDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetDelete](ctx, repo.db, `SELECT id, asset_id FROM asset_audit
		WHERE `+where+` AND owner_id = $`+strconv.Itoa(len(args))+b.tail("id"),
		args...,
	)
}

type AssetExif struct {
	AssetID            uuid.UUID
	Description        string
	ExifImageWidth     sql.Null[int64]
	ExifImageHeight    sql.Null[int64]
	FileSizeInByte     sql.Null[int64]
	Orientation        sql.Null[string]
	DateTimeOriginal   sql.Null[time.Time]
	ModifyDate         sql.Null[time.Time]
	TimeZone           sql.Null[string]
	Latitude           sql.Null[float64]
	Longitude          sql.Null[float64]
	ProjectionType     sql.Null[string]
	City               sql.Null[string]
	State              sql.Null[string]
	Country            sql.Null[string]
	Make               sql.Null[string]
	Model              sql.Null[string]
	LensModel          sql.Null[string]
	FNumber            sql.Null[float64]
	FocalLength        sql.Null[float64]
	Iso                sql.Null[int64]
	ExposureTime       sql.Null[string]
	ProfileDescription sql.Null[string]
	Rating             sql.Null[int64]
	Fps                sql.Null[float64]
	UpdateID           string
}

const exifColumns = `e.asset_id, e.description, e.exif_image_width, e.exif_image_height, e.file_size_in_byte,
	e.orientation, e.date_time_original, e.modify_date, e.time_zone, e.latitude, e.longitude, e.projection_type,
	e.city, e.state, e.country, e.make, e.model, e.lens_model, e.f_number, e.focal_length, e.iso, e.exposure_time,
	e.profile_description, e.rating, e.fps, e.update_id`

func (repo *Repository) AssetExifs(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]AssetExif, error) {
	args := []any{}
	where := b.where("e.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetExif](ctx, repo.db, `SELECT `+exifColumns+` FROM asset_exif e
		WHERE `+where+` AND e.asset_id IN (SELECT id FROM assets WHERE owner_id = $`+strconv.Itoa(len(args))+`)`+
		b.tail("e.update_id"),
		args...,
	)
}

type AssetMetadata struct {
	AssetID  uuid.UUID
	Key      string
	Value    string
	UpdateID string
}

type AssetMetadataDelete struct {
	ID      string
	AssetID uuid.UUID
	Key     string
}

func (repo *Repository) AssetMetadata(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]AssetMetadata, error) {
	args := []any{}
	where := b.where("m.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetMetadata](ctx, repo.db, `SELECT m.asset_id, m.key, m.value, m.update_id
		FROM asset_metadata m JOIN assets a ON a.id = m.asset_id
		WHERE `+where+` AND a.owner_id = $`+strconv.Itoa(len(args))+b.tail("m.update_id"),
		args...,
	)
}

func (repo *Repository) AssetMetadataDeletes(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]AssetMetadataDelete, error) {
	args := []any{}
	where := b.where("d.id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetMetadataDelete](ctx, repo.db, `SELECT d.id, d.asset_id, d.key
		FROM asset_metadata_audit d LEFT JOIN assets a ON a.id = d.asset_id
		WHERE `+where+` AND a.owner_id = $`+strconv.Itoa(len(args))+b.tail("d.id"),
		args...,
	)
}

type AssetEdit struct {
	ID         uuid.UUID
	AssetID    uuid.UUID
	Sequence   int64
	Action     string
	Parameters string
	UpdateID   string
}

type AssetEditDelete struct {
	ID     string
	EditID uuid.UUID
}

func (repo *Repository) AssetEdits(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]AssetEdit, error) {
	args := []any{}
	where := b.where("x.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetEdit](ctx, repo.db, `SELECT x.id, x.asset_id, x.sequence, x.action, x.parameters, x.update_id
		FROM asset_edits x JOIN assets a ON a.id = x.asset_id
		WHERE `+where+` AND a.owner_id = $`+strconv.Itoa(len(args))+b.tail("x.update_id"),
		args...,
	)
}

func (repo *Repository) AssetEditDeletes(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]AssetEditDelete, error) {
	args := []any{}
	where := b.where("d.id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetEditDelete](ctx, repo.db, `SELECT d.id, d.edit_id
		FROM asset_edit_audit d JOIN assets a ON a.id = d.asset_id
		WHERE `+where+` AND a.owner_id = $`+strconv.Itoa(len(args))+b.tail("d.id"),
		args...,
	)
}

var auditTables = []string{"album_audit", "album_user_audit", "album_asset_audit", "asset_audit",
	"asset_edit_audit", "asset_metadata_audit", "memory_audit", "memory_asset_audit", "partner_audit", "stack_audit", "user_audit",
	"user_metadata_audit"}

func (repo *Repository) CleanupAudit(ctx context.Context, before time.Time) (int64, error) {
	var total int64
	for _, table := range auditTables {
		res, err := database.Exec(ctx, repo.db, `DELETE FROM `+table+` WHERE deleted_at < $1`, before)
		if err != nil {
			return total, err
		}
		total += res.RowsAffected()
	}

	return total, nil
}
