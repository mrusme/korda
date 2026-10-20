package view

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/helpers/optional"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/assetmetadata"
	"tty.fail/mrus/korda/models/tag"
	"tty.fail/mrus/korda/models/user"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

type AssetOwner struct {
	ID               string            `json:"id"`
	Email            string            `json:"email"`
	Name             string            `json:"name"`
	ProfileImagePath string            `json:"profileImagePath"`
	AvatarColor      user.AvatarColor  `json:"avatarColor"`
	ProfileChangedAt jsontime.Postgres `json:"profileChangedAt"`
}

type Exif struct {
	Make             *string            `json:"make"`
	Model            *string            `json:"model"`
	ExifImageWidth   *int64             `json:"exifImageWidth"`
	ExifImageHeight  *int64             `json:"exifImageHeight"`
	FileSizeInByte   *int64             `json:"fileSizeInByte"`
	Orientation      *string            `json:"orientation"`
	DateTimeOriginal *jsontime.Postgres `json:"dateTimeOriginal"`
	ModifyDate       *jsontime.Postgres `json:"modifyDate"`
	TimeZone         *string            `json:"timeZone"`
	LensModel        *string            `json:"lensModel"`
	FNumber          *float64           `json:"fNumber"`
	FocalLength      *float64           `json:"focalLength"`
	ISO              *int64             `json:"iso"`
	ExposureTime     *string            `json:"exposureTime"`
	Latitude         *float64           `json:"latitude"`
	Longitude        *float64           `json:"longitude"`
	City             *string            `json:"city"`
	State            *string            `json:"state"`
	Country          *string            `json:"country"`
	Description      string             `json:"description"`
	ProjectionType   *string            `json:"projectionType"`
	Rating           *int64             `json:"rating"`
}

type AssetStack struct {
	ID             string `json:"id"`
	PrimaryAssetID string `json:"primaryAssetId"`
	AssetCount     int64  `json:"assetCount"`
}

type Tag struct {
	ID        string         `json:"id"`
	ParentID  *string        `json:"parentId,omitempty"`
	Name      string         `json:"name"`
	Value     string         `json:"value"`
	CreatedAt jsontime.Stamp `json:"createdAt"`
	UpdatedAt jsontime.Stamp `json:"updatedAt"`
	Color     *string        `json:"color,omitempty"`
}

type Person struct{}

type Asset struct {
	ID               string                     `json:"id"`
	CreatedAt        jsontime.Stamp             `json:"createdAt"`
	OwnerID          string                     `json:"ownerId"`
	Owner            *AssetOwner                `json:"owner,omitempty"`
	LibraryID        *string                    `json:"libraryId"`
	Type             asset.Type                 `json:"type"`
	OriginalPath     string                     `json:"originalPath"`
	OriginalFileName string                     `json:"originalFileName"`
	OriginalMimeType string                     `json:"originalMimeType"`
	Thumbhash        *string                    `json:"thumbhash"`
	FileCreatedAt    jsontime.Stamp             `json:"fileCreatedAt"`
	FileModifiedAt   jsontime.Stamp             `json:"fileModifiedAt"`
	LocalDateTime    jsontime.Stamp             `json:"localDateTime"`
	UpdatedAt        jsontime.Stamp             `json:"updatedAt"`
	IsFavorite       bool                       `json:"isFavorite"`
	IsArchived       bool                       `json:"isArchived"`
	IsTrashed        bool                       `json:"isTrashed"`
	Visibility       asset.Visibility           `json:"visibility"`
	Duration         *int64                     `json:"duration"`
	ExifInfo         *Exif                      `json:"exifInfo,omitempty"`
	LivePhotoVideoID *string                    `json:"livePhotoVideoId"`
	Tags             *[]Tag                     `json:"tags,omitempty"`
	People           []Person                   `json:"people"`
	Checksum         string                     `json:"checksum"`
	Stack            optional.Value[AssetStack] `json:"stack,omitzero"`
	IsOffline        bool                       `json:"isOffline"`
	HasMetadata      bool                       `json:"hasMetadata"`
	DuplicateID      *string                    `json:"duplicateId"`
	Resized          bool                       `json:"resized"`
	Width            *int64                     `json:"width"`
	Height           *int64                     `json:"height"`
	IsEdited         bool                       `json:"isEdited"`
}

