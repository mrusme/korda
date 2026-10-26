package view

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/jsonb"
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/http/api/problem"
	logicsync "tty.fail/mrus/korda/logic/root/sync"
	modelsync "tty.fail/mrus/korda/models/sync"
	"tty.fail/mrus/korda/models/synccheckpoint"
	reposync "tty.fail/mrus/korda/services/repositories/sync"
)

type SyncLine struct {
	Type string `json:"type"`
	Data any    `json:"data"`
	Ack  string `json:"ack"`
}

type SyncEmpty struct{}

type SyncAuthUserV2 struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Email             string         `json:"email"`
	AvatarColor       *string        `json:"avatarColor"`
	DeletedAt         *jsontime.Time `json:"deletedAt"`
	ProfileChangedAt  jsontime.Time  `json:"profileChangedAt"`
	IsAdmin           bool           `json:"isAdmin"`
	PinCode           *string        `json:"pinCode"`
	OAuthID           *string        `json:"oauthId"`
	StorageLabel      *string        `json:"storageLabel"`
	QuotaSizeInBytes  *int64         `json:"quotaSizeInBytes"`
	QuotaUsageInBytes int64          `json:"quotaUsageInBytes"`
	HasProfileImage   bool           `json:"hasProfileImage"`
}

type SyncAuthUserV1 struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Email             string         `json:"email"`
	AvatarColor       *string        `json:"avatarColor"`
	DeletedAt         *jsontime.Time `json:"deletedAt"`
	ProfileChangedAt  jsontime.Time  `json:"profileChangedAt"`
	IsAdmin           bool           `json:"isAdmin"`
	PinCode           *string        `json:"pinCode"`
	OAuthID           string         `json:"oauthId"`
	StorageLabel      *string        `json:"storageLabel"`
	QuotaSizeInBytes  *int64         `json:"quotaSizeInBytes"`
	QuotaUsageInBytes int64          `json:"quotaUsageInBytes"`
	HasProfileImage   bool           `json:"hasProfileImage"`
}

type SyncUserV1 struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Email            string         `json:"email"`
	AvatarColor      *string        `json:"avatarColor"`
	DeletedAt        *jsontime.Time `json:"deletedAt"`
	ProfileChangedAt jsontime.Time  `json:"profileChangedAt"`
	HasProfileImage  bool           `json:"hasProfileImage"`
}

type SyncUserDeleteV1 struct {
	UserID string `json:"userId"`
}

type SyncAssetV2 struct {
	ID               string         `json:"id"`
	OwnerID          string         `json:"ownerId"`
	OriginalFileName string         `json:"originalFileName"`
	FileCreatedAt    jsontime.Time  `json:"fileCreatedAt"`
	FileModifiedAt   jsontime.Time  `json:"fileModifiedAt"`
	CreatedAt        jsontime.Time  `json:"createdAt"`
	LocalDateTime    jsontime.Time  `json:"localDateTime"`
	Type             string         `json:"type"`
	DeletedAt        *jsontime.Time `json:"deletedAt"`
	IsFavorite       bool           `json:"isFavorite"`
	Visibility       string         `json:"visibility"`
	Duration         *int64         `json:"duration"`
	LivePhotoVideoID *string        `json:"livePhotoVideoId"`
	StackID          *string        `json:"stackId"`
	LibraryID        *string        `json:"libraryId"`
	Width            *int64         `json:"width"`
	Height           *int64         `json:"height"`
	IsEdited         bool           `json:"isEdited"`
	Checksum         string         `json:"checksum"`
	Thumbhash        *string        `json:"thumbhash"`
}

type SyncAssetDeleteV1 struct {
	AssetID string `json:"assetId"`
}

