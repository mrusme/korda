package assets

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/jsonb"
	"tty.fail/mrus/korda/models/assetmetadata"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

var ErrEmptyBulkMetadata = errors.New("no metadata items to upsert")

type MetadataRouteInput struct {
	ID  string `json:"id" v:"uuid"`
	Key string `json:"key"`
}

func duplicateItem(key string) error {
	return fmt.Errorf("%w: \"%s\"", errs.ErrDuplicateItems, key)
}

func ListMetadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) ([]assetmetadata.Item, error) {
	if err := require(ctx, rt, a, permission.AssetRead, id); err != nil {
		return nil, err
	}

	return gh.Repositories(rt).AssetMetadata.List(ctx, id)
}

func GetMetadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, key string) (*assetmetadata.Item, error) {
	if err := require(ctx, rt, a, permission.AssetRead, id); err != nil {
		return nil, err
	}

	item, err := gh.Repositories(rt).AssetMetadata.Get(ctx, id, key)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, fmt.Errorf("%w \"%s\" not found for asset with id \"%s\"", errs.ErrMetadataNotFound, key, id)
	}

	return item, err
}

func items(assetID uuid.UUID, in []MetadataItemInput) ([]assetmetadata.Item, error) {
	out := make([]assetmetadata.Item, 0, len(in))
	for _, item := range in {
		value, err := jsonb.Normalize(item.Value)
		if err != nil {
			return nil, err
		}
		out = append(out, assetmetadata.Item{AssetID: assetID, Key: item.Key, Value: string(value)})
	}

	return out, nil
}

func UpsertMetadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *MetadataUpsertInput) ([]assetmetadata.Item, error) {
	if err := require(ctx, rt, a, permission.AssetUpdate, id); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(in.Items))
	for _, item := range in.Items {
		if seen[item.Key] {
			return nil, duplicateItem(item.Key)
		}
		seen[item.Key] = true
	}
	if len(in.Items) == 0 {
		return []assetmetadata.Item{}, nil
	}

	list, err := items(id, in.Items)
	if err != nil {
		return nil, err
	}

	return gh.Repositories(rt).AssetMetadata.Upsert(ctx, list, now())
}

func UpsertBulkMetadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *MetadataBulkUpsertInput) ([]assetmetadata.Item, error) {
	ids := make([]uuid.UUID, 0, len(in.Items))
	for _, item := range in.Items {
		ids = append(ids, uuid.MustParse(item.AssetID))
	}
	if err := require(ctx, rt, a, permission.AssetUpdate, ids...); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(in.Items))
	list := make([]assetmetadata.Item, 0, len(in.Items))
	for i, item := range in.Items {
		key := "(" + item.AssetID + ", " + item.Key + ")"
		if seen[key] {
			return nil, duplicateItem(key)
		}
		seen[key] = true

		value, err := jsonb.Normalize(item.Value)
		if err != nil {
			return nil, err
		}
		list = append(list, assetmetadata.Item{AssetID: ids[i], Key: item.Key, Value: string(value)})
	}
	if len(list) == 0 {
		return nil, ErrEmptyBulkMetadata
	}

	return gh.Repositories(rt).AssetMetadata.Upsert(ctx, list, now())
}

func DeleteMetadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, key string) error {
	if err := require(ctx, rt, a, permission.AssetUpdate, id); err != nil {
		return err
	}

	return gh.Repositories(rt).AssetMetadata.Delete(ctx, []assetmetadata.Item{{AssetID: id, Key: key}}, now())
}

func DeleteBulkMetadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *MetadataBulkDeleteInput) error {
	list := make([]assetmetadata.Item, 0, len(in.Items))
	ids := make([]uuid.UUID, 0, len(in.Items))
	for _, item := range in.Items {
		id := uuid.MustParse(item.AssetID)
		ids = append(ids, id)
		list = append(list, assetmetadata.Item{AssetID: id, Key: item.Key})
	}
	if err := require(ctx, rt, a, permission.AssetUpdate, ids...); err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}

	return gh.Repositories(rt).AssetMetadata.Delete(ctx, list, now())
}
