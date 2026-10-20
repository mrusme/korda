package view

import (
	"context"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	logicstacks "tty.fail/mrus/korda/logic/root/stacks"
	"tty.fail/mrus/korda/models/tag"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Stack struct {
	ID             string  `json:"id"`
	PrimaryAssetID string  `json:"primaryAssetId"`
	Assets         []Asset `json:"assets"`
}

func NewStack(ctx context.Context, rt *runtime.Runtime, d *logicstacks.Details, viewer uuid.UUID) (Stack, error) {
	ids := make([]uuid.UUID, 0, len(d.Assets))
	for _, a := range d.Assets {
		ids = append(ids, a.ID)
	}

	exifs, err := gh.Repositories(rt).AssetExif.ByAssets(ctx, ids)
	if err != nil {
		return Stack{}, err
	}
	tags := map[uuid.UUID][]tag.Tag{}
	if d.WithTags {
		if tags, err = gh.Repositories(rt).Tag.ForAssets(ctx, ids); err != nil {
			return Stack{}, err
		}
	}

	out := Stack{ID: id(d.Stack.ID), PrimaryAssetID: id(d.Stack.PrimaryAssetID), Assets: make([]Asset, 0, len(d.Assets))}
	for i := range d.Assets {
		a := &d.Assets[i]
		out.Assets = append(out.Assets, NewAsset(a, viewer, AssetDetails{Exif: exifs[a.ID], Tags: d.WithTags, TagList: tags[a.ID],
			Aggregated: true}))
	}

	return out, nil
}

func NewStacks(ctx context.Context, rt *runtime.Runtime, list []logicstacks.Details, viewer uuid.UUID) ([]Stack, error) {
	out := make([]Stack, 0, len(list))
	for i := range list {
		s, err := NewStack(ctx, rt, &list[i], viewer)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, nil
}
