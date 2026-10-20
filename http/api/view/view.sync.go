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
		var thumbhash *string
		if row.Thumbhash != nil {
			s := base64.StdEncoding.EncodeToString(row.Thumbhash)
			thumbhash = &s
		}
		return SyncAssetV2{
			ID: row.ID.String(), OwnerID: row.OwnerID.String(), OriginalFileName: row.OriginalFileName,
			FileCreatedAt: jsontime.New(row.FileCreatedAt), FileModifiedAt: jsontime.New(row.FileModifiedAt),
			CreatedAt: jsontime.New(row.CreatedAt), LocalDateTime: jsontime.New(row.LocalDateTime), Type: row.Type,
			DeletedAt: nullTime(row.DeletedAt), IsFavorite: row.IsFavorite, Visibility: row.Visibility,
			Duration: nullValue(row.Duration), LivePhotoVideoID: nullUUID(row.LivePhotoVideoID),
			StackID: nullUUID(row.StackID), LibraryID: nullUUID(row.LibraryID), Width: nullValue(row.Width),
			Height: nullValue(row.Height), IsEdited: row.IsEdited,
			Checksum: base64.StdEncoding.EncodeToString(row.Checksum), Thumbhash: thumbhash,
		}, nil
	case reposync.AssetDelete:
		return SyncAssetDeleteV1{AssetID: row.AssetID.String()}, nil
	case reposync.AssetExif:
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
		}, nil
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
	}

	return nil, fmt.Errorf("no sync payload for %T", l.Data)
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
