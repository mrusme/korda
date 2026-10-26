package stacks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/stack"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

var ErrNoStacks = errors.New("no stacks to delete")

type SearchInput struct {
	PrimaryAssetID *string `json:"primaryAssetId" v:"uuid"`
}

type CreateInput struct {
	AssetIDs []string `json:"assetIds" v:"min=2" vi:"uuid"`
}

type UpdateInput struct {
	PrimaryAssetID *string `json:"primaryAssetId" v:"uuid"`
}

type BulkIDsInput struct {
	IDs []string `json:"ids" vi:"uuid"`
}

type RemoveAssetInput struct {
	ID      string `json:"id" v:"uuid"`
	AssetID string `json:"assetId" v:"uuid"`
}

type Details struct {
	Stack    *stack.Stack
	Assets   []asset.Asset
	WithTags bool
}

func now() time.Time {
	return time.Now().UTC()
}

func parseIDs(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if parsed, err := uuid.Parse(id); err == nil {
			out = append(out, parsed)
		}
	}

	return out
}

func load(ctx context.Context, rt *runtime.Runtime, s *stack.Stack, withTags bool) (*Details, error) {
	members, err := gh.Repositories(rt).Asset.StackMembers(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	ordered := make([]asset.Asset, 0, len(members))
	for _, m := range members {
		if m.ID == s.PrimaryAssetID {
			ordered = append(ordered, m)
		}
	}
	for _, m := range members {
		if m.ID != s.PrimaryAssetID {
			ordered = append(ordered, m)
		}
	}

	return &Details{Stack: s, Assets: ordered, WithTags: withTags}, nil
}

func find(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*stack.Stack, error) {
	s, err := gh.Repositories(rt).Stack.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrStackNotFound
	}

	return s, err
}

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]Details, error) {
	var primary uuid.NullUUID
	if in.PrimaryAssetID != nil {
		primary = uuid.NullUUID{UUID: uuid.MustParse(*in.PrimaryAssetID), Valid: true}
	}

	list, err := gh.Repositories(rt).Stack.Search(ctx, a.User.ID, primary)
	if err != nil {
		return nil, err
	}

	out := make([]Details, 0, len(list))
	for i := range list {
		d, err := load(ctx, rt, &list[i], false)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}

	return out, nil
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*Details, error) {
	ids := parseIDs(in.AssetIDs)
	if err := access.Require(ctx, rt, a, permission.AssetUpdate, ids...); err != nil {
		return nil, err
	}

	s, err := gh.Repositories(rt).Stack.Create(ctx, a.User.ID, ids, now())
	if err != nil {
		return nil, err
	}
	gh.Emit(ctx, rt, event.StackChanged{UserID: a.User.ID})

	return load(ctx, rt, s, false)
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.StackRead, id); err != nil {
		return nil, err
	}

	s, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	return load(ctx, rt, s, true)
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.StackUpdate, id); err != nil {
		return nil, err
	}

	s, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	var primary uuid.NullUUID
	if in.PrimaryAssetID != nil {
		primary = uuid.NullUUID{UUID: uuid.MustParse(*in.PrimaryAssetID), Valid: true}
		current, err := load(ctx, rt, s, false)
		if err != nil {
			return nil, err
		}
		found := false
		for _, m := range current.Assets {
			found = found || m.ID == primary.UUID
		}
		if !found {
			return nil, errs.ErrPrimaryNotInStack
		}
	}

	updated, err := gh.Repositories(rt).Stack.Update(ctx, id, primary, now())
	if err != nil {
		return nil, err
	}
	gh.Emit(ctx, rt, event.StackChanged{UserID: a.User.ID})

	return load(ctx, rt, updated, true)
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.StackDelete, id); err != nil {
		return err
	}

	return changed(ctx, rt, a, gh.Repositories(rt).Stack.Delete(ctx, []uuid.UUID{id}, now()))
}

func changed(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, err error) error {
	if err == nil {
		gh.Emit(ctx, rt, event.StackChanged{UserID: a.User.ID})
	}

	return err
}

func DeleteAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BulkIDsInput) error {
	ids := parseIDs(in.IDs)
	if err := access.Require(ctx, rt, a, permission.StackDelete, ids...); err != nil {
		return err
	}
	if len(ids) == 0 {
		return ErrNoStacks
	}

	return changed(ctx, rt, a, gh.Repositories(rt).Stack.Delete(ctx, ids, now()))
}

func RemoveAsset(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *RemoveAssetInput) error {
	stackID, assetID := uuid.MustParse(in.ID), uuid.MustParse(in.AssetID)
	if err := access.Require(ctx, rt, a, permission.StackUpdate, stackID); err != nil {
		return err
	}

	repos := gh.Repositories(rt)
	current, primary, err := repos.Asset.StackOf(ctx, assetID)
	if errors.Is(err, errs.ErrNoRows) || (err == nil && (!current.Valid || current.UUID != stackID)) {
		return errs.ErrAssetNotInStack
	}
	if err != nil {
		return err
	}
	if primary.Valid && primary.UUID == assetID {
		return errs.ErrRemovePrimaryAsset
	}

	_, err = repos.Asset.Update(ctx, assetID, repoasset.Update{StackID: optional.Value[uuid.UUID]{Set: true, Null: true}}, now())

	return changed(ctx, rt, a, err)
}
