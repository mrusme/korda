package view

import (
	"context"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/tag"
	"xn--gckvb8fzb.com/glides/runtime"
)

type SearchAlbums struct {
	Total  int   `json:"total"`
	Count  int   `json:"count"`
	Items  []any `json:"items"`
	Facets []any `json:"facets"`
}

type SearchAssets struct {
	Total      int     `json:"total"`
	Count      int     `json:"count"`
	Items      []Asset `json:"items"`
	Facets     []any   `json:"facets"`
	NextPage   *string `json:"nextPage"`
	NextCursor *string `json:"nextCursor"`
}

type SearchResponse struct {
	Albums SearchAlbums `json:"albums"`
	Assets SearchAssets `json:"assets"`
}

type SearchTotal struct {
	Total int64 `json:"total"`
}

type ExploreItem struct {
	Value string `json:"value"`
	Data  Asset  `json:"data"`
}

type Explore struct {
	FieldName string        `json:"fieldName"`
	Items     []ExploreItem `json:"items"`
}

func exifFor(ctx context.Context, rt *runtime.Runtime, list []asset.Asset) (map[uuid.UUID]*assetexif.Exif, error) {
	ids := make([]uuid.UUID, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}

	return gh.Repositories(rt).AssetExif.ByAssets(ctx, ids)
}

func NewAssetList(ctx context.Context, rt *runtime.Runtime, list []asset.Asset, viewer uuid.UUID, withExif bool, withTags bool) ([]Asset, error) {
	var exifs map[uuid.UUID]*assetexif.Exif
	if withExif {
		var err error
		if exifs, err = exifFor(ctx, rt, list); err != nil {
			return nil, err
		}
	}

	var tags map[uuid.UUID][]tag.Tag
	if withTags {
		ids := make([]uuid.UUID, 0, len(list))
		for _, a := range list {
			ids = append(ids, a.ID)
		}
		var err error
		if tags, err = gh.Repositories(rt).Tag.ForAssets(ctx, ids); err != nil {
			return nil, err
		}
	}

	out := make([]Asset, 0, len(list))
	for i := range list {
		d := AssetDetails{Tags: withTags, TagList: tags[list[i].ID]}
		if withExif {
			d.Exif = exifs[list[i].ID]
		}
		out = append(out, NewAsset(&list[i], viewer, d))
	}

	return out, nil
}

func NewSearchResponse(items []Asset, nextPage *string, nextCursor *string) SearchResponse {
	return SearchResponse{
		Albums: SearchAlbums{Items: []any{}, Facets: []any{}},
		Assets: SearchAssets{
			Total:      len(items),
			Count:      len(items),
			Items:      items,
			Facets:     []any{},
			NextPage:   nextPage,
			NextCursor: nextCursor,
		},
	}
}
