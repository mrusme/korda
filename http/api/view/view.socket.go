package view

import (
	"encoding/base64"
	"encoding/json"

	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetedit"
	"tty.fail/mrus/korda/models/assetexif"
)

type UploadReadyAsset struct {
	ID               string         `json:"id"`
	OwnerID          string         `json:"ownerId"`
	OriginalFileName string         `json:"originalFileName"`
	Thumbhash        *string        `json:"thumbhash"`
	Checksum         string         `json:"checksum"`
	FileCreatedAt    jsontime.Time  `json:"fileCreatedAt"`
	FileModifiedAt   jsontime.Time  `json:"fileModifiedAt"`
	CreatedAt        jsontime.Time  `json:"createdAt"`
	LocalDateTime    jsontime.Time  `json:"localDateTime"`
	Duration         *int64         `json:"duration"`
	Type             asset.Type     `json:"type"`
	DeletedAt        *jsontime.Time `json:"deletedAt"`
	IsFavorite       bool           `json:"isFavorite"`
	Visibility       string         `json:"visibility"`
	LivePhotoVideoID *string        `json:"livePhotoVideoId"`
	StackID          *string        `json:"stackId"`
	LibraryID        *string        `json:"libraryId"`
	Width            *int64         `json:"width"`
	Height           *int64         `json:"height"`
	IsEdited         bool           `json:"isEdited"`
}

type UploadReady struct {
	Asset UploadReadyAsset `json:"asset"`
	Exif  SyncAssetExifV1  `json:"exif"`
}

func NewUploadReadyAsset(a *asset.Asset) UploadReadyAsset {
	return UploadReadyAsset{
		ID:               id(a.ID),
		OwnerID:          id(a.OwnerID),
		OriginalFileName: a.OriginalFileName,
		Thumbhash:        base64Bytes(a.Thumbhash),
		Checksum:         base64.StdEncoding.EncodeToString(a.Checksum),
		FileCreatedAt:    time3(a.FileCreatedAt),
		FileModifiedAt:   time3(a.FileModifiedAt),
		CreatedAt:        time3(a.CreatedAt),
		LocalDateTime:    time3(a.LocalDateTime),
		Duration:         nullable(a.Duration),
		Type:             a.Type,
		DeletedAt:        nullTime(a.DeletedAt),
		IsFavorite:       a.IsFavorite,
		Visibility:       string(a.Visibility),
		LivePhotoVideoID: nullableID(a.LivePhotoVideoID),
		StackID:          nullableID(a.StackID),
		LibraryID:        nullableID(a.LibraryID),
		Width:            nullable(a.Width),
		Height:           nullable(a.Height),
		IsEdited:         a.IsEdited,
	}
}

func NewSocketExif(e *assetexif.Exif) SyncAssetExifV1 {
	return SyncAssetExifV1{
		AssetID: id(e.AssetID), Description: e.Description,
		ExifImageWidth: nullable(e.ExifImageWidth), ExifImageHeight: nullable(e.ExifImageHeight),
		FileSizeInByte: nullable(e.FileSizeInByte), Orientation: nullable(e.Orientation),
		DateTimeOriginal: nullTime(e.DateTimeOriginal), ModifyDate: nullTime(e.ModifyDate),
		TimeZone: nullable(e.TimeZone), Latitude: nullable(e.Latitude), Longitude: nullable(e.Longitude),
		ProjectionType: nullable(e.ProjectionType), City: nullable(e.City), State: nullable(e.State),
		Country: nullable(e.Country), Make: nullable(e.Make), Model: nullable(e.Model),
		LensModel: nullable(e.LensModel), FNumber: nullable(e.FNumber), FocalLength: nullable(e.FocalLength),
		Iso: nullable(e.ISO), ExposureTime: nullable(e.ExposureTime),
		ProfileDescription: nullable(e.ProfileDescription), Rating: nullable(e.Rating), Fps: nullable(e.FPS),
	}
}

type EditReady struct {
	Asset UploadReadyAsset  `json:"asset"`
	Edit  []SyncAssetEditV1 `json:"edit"`
}

func NewEditReady(a *asset.Asset, edits []assetedit.Edit) EditReady {
	out := EditReady{Asset: NewUploadReadyAsset(a), Edit: make([]SyncAssetEditV1, 0, len(edits))}
	for _, e := range edits {
		out.Edit = append(out.Edit, SyncAssetEditV1{ID: id(e.ID), AssetID: id(e.AssetID), Sequence: e.Sequence,
			Action: string(e.Action), Parameters: json.RawMessage(e.Parameters)})
	}

	return out
}
