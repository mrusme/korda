package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	logicalbums "tty.fail/mrus/korda/logic/root/albums"
	"tty.fail/mrus/korda/models/album"
	repoalbum "tty.fail/mrus/korda/services/repositories/album"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
)

type AlbumUser struct {
	User AssetOwner `json:"user"`
	Role album.Role `json:"role"`
}

type ContributorCount struct {
	UserID     string `json:"userId"`
	AssetCount int64  `json:"assetCount"`
}

type Album struct {
	AlbumName                  string              `json:"albumName"`
	Description                string              `json:"description"`
	AlbumThumbnailAssetID      *string             `json:"albumThumbnailAssetId"`
	CreatedAt                  jsontime.Stamp      `json:"createdAt"`
	UpdatedAt                  jsontime.Stamp      `json:"updatedAt"`
	ID                         string              `json:"id"`
	AlbumUsers                 []AlbumUser         `json:"albumUsers"`
	Shared                     bool                `json:"shared"`
	HasSharedLink              bool                `json:"hasSharedLink"`
	StartDate                  *string             `json:"startDate,omitempty"`
	EndDate                    *string             `json:"endDate,omitempty"`
	AssetCount                 int64               `json:"assetCount"`
	IsActivityEnabled          bool                `json:"isActivityEnabled"`
	Order                      album.Order         `json:"order"`
	LastModifiedAssetTimestamp *jsontime.Time      `json:"lastModifiedAssetTimestamp,omitempty"`
	ContributorCounts          *[]ContributorCount `json:"contributorCounts,omitempty"`
}

type AlbumStatistics struct {
	Owned     int `json:"owned"`
	Shared    int `json:"shared"`
	NotShared int `json:"notShared"`
}

type AlbumsAddAssets struct {
	Success bool        `json:"success"`
	Error   bulk.Reason `json:"error,omitempty"`
}

type MapMarker struct {
	ID      string  `json:"id"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	City    *string `json:"city"`
	State   *string `json:"state"`
	Country *string `json:"country"`
}

func NewAlbum(d *logicalbums.Details) Album {
	users := make([]AlbumUser, 0, len(d.Members))
	for i := range d.Members {
		users = append(users, AlbumUser{User: *NewAssetOwner(&d.Members[i].User), Role: d.Members[i].Role})
	}

	out := Album{
		AlbumName:             d.Album.AlbumName,
		Description:           d.Album.Description.V,
		AlbumThumbnailAssetID: nullableID(d.Album.AlbumThumbnailAssetID),
		CreatedAt:             jsontime.NewStamp(d.Album.CreatedAt, false),
		UpdatedAt:             jsontime.NewStamp(d.Album.UpdatedAt, false),
		ID:                    id(d.Album.ID),
		AlbumUsers:            users,
		Shared:                len(users) > 1 || d.HasSharedLink,
		HasSharedLink:         d.HasSharedLink,
		StartDate:             d.StartDate,
		EndDate:               d.EndDate,
		AssetCount:            d.AssetCount,
		IsActivityEnabled:     d.Album.IsActivityEnabled,
		Order:                 d.Album.Order,
	}
	if d.LastModified != nil {
		t := time3(*d.LastModified)
		out.LastModifiedAssetTimestamp = &t
	}
	if d.Contributors != nil {
		counts := make([]ContributorCount, 0, len(*d.Contributors))
		for _, c := range *d.Contributors {
			counts = append(counts, ContributorCount{UserID: id(c.UserID), AssetCount: c.AssetCount})
		}
		out.ContributorCounts = &counts
	}

	return out
}

func NewAlbums(list []logicalbums.Details) []Album {
	out := make([]Album, 0, len(list))
	for i := range list {
		out = append(out, NewAlbum(&list[i]))
	}

	return out
}

func NewMapMarkers(list []repoalbum.Marker) []MapMarker {
	out := make([]MapMarker, 0, len(list))
	for _, m := range list {
		out = append(out, MapMarker{ID: id(m.ID), Lat: m.Lat, Lon: m.Lon, City: nullString(m.City),
			State: nullString(m.State), Country: nullString(m.Country)})
	}

	return out
}

func NewAssetMapMarkers(list []repoasset.Marker) []MapMarker {
	out := make([]MapMarker, 0, len(list))
	for _, m := range list {
		out = append(out, MapMarker{ID: id(m.ID), Lat: m.Lat, Lon: m.Lon, City: nullString(m.City),
			State: nullString(m.State), Country: nullString(m.Country)})
	}

	return out
}

type ReverseGeocode struct {
	Country *string `json:"country"`
	State   *string `json:"state"`
	City    *string `json:"city"`
}