type SyncPartnerAssetV2 struct {
	ID               string         `json:"id"`
	OwnerID          string         `json:"ownerId"`
	OriginalFileName string         `json:"originalFileName"`
	FileCreatedAt    jsontime.Time  `json:"fileCreatedAt"`
	FileModifiedAt   jsontime.Time  `json:"fileModifiedAt"`
	LocalDateTime    jsontime.Time  `json:"localDateTime"`
	CreatedAt        jsontime.Time  `json:"createdAt"`
	Type             string         `json:"type"`
	DeletedAt        *jsontime.Time `json:"deletedAt"`
	Visibility       string         `json:"visibility"`
	Duration         *int64         `json:"duration"`
	LivePhotoVideoID *string        `json:"livePhotoVideoId"`
	StackID          *string        `json:"stackId"`
	LibraryID        *string        `json:"libraryId"`
	Width            *int64         `json:"width"`
	Height           *int64         `json:"height"`
	IsEdited         bool           `json:"isEdited"`
	IsFavorite       bool           `json:"isFavorite"`
	Checksum         string         `json:"checksum"`
	Thumbhash        *string        `json:"thumbhash"`
}

type SyncAlbumAssetV2 struct {
	ID               string         `json:"id"`
	OwnerID          string         `json:"ownerId"`
	OriginalFileName string         `json:"originalFileName"`
	FileCreatedAt    jsontime.Time  `json:"fileCreatedAt"`
	FileModifiedAt   jsontime.Time  `json:"fileModifiedAt"`
	CreatedAt        jsontime.Time  `json:"createdAt"`
	LocalDateTime    jsontime.Time  `json:"localDateTime"`
	Type             string         `json:"type"`
	DeletedAt        *jsontime.Time `json:"deletedAt"`
	Visibility       string         `json:"visibility"`
	Duration         *int64         `json:"duration"`
	LivePhotoVideoID *string        `json:"livePhotoVideoId"`
	StackID          *string        `json:"stackId"`
	LibraryID        *string        `json:"libraryId"`
	Width            *int64         `json:"width"`
	Height           *int64         `json:"height"`
	IsEdited         bool           `json:"isEdited"`
	IsFavorite       bool           `json:"isFavorite"`
	Checksum         string         `json:"checksum"`
	Thumbhash        *string        `json:"thumbhash"`
}

type SyncPartnerV1 struct {
	SharedByID   string `json:"sharedById"`
	SharedWithID string `json:"sharedWithId"`
	InTimeline   bool   `json:"inTimeline"`
}

type SyncPartnerDeleteV1 struct {
	SharedByID   string `json:"sharedById"`
	SharedWithID string `json:"sharedWithId"`
}

type SyncStackV1 struct {
	ID             string        `json:"id"`
	CreatedAt      jsontime.Time `json:"createdAt"`
	UpdatedAt      jsontime.Time `json:"updatedAt"`
	PrimaryAssetID string        `json:"primaryAssetId"`
	OwnerID        string        `json:"ownerId"`
}

type SyncStackDeleteV1 struct {
	StackID string `json:"stackId"`
}

type SyncAlbumV2 struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	CreatedAt         jsontime.Time `json:"createdAt"`
	UpdatedAt         jsontime.Time `json:"updatedAt"`
	ThumbnailAssetID  *string       `json:"thumbnailAssetId"`
	IsActivityEnabled bool          `json:"isActivityEnabled"`
	Order             string        `json:"order"`
}

type SyncAlbumV1 struct {
	SyncAlbumV2
	OwnerID string `json:"ownerId"`
}

type SyncAlbumDeleteV1 struct {
	AlbumID string `json:"albumId"`
}

type SyncAlbumUserV1 struct {
	AlbumID string `json:"albumId"`
	UserID  string `json:"userId"`
	Role    string `json:"role"`
}

type SyncAlbumUserDeleteV1 struct {
	UserID  string `json:"userId"`
	AlbumID string `json:"albumId"`
}

type SyncAlbumToAssetV1 struct {
	AssetID string `json:"assetId"`
	AlbumID string `json:"albumId"`
}

