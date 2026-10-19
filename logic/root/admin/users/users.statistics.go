package users

import (
	"context"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
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

func Statistics(ctx context.Context, id uuid.UUID, in *StatisticsInput) (*AssetStats, error) {
	return &AssetStats{}, nil
}