type AssetDetails struct {
	Owner      *user.User
	Exif       *assetexif.Exif
	Tags       bool
	TagList    []tag.Tag
	WithStack  bool
	Stack      *repoasset.StackSummary
	Aggregated bool
}

func nullableID(v uuid.NullUUID) *string {
	if !v.Valid {
		return nil
	}

	s := v.UUID.String()
	return &s
}

func base64Bytes(b []byte) *string {
	if len(b) == 0 {
		return nil
	}

	s := base64.StdEncoding.EncodeToString(b)
	return &s
}

func nullablePostgres(t sql.Null[time.Time]) *jsontime.Postgres {
	return jsontime.NullablePostgres(t)
}

func NewAssetOwner(u *user.User) *AssetOwner {
	return &AssetOwner{
		ID:               id(u.ID),
		Email:            u.Email,
		Name:             u.Name,
		ProfileImagePath: u.ProfileImagePath,
		AvatarColor:      u.Color(),
		ProfileChangedAt: jsontime.NewPostgres(u.ProfileChangedAt),
	}
}

func NewExif(e *assetexif.Exif) *Exif {
	var size *int64
	if e.FileSizeInByte.Valid && e.FileSizeInByte.V != 0 {
		size = &e.FileSizeInByte.V
	}

	return &Exif{
		Make:             nullable(e.Make),
		Model:            nullable(e.Model),
		ExifImageWidth:   nullable(e.ExifImageWidth),
		ExifImageHeight:  nullable(e.ExifImageHeight),
		FileSizeInByte:   size,
		Orientation:      nullable(e.Orientation),
		DateTimeOriginal: nullablePostgres(e.DateTimeOriginal),
		ModifyDate:       nullablePostgres(e.ModifyDate),
		TimeZone:         nullable(e.TimeZone),
		LensModel:        nullable(e.LensModel),
		FNumber:          nullable(e.FNumber),
		FocalLength:      nullable(e.FocalLength),
		ISO:              nullable(e.ISO),
		ExposureTime:     nullable(e.ExposureTime),
		Latitude:         nullable(e.Latitude),
		Longitude:        nullable(e.Longitude),
		City:             nullable(e.City),
		State:            nullable(e.State),
		Country:          nullable(e.Country),
		Description:      e.Description,
		ProjectionType:   nullable(e.ProjectionType),
		Rating:           nullable(e.Rating),
	}
}

func NewAsset(a *asset.Asset, viewer uuid.UUID, d AssetDetails) Asset {
	out := Asset{
		ID:               id(a.ID),
		CreatedAt:        jsontime.NewStamp(a.CreatedAt, d.Aggregated),
		OwnerID:          id(a.OwnerID),
		LibraryID:        nullableID(a.LibraryID),
		Type:             a.Type,
		OriginalPath:     a.OriginalPath,
		OriginalFileName: a.OriginalFileName,
		OriginalMimeType: mimetypes.Lookup(a.OriginalFileName),
		Thumbhash:        base64Bytes(a.Thumbhash),
		FileCreatedAt:    jsontime.NewStamp(a.FileCreatedAt, d.Aggregated),
		FileModifiedAt:   jsontime.NewStamp(a.FileModifiedAt, d.Aggregated),
		LocalDateTime:    jsontime.NewStamp(a.LocalDateTime, d.Aggregated),
		UpdatedAt:        jsontime.NewStamp(a.UpdatedAt, d.Aggregated),
		IsFavorite:       viewer == a.OwnerID && a.IsFavorite,
		IsArchived:       a.Visibility == asset.VisibilityArchive,
		IsTrashed:        a.DeletedAt.Valid,
		Visibility:       a.Visibility,
		Duration:         nullable(a.Duration),
		LivePhotoVideoID: nullableID(a.LivePhotoVideoID),
		People:           []Person{},
		Checksum:         base64.StdEncoding.EncodeToString(a.Checksum),
		IsOffline:        a.IsOffline,
		HasMetadata:      true,
		DuplicateID:      nullableID(a.DuplicateID),
		Resized:          true,
		Width:            nullable(a.Width),
		Height:           nullable(a.Height),
		IsEdited:         a.IsEdited,
	}

	if d.Owner != nil {
		out.Owner = NewAssetOwner(d.Owner)
	}
	if d.Exif != nil {
		out.ExifInfo = NewExif(d.Exif)
	}
	if d.Tags {
		tags := make([]Tag, 0, len(d.TagList))
		for i := range d.TagList {
			tags = append(tags, NewTag(&d.TagList[i], true))
		}
		out.Tags = &tags
	}
	if d.WithStack {
		out.Stack = optional.Value[AssetStack]{Set: true, Null: true}
		if d.Stack != nil {
			out.Stack = optional.Of(AssetStack{
				ID: id(d.Stack.ID), PrimaryAssetID: id(d.Stack.PrimaryAssetID), AssetCount: d.Stack.AssetCount,
			})
		}
	}

	return out
}