type SyncMemoryV2 struct {
	ID        string          `json:"id"`
	CreatedAt jsontime.Time   `json:"createdAt"`
	UpdatedAt jsontime.Time   `json:"updatedAt"`
	DeletedAt *jsontime.Time  `json:"deletedAt"`
	OwnerID   string          `json:"ownerId"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	IsSaved   bool            `json:"isSaved"`
	MemoryAt  jsontime.Time   `json:"memoryAt"`
	SeenAt    *jsontime.Time  `json:"seenAt"`
	ShowAt    *jsontime.Time  `json:"showAt"`
	HideAt    *jsontime.Time  `json:"hideAt"`
}

type SyncMemoryDeleteV1 struct {
	MemoryID string `json:"memoryId"`
}

type SyncMemoryAssetV1 struct {
	MemoryID string `json:"memoryId"`
	AssetID  string `json:"assetId"`
}

type SyncAssetExifV1 struct {
	AssetID            string         `json:"assetId"`
	Description        string         `json:"description"`
	ExifImageWidth     *int64         `json:"exifImageWidth"`
	ExifImageHeight    *int64         `json:"exifImageHeight"`
	FileSizeInByte     *int64         `json:"fileSizeInByte"`
	Orientation        *string        `json:"orientation"`
	DateTimeOriginal   *jsontime.Time `json:"dateTimeOriginal"`
	ModifyDate         *jsontime.Time `json:"modifyDate"`
	TimeZone           *string        `json:"timeZone"`
	Latitude           *float64       `json:"latitude"`
	Longitude          *float64       `json:"longitude"`
	ProjectionType     *string        `json:"projectionType"`
	City               *string        `json:"city"`
	State              *string        `json:"state"`
	Country            *string        `json:"country"`
	Make               *string        `json:"make"`
	Model              *string        `json:"model"`
	LensModel          *string        `json:"lensModel"`
	FNumber            *float64       `json:"fNumber"`
	FocalLength        *float64       `json:"focalLength"`
	Iso                *int64         `json:"iso"`
	ExposureTime       *string        `json:"exposureTime"`
	ProfileDescription *string        `json:"profileDescription"`
	Rating             *int64         `json:"rating"`
	Fps                *float64       `json:"fps"`
}

type SyncUserMetadataV1 struct {
	UserID string          `json:"userId"`
	Key    string          `json:"key"`
	Value  json.RawMessage `json:"value"`
}

type SyncUserMetadataDeleteV1 struct {
	UserID string `json:"userId"`
	Key    string `json:"key"`
}

type SyncAssetMetadataV1 struct {
	AssetID string          `json:"assetId"`
	Key     string          `json:"key"`
	Value   json.RawMessage `json:"value"`
}

type SyncAssetMetadataDeleteV1 struct {
	AssetID string `json:"assetId"`
	Key     string `json:"key"`
}

type SyncAssetEditV1 struct {
	ID         string          `json:"id"`
	AssetID    string          `json:"assetId"`
	Sequence   int64           `json:"sequence"`
	Action     string          `json:"action"`
	Parameters json.RawMessage `json:"parameters"`
}

type SyncAssetEditDeleteV1 struct {
	EditID string `json:"editId"`
}

type SyncAck struct {
	Type string `json:"type"`
	Ack  string `json:"ack"`
}

func NewSyncAcks(list []synccheckpoint.Checkpoint) []SyncAck {
	out := make([]SyncAck, 0, len(list))
	for _, c := range list {
		out = append(out, SyncAck{Type: c.Type, Ack: c.Ack})
	}

	return out
}

func nullString[T ~string](v sql.Null[T]) *string {
	if !v.Valid {
		return nil
	}
	s := string(v.V)

	return &s
}

func nullValue[T any](v sql.Null[T]) *T {
	if !v.Valid {
		return nil
	}

	return &v.V
}

func nullUUID(v uuid.NullUUID) *string {
	if !v.Valid {
		return nil
	}
	s := v.UUID.String()

	return &s
}

func nullTime(v sql.Null[time.Time]) *jsontime.Time {
	return jsontime.Nullable(v)
}

func normalized(value string) (json.RawMessage, error) {
	out, err := jsonb.Normalize([]byte(value))
	if err != nil {
		return nil, err
	}

	return json.RawMessage(out), nil
}

func syncData(l logicsync.Line) (any, error) {
	switch row := l.Data.(type) {
	case nil:
		return SyncEmpty{}, nil
	case reposync.AuthUser:
		if l.Type == modelsync.AuthUserV1 {
			oauthID := ""
			if row.OAuthID.Valid {
				oauthID = row.OAuthID.V
			}
			return SyncAuthUserV1{
				ID: row.ID.String(), Name: row.Name, Email: row.Email, AvatarColor: nullString(row.AvatarColor),
				DeletedAt: nullTime(row.DeletedAt), ProfileChangedAt: jsontime.New(row.ProfileChangedAt),
				IsAdmin: row.IsAdmin, PinCode: nullString(row.PinCode), OAuthID: oauthID,
				StorageLabel: nullString(row.StorageLabel), QuotaSizeInBytes: nullValue(row.QuotaSizeInBytes),
				QuotaUsageInBytes: row.QuotaUsageInBytes, HasProfileImage: row.ProfileImagePath != "",
			}, nil
		}
		return SyncAuthUserV2{
			ID: row.ID.String(), Name: row.Name, Email: row.Email, AvatarColor: nullString(row.AvatarColor),
			DeletedAt: nullTime(row.DeletedAt), ProfileChangedAt: jsontime.New(row.ProfileChangedAt),
			IsAdmin: row.IsAdmin, PinCode: nullString(row.PinCode), OAuthID: nullString(row.OAuthID),
			StorageLabel: nullString(row.StorageLabel), QuotaSizeInBytes: nullValue(row.QuotaSizeInBytes),
			QuotaUsageInBytes: row.QuotaUsageInBytes, HasProfileImage: row.ProfileImagePath != "",
		}, nil
	case reposync.User:
		return SyncUserV1{
			ID: row.ID.String(), Name: row.Name, Email: row.Email, AvatarColor: nullString(row.AvatarColor),
			DeletedAt: nullTime(row.DeletedAt), ProfileChangedAt: jsontime.New(row.ProfileChangedAt),
			HasProfileImage: row.ProfileImagePath != "",
		}, nil
	case reposync.UserDelete:
		return SyncUserDeleteV1{UserID: row.UserID.String()}, nil
	case reposync.Asset:
		return SyncAssetV2{
			ID: row.ID.String(), OwnerID: row.OwnerID.String(), OriginalFileName: row.OriginalFileName,
			FileCreatedAt: jsontime.New(row.FileCreatedAt), FileModifiedAt: jsontime.New(row.FileModifiedAt),
			CreatedAt: jsontime.New(row.CreatedAt), LocalDateTime: jsontime.New(row.LocalDateTime), Type: row.Type,
			DeletedAt: nullTime(row.DeletedAt), IsFavorite: row.IsFavorite, Visibility: row.Visibility,
			Duration: nullValue(row.Duration), LivePhotoVideoID: nullUUID(row.LivePhotoVideoID),
			StackID: nullUUID(row.StackID), LibraryID: nullUUID(row.LibraryID), Width: nullValue(row.Width),
			Height: nullValue(row.Height), IsEdited: row.IsEdited,
			Checksum: base64.StdEncoding.EncodeToString(row.Checksum), Thumbhash: thumbhash(row.Thumbhash),
		}, nil
	case reposync.AssetDelete:
		return SyncAssetDeleteV1{AssetID: row.AssetID.String()}, nil
	case reposync.AssetExif:
		return syncExif(row), nil
	case reposync.AlbumAssetExifUpdate:
		return syncExif(row.AssetExif), nil
	case reposync.PartnerAsset:
		return syncPartnerAsset(row), nil
	case reposync.AlbumAsset:
		return syncAlbumAsset(row), nil
	case reposync.AlbumAssetUpdate:
		return syncAlbumAsset(row.AlbumAsset), nil
	case reposync.Partner:
		return SyncPartnerV1{SharedByID: row.SharedByID.String(), SharedWithID: row.SharedWithID.String(),
			InTimeline: row.InTimeline}, nil
	case reposync.PartnerDelete:
		return SyncPartnerDeleteV1{SharedByID: row.SharedByID.String(), SharedWithID: row.SharedWithID.String()}, nil
	case reposync.Stack:
		return SyncStackV1{ID: row.ID.String(), CreatedAt: jsontime.New(row.CreatedAt), UpdatedAt: jsontime.New(row.UpdatedAt),
			PrimaryAssetID: row.PrimaryAssetID.String(), OwnerID: row.OwnerID.String()}, nil
	case reposync.StackDelete:
		return SyncStackDeleteV1{StackID: row.StackID.String()}, nil
	case reposync.Album:
		return syncAlbum(row), nil
	case reposync.AlbumV1:
		return SyncAlbumV1{SyncAlbumV2: syncAlbum(row.Album), OwnerID: row.OwnerID.String()}, nil
	case reposync.AlbumDelete:
		return SyncAlbumDeleteV1{AlbumID: row.AlbumID.String()}, nil
	case reposync.AlbumUser:
		return SyncAlbumUserV1{AlbumID: row.AlbumID.String(), UserID: row.UserID.String(), Role: row.Role}, nil
	case reposync.AlbumUserDelete:
		return SyncAlbumUserDeleteV1{UserID: row.UserID.String(), AlbumID: row.AlbumID.String()}, nil
	case reposync.AlbumToAsset:
		return SyncAlbumToAssetV1{AssetID: row.AssetID.String(), AlbumID: row.AlbumID.String()}, nil
	case reposync.AlbumToAssetDelete:
		return SyncAlbumToAssetV1{AssetID: row.AssetID.String(), AlbumID: row.AlbumID.String()}, nil
	case reposync.Memory:
		data, err := normalized(row.Data)
		if err != nil {
			return nil, err
		}
		return SyncMemoryV2{
			ID: row.ID.String(), CreatedAt: jsontime.New(row.CreatedAt), UpdatedAt: jsontime.New(row.UpdatedAt),
			DeletedAt: nullTime(row.DeletedAt), OwnerID: row.OwnerID.String(), Type: row.Type, Data: data,
			IsSaved: row.IsSaved, MemoryAt: jsontime.New(row.MemoryAt), SeenAt: nullTime(row.SeenAt),
			ShowAt: nullTime(row.ShowAt), HideAt: nullTime(row.HideAt),
		}, nil
	case reposync.MemoryDelete:
		return SyncMemoryDeleteV1{MemoryID: row.MemoryID.String()}, nil
	case reposync.MemoryAsset:
		return SyncMemoryAssetV1{MemoryID: row.MemoryID.String(), AssetID: row.AssetID.String()}, nil
	case reposync.MemoryAssetDelete:
		return SyncMemoryAssetV1{MemoryID: row.MemoryID.String(), AssetID: row.AssetID.String()}, nil
	case reposync.UserMetadata:
		value, err := normalized(row.Value)
		if err != nil {
			return nil, err
		}
		return SyncUserMetadataV1{UserID: row.UserID.String(), Key: row.Key, Value: value}, nil
	case reposync.UserMetadataDelete:
		return SyncUserMetadataDeleteV1{UserID: row.UserID.String(), Key: row.Key}, nil
	case reposync.AssetMetadata:
		value, err := normalized(row.Value)
		if err != nil {
			return nil, err
		}
		return SyncAssetMetadataV1{AssetID: row.AssetID.String(), Key: row.Key, Value: value}, nil
	case reposync.AssetMetadataDelete:
		return SyncAssetMetadataDeleteV1{AssetID: row.AssetID.String(), Key: row.Key}, nil
	case reposync.AssetEdit:
		return SyncAssetEditV1{ID: row.ID.String(), AssetID: row.AssetID.String(), Sequence: row.Sequence,
			Action: row.Action, Parameters: json.RawMessage(row.Parameters)}, nil
	case reposync.AssetEditDelete:
		return SyncAssetEditDeleteV1{EditID: row.EditID.String()}, nil
	}

	return nil, fmt.Errorf("no sync payload for %T", l.Data)
}

func syncExif(row reposync.AssetExif) SyncAssetExifV1 {
	return SyncAssetExifV1{
		AssetID: row.AssetID.String(), Description: row.Description,
		ExifImageWidth: nullValue(row.ExifImageWidth), ExifImageHeight: nullValue(row.ExifImageHeight),
		FileSizeInByte: nullValue(row.FileSizeInByte), Orientation: nullString(row.Orientation),
		DateTimeOriginal: nullTime(row.DateTimeOriginal), ModifyDate: nullTime(row.ModifyDate),
		TimeZone: nullString(row.TimeZone), Latitude: nullValue(row.Latitude), Longitude: nullValue(row.Longitude),
		ProjectionType: nullString(row.ProjectionType), City: nullString(row.City), State: nullString(row.State),
		Country: nullString(row.Country), Make: nullString(row.Make), Model: nullString(row.Model),
		LensModel: nullString(row.LensModel), FNumber: nullValue(row.FNumber), FocalLength: nullValue(row.FocalLength),
		Iso: nullValue(row.Iso), ExposureTime: nullString(row.ExposureTime),
		ProfileDescription: nullString(row.ProfileDescription), Rating: nullValue(row.Rating), Fps: nullValue(row.Fps),
	}
}

func thumbhash(b []byte) *string {
	if b == nil {
		return nil
	}
	s := base64.StdEncoding.EncodeToString(b)

	return &s
}

func syncPartnerAsset(row reposync.PartnerAsset) SyncPartnerAssetV2 {
	return SyncPartnerAssetV2{
		ID: row.ID.String(), OwnerID: row.OwnerID.String(), OriginalFileName: row.OriginalFileName,
		FileCreatedAt: jsontime.New(row.FileCreatedAt), FileModifiedAt: jsontime.New(row.FileModifiedAt),
		LocalDateTime: jsontime.New(row.LocalDateTime), CreatedAt: jsontime.New(row.CreatedAt), Type: row.Type,
		DeletedAt: nullTime(row.DeletedAt), Visibility: row.Visibility, Duration: nullValue(row.Duration),
		LivePhotoVideoID: nullUUID(row.LivePhotoVideoID), StackID: nullUUID(row.StackID),
		LibraryID: nullUUID(row.LibraryID), Width: nullValue(row.Width), Height: nullValue(row.Height),
		IsEdited: row.IsEdited, IsFavorite: row.IsFavorite,
		Checksum: base64.StdEncoding.EncodeToString(row.Checksum), Thumbhash: thumbhash(row.Thumbhash),
	}
}

func syncAlbumAsset(row reposync.AlbumAsset) SyncAlbumAssetV2 {
	return SyncAlbumAssetV2{
		ID: row.ID.String(), OwnerID: row.OwnerID.String(), OriginalFileName: row.OriginalFileName,
		FileCreatedAt: jsontime.New(row.FileCreatedAt), FileModifiedAt: jsontime.New(row.FileModifiedAt),
		CreatedAt: jsontime.New(row.CreatedAt), LocalDateTime: jsontime.New(row.LocalDateTime), Type: row.Type,
		DeletedAt: nullTime(row.DeletedAt), Visibility: row.Visibility, Duration: nullValue(row.Duration),
		LivePhotoVideoID: nullUUID(row.LivePhotoVideoID), StackID: nullUUID(row.StackID),
		LibraryID: nullUUID(row.LibraryID), Width: nullValue(row.Width), Height: nullValue(row.Height),
		IsEdited: row.IsEdited, IsFavorite: row.IsFavorite,
		Checksum: base64.StdEncoding.EncodeToString(row.Checksum), Thumbhash: thumbhash(row.Thumbhash),
	}
}

func syncAlbum(row reposync.Album) SyncAlbumV2 {
	return SyncAlbumV2{
		ID: row.ID.String(), Name: row.Name, Description: row.Description.V, CreatedAt: jsontime.New(row.CreatedAt),
		UpdatedAt: jsontime.New(row.UpdatedAt), ThumbnailAssetID: nullUUID(row.ThumbnailAssetID),
		IsActivityEnabled: row.IsActivityEnabled, Order: row.Order,
	}
}

func EncodeSyncLine(l logicsync.Line) ([]byte, error) {
	data, err := syncData(l)
	if err != nil {
		return nil, err
	}
	encoded, err := problem.Encode(SyncLine{Type: string(l.Type), Data: data, Ack: l.Ack})
	if err != nil {
		return nil, err
	}

	return append(encoded, '\n'), nil
}
