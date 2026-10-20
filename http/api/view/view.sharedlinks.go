package view

import (
	"encoding/base64"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	logicsharedlinks "tty.fail/mrus/korda/logic/root/sharedlinks"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/sharedlink"
)

type SharedLink struct {
	ID            string          `json:"id"`
	Description   *string         `json:"description"`
	Password      *string         `json:"password"`
	UserID        string          `json:"userId"`
	Key           string          `json:"key"`
	Type          sharedlink.Type `json:"type"`
	CreatedAt     jsontime.Time   `json:"createdAt"`
	ExpiresAt     *jsontime.Time  `json:"expiresAt"`
	Assets        []any           `json:"assets"`
	Album         *Album          `json:"album,omitempty"`
	AllowUpload   bool            `json:"allowUpload"`
	AllowDownload bool            `json:"allowDownload"`
	ShowMetadata  bool            `json:"showMetadata"`
	Slug          *string         `json:"slug"`
}

type StrippedAsset struct {
	ID               string         `json:"id"`
	Type             asset.Type     `json:"type"`
	OriginalMimeType string         `json:"originalMimeType"`
	Thumbhash        *string        `json:"thumbhash"`
	LocalDateTime    jsontime.Stamp `json:"localDateTime"`
	Duration         *int64         `json:"duration"`
	LivePhotoVideoID *string        `json:"livePhotoVideoId"`
	HasMetadata      bool           `json:"hasMetadata"`
	Width            *int64         `json:"width"`
	Height           *int64         `json:"height"`
}

type AssetIDResult struct {
	AssetID string      `json:"assetId"`
	Success bool        `json:"success"`
	Error   bulk.Reason `json:"error,omitempty"`
}

func NewStrippedAsset(a *asset.Asset, postgres bool) StrippedAsset {
	return StrippedAsset{
		ID:               id(a.ID),
		Type:             a.Type,
		OriginalMimeType: mimetypes.Lookup(a.OriginalFileName),
		Thumbhash:        base64Bytes(a.Thumbhash),
		LocalDateTime:    jsontime.NewStamp(a.LocalDateTime, postgres),
		Duration:         nullable(a.Duration),
		LivePhotoVideoID: nullableID(a.LivePhotoVideoID),
		Width:            nullable(a.Width),
		Height:           nullable(a.Height),
	}
}

func NewLinkAlbum(a *logicsharedlinks.Album) *Album {
	users := make([]AlbumUser, 0, len(a.Owners))
	for i := range a.Owners {
		users = append(users, AlbumUser{User: *NewAssetOwner(&a.Owners[i]), Role: album.RoleOwner})
	}

	out := &Album{
		AlbumName:             a.Album.AlbumName,
		Description:           a.Album.Description.V,
		AlbumThumbnailAssetID: nullableID(a.Album.AlbumThumbnailAssetID),
		CreatedAt:             jsontime.NewStamp(a.Album.CreatedAt, true),
		UpdatedAt:             jsontime.NewStamp(a.Album.UpdatedAt, true),
		ID:                    id(a.Album.ID),
		AlbumUsers:            users,
		Shared:                true,
		HasSharedLink:         true,
		AssetCount:            int64(len(a.Assets)),
		IsActivityEnabled:     a.Album.IsActivityEnabled,
		Order:                 a.Album.Order,
	}
	if len(a.Assets) > 0 {
		start := jsontime.FormatPostgres(a.Assets[0].LocalDateTime)
		end := jsontime.FormatPostgres(a.Assets[len(a.Assets)-1].LocalDateTime)
		if start > end {
			start, end = end, start
		}
		out.StartDate, out.EndDate = &start, &end
	}

	return out
}

func NewSharedLink(d *logicsharedlinks.Details) SharedLink {
	l := d.Link
	assets := make([]any, 0, len(d.Assets))
	for i := range d.Assets {
		if d.Strip {
			assets = append(assets, NewStrippedAsset(&d.Assets[i], true))
			continue
		}
		details := AssetDetails{Aggregated: true}
		if d.Exif != nil {
			details.Exif = d.Exif[d.Assets[i].ID]
		}
		assets = append(assets, NewAsset(&d.Assets[i], uuid.Nil, details))
	}

	out := SharedLink{
		ID:            id(l.ID),
		Description:   nullable(l.Description),
		Password:      nullable(l.Password),
		UserID:        id(l.UserID),
		Key:           base64.RawURLEncoding.EncodeToString(l.Key),
		Type:          l.Type,
		CreatedAt:     time3(l.CreatedAt),
		ExpiresAt:     nullTime(l.ExpiresAt),
		Assets:        assets,
		AllowUpload:   l.AllowUpload,
		AllowDownload: l.AllowDownload,
		ShowMetadata:  l.ShowExif,
		Slug:          nullable(l.Slug),
	}
	if d.Album != nil {
		out.Album = NewLinkAlbum(d.Album)
	}

	return out
}

func NewSharedLinks(list []logicsharedlinks.Details) []SharedLink {
	out := make([]SharedLink, 0, len(list))
	for i := range list {
		out = append(out, NewSharedLink(&list[i]))
	}

	return out
}

func NewAssetIDResults(results []logicsharedlinks.Result) []AssetIDResult {
	out := make([]AssetIDResult, 0, len(results))
	for _, r := range results {
		out = append(out, AssetIDResult{AssetID: r.AssetID, Success: r.Success, Error: r.Error})
	}

	return out
}