type AssetUploaded struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type AssetDuplicate struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

func NewAssetUploaded(assetID uuid.UUID) AssetUploaded {
	return AssetUploaded{ID: id(assetID), Status: "created"}
}

func NewAssetDuplicate(assetID uuid.UUID) AssetDuplicate {
	return AssetDuplicate{Status: "duplicate", ID: id(assetID)}
}

type AssetStatistics struct {
	Images int64 `json:"images"`
	Videos int64 `json:"videos"`
	Total  int64 `json:"total"`
}

type BulkUploadCheckResult struct {
	ID        string  `json:"id"`
	Action    string  `json:"action"`
	Reason    string  `json:"reason,omitempty"`
	AssetID   *string `json:"assetId,omitempty"`
	IsTrashed *bool   `json:"isTrashed,omitempty"`
}

type BulkUploadCheck struct {
	Results []BulkUploadCheckResult `json:"results"`
}

func NewBulkUploadCheck(results []logicassets.BulkCheckResult) BulkUploadCheck {
	out := BulkUploadCheck{Results: make([]BulkUploadCheckResult, 0, len(results))}
	for _, r := range results {
		if r.Accept {
			out.Results = append(out.Results, BulkUploadCheckResult{ID: r.ID, Action: "accept"})
			continue
		}

		assetID := r.AssetID.String()
		trashed := r.IsTrashed
		out.Results = append(out.Results, BulkUploadCheckResult{
			ID:        r.ID,
			Action:    "reject",
			Reason:    "duplicate",
			AssetID:   &assetID,
			IsTrashed: &trashed,
		})
	}

	return out
}

type AssetMetadata struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt jsontime.Time   `json:"updatedAt"`
}

type AssetMetadataBulk struct {
	AssetID   string          `json:"assetId"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt jsontime.Time   `json:"updatedAt"`
}

func NewAssetMetadata(item *assetmetadata.Item) AssetMetadata {
	return AssetMetadata{Key: item.Key, Value: json.RawMessage(item.Value), UpdatedAt: time3(item.UpdatedAt)}
}

func NewAssetMetadataList(list []assetmetadata.Item) []AssetMetadata {
	out := make([]AssetMetadata, 0, len(list))
	for i := range list {
		out = append(out, NewAssetMetadata(&list[i]))
	}

	return out
}

func NewAssetMetadataBulkList(list []assetmetadata.Item) []AssetMetadataBulk {
	out := make([]AssetMetadataBulk, 0, len(list))
	for _, item := range list {
		out = append(out, AssetMetadataBulk{
			AssetID:   id(item.AssetID),
			Key:       item.Key,
			Value:     json.RawMessage(item.Value),
			UpdatedAt: time3(item.UpdatedAt),
		})
	}

	return out
}

type AssetOcr struct{}

type AssetEditAction struct {
	ID         string          `json:"id"`
	Action     string          `json:"action"`
	Parameters json.RawMessage `json:"parameters"`
}

type AssetEdits struct {
	AssetID string            `json:"assetId"`
	Edits   []AssetEditAction `json:"edits"`
}

func NewAssetEdits(assetID uuid.UUID) AssetEdits {
	return AssetEdits{AssetID: id(assetID), Edits: []AssetEditAction{}}
}

func NewAssetsWithExif(ctx context.Context, rt *runtime.Runtime, list []asset.Asset, viewer uuid.UUID) ([]Asset, error) {
	ids := make([]uuid.UUID, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}

	exifs, err := gh.Repositories(rt).AssetExif.ByAssets(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]Asset, 0, len(list))
	for i := range list {
		out = append(out, NewAsset(&list[i], viewer, AssetDetails{Exif: exifs[list[i].ID]}))
	}

	return out, nil
}
