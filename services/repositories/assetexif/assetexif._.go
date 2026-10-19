package assetexif

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/assetexif"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `asset_id, make, model, exif_image_width, exif_image_height, file_size_in_byte, orientation,
	date_time_original, modify_date, lens_model, f_number, focal_length, iso, latitude, longitude,
	city, state, country, description, fps, exposure_time, live_photo_cid, time_zone, projection_type,
	profile_description, colorspace, bits_per_sample, auto_stack_id, rating, tags, locked_properties,
	updated_at, update_id`

type Repository struct {
	db *database.Database
}

func New(db *database.Database) (*Repository, error) {
	return &Repository{db: db}, nil
}

func (repo *Repository) Startup() error {
	return nil
}

func (repo *Repository) Shutdown() error {
	return nil
}

func (repo *Repository) Get(ctx context.Context, assetID uuid.UUID) (*assetexif.Exif, error) {
	return database.Row[assetexif.Exif](ctx, repo.db, `SELECT `+columns+` FROM asset_exif WHERE asset_id = $1`, assetID)
}

func (repo *Repository) Upsert(ctx context.Context, e *assetexif.Exif) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO asset_exif (`+columns+`) VALUES (
		 $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		,$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		,$21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		,$31, $32, $33)
		ON CONFLICT (asset_id) DO UPDATE SET
			make = excluded.make, model = excluded.model,
			exif_image_width = excluded.exif_image_width, exif_image_height = excluded.exif_image_height,
			file_size_in_byte = excluded.file_size_in_byte, orientation = excluded.orientation,
			date_time_original = excluded.date_time_original, modify_date = excluded.modify_date,
			lens_model = excluded.lens_model, f_number = excluded.f_number, focal_length = excluded.focal_length,
			iso = excluded.iso, latitude = excluded.latitude, longitude = excluded.longitude,
			city = excluded.city, state = excluded.state, country = excluded.country,
			description = excluded.description, fps = excluded.fps, exposure_time = excluded.exposure_time,
			live_photo_cid = excluded.live_photo_cid, time_zone = excluded.time_zone,
			projection_type = excluded.projection_type, profile_description = excluded.profile_description,
			colorspace = excluded.colorspace, bits_per_sample = excluded.bits_per_sample,
			auto_stack_id = excluded.auto_stack_id, rating = excluded.rating, tags = excluded.tags,
			locked_properties = excluded.locked_properties,
			updated_at = excluded.updated_at, update_id = excluded.update_id`,
		e.AssetID, e.Make, e.Model, e.ExifImageWidth, e.ExifImageHeight, e.FileSizeInByte, e.Orientation,
		e.DateTimeOriginal, e.ModifyDate, e.LensModel, e.FNumber, e.FocalLength, e.ISO, e.Latitude, e.Longitude,
		e.City, e.State, e.Country, e.Description, e.FPS, e.ExposureTime, e.LivePhotoCID, e.TimeZone, e.ProjectionType,
		e.ProfileDescription, e.Colorspace, e.BitsPerSample, e.AutoStackID, e.Rating, e.Tags, e.LockedProperties,
		e.UpdatedAt, e.UpdateID,
	)

	return err
}

func (repo *Repository) ByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID]*assetexif.Exif, error) {
	out := make(map[uuid.UUID]*assetexif.Exif, len(assetIDs))
	for chunk := range slices.Chunk(assetIDs, 1000) {
		args := make([]any, 0, len(chunk))
		marks := make([]string, 0, len(chunk))
		for _, id := range chunk {
			args = append(args, id)
			marks = append(marks, "$"+strconv.Itoa(len(args)))
		}

		rows, err := database.Rows[assetexif.Exif](ctx, repo.db, `SELECT `+columns+` FROM asset_exif
			WHERE asset_id IN (`+strings.Join(marks, ", ")+`)`, args...)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			out[rows[i].AssetID] = &rows[i]
		}
	}

	return out, nil
}
