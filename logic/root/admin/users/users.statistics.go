package users

import (
	"context"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/asset"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

type StatisticsInput struct {
	Visibility *asset.Visibility `json:"visibility"`
	IsFavorite *bool             `json:"isFavorite"`
	IsTrashed  *bool             `json:"isTrashed"`
}

type AssetStats struct {
	Images int64
	Videos int64
	Total  int64
}

func Statistics(ctx context.Context, rt *runtime.Runtime, id uuid.UUID, in *StatisticsInput) (*AssetStats, error) {
	stats, err := gh.Repositories(rt).Asset.Statistics(ctx, id, repoasset.StatisticsFilter{
		Visibility: in.Visibility,
		IsFavorite: in.IsFavorite,
		IsTrashed:  in.IsTrashed,
	})
	if err != nil {
		return nil, err
	}

	return &AssetStats{
		Images: stats.Image,
		Videos: stats.Video,
		Total:  stats.Audio + stats.Image + stats.Video + stats.Other,
	}, nil
}
