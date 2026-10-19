package assetexif

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Exif struct {
	AssetID            uuid.UUID
	Make               sql.Null[string]
	Model              sql.Null[string]
	ExifImageWidth     sql.Null[int64]
	ExifImageHeight    sql.Null[int64]
	FileSizeInByte     sql.Null[int64]
	Orientation        sql.Null[string]
	DateTimeOriginal   sql.Null[time.Time]
	ModifyDate         sql.Null[time.Time]
	LensModel          sql.Null[string]
	FNumber            sql.Null[float64]
	FocalLength        sql.Null[float64]
	ISO                sql.Null[int64]
	Latitude           sql.Null[float64]
	Longitude          sql.Null[float64]
	City               sql.Null[string]
	State              sql.Null[string]
	Country            sql.Null[string]
	Description        string
	FPS                sql.Null[float64]
	ExposureTime       sql.Null[string]
	LivePhotoCID       sql.Null[string]
	TimeZone           sql.Null[string]
	ProjectionType     sql.Null[string]
	ProfileDescription sql.Null[string]
	Colorspace         sql.Null[string]
	BitsPerSample      sql.Null[int64]
	AutoStackID        sql.Null[string]
	Rating             sql.Null[int64]
	Tags               sql.Null[string]
	LockedProperties   sql.Null[string]
	UpdatedAt          time.Time
	UpdateID           uuid.UUID
}
