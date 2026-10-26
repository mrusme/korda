package geomap

import (
	"context"
	"time"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

type MarkersInput struct {
	IsArchived        *bool      `json:"isArchived"`
	IsFavorite        *bool      `json:"isFavorite"`
	FileCreatedAfter  *time.Time `json:"fileCreatedAfter" v:"datetime"`
	FileCreatedBefore *time.Time `json:"fileCreatedBefore" v:"datetime"`
	WithPartners      *bool      `json:"withPartners"`
	WithSharedAlbums  *bool      `json:"withSharedAlbums"`
}

type ReverseInput struct {
	Lat float64 `json:"lat" v:"coerce,min=-90,max=90"`
	Lon float64 `json:"lon" v:"coerce,min=-180,max=180"`
}

func Markers(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *MarkersInput) ([]repoasset.Marker, error) {
	repos := gh.Repositories(rt)
	owners := []uuid.UUID{a.User.ID}
	if in.WithPartners != nil && *in.WithPartners {
		partners, err := repos.Partner.SharingWithMe(ctx, a.User.ID)
		if err != nil {
			return nil, err
		}
		owners = append(owners, partners...)
	}

	return repos.Asset.MapMarkers(ctx, repoasset.MarkerFilter{
		Caller:            a.User.ID,
		OwnerIDs:          owners,
		WithSharedAlbums:  in.WithSharedAlbums != nil && *in.WithSharedAlbums,
		IsArchived:        in.IsArchived,
		IsFavorite:        in.IsFavorite,
		FileCreatedAfter:  in.FileCreatedAfter,
		FileCreatedBefore: in.FileCreatedBefore,
	})
}
