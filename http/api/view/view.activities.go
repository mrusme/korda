package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/activity"
	repoactivity "tty.fail/mrus/korda/services/repositories/activity"
)

type Activity struct {
	ID        string        `json:"id"`
	AssetID   *string       `json:"assetId"`
	CreatedAt jsontime.Time `json:"createdAt"`
	Comment   *string       `json:"comment"`
	Type      activity.Type `json:"type"`
	User      AssetOwner    `json:"user"`
}

type ActivityStatistics struct {
	Comments int64 `json:"comments"`
	Likes    int64 `json:"likes"`
}

func NewActivity(r *repoactivity.Row) Activity {
	kind := activity.TypeComment
	if r.IsLiked {
		kind = activity.TypeLike
	}

	return Activity{
		ID:        id(r.ID),
		AssetID:   nullableID(r.AssetID),
		CreatedAt: time3(r.CreatedAt),
		Comment:   nullString(r.Comment),
		Type:      kind,
		User:      *NewAssetOwner(r.User()),
	}
}

func NewActivities(list []repoactivity.Row) []Activity {
	out := make([]Activity, 0, len(list))
	for i := range list {
		out = append(out, NewActivity(&list[i]))
	}

	return out
}
