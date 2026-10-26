package people

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

var ErrOwnerID = errors.New("person owner id is not a UUID")

type Result struct {
	ID      string
	Success bool
	Error   bulk.Reason
}

func denied(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, p permission.Permission, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	parsed := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		parsed = append(parsed, uuid.MustParse(id))
	}

	return access.Require(ctx, rt, a, p, parsed...)
}

func Search(in *SearchInput) error {
	if in.ClosestPersonID != nil {
		return errs.ErrPersonNotFound
	}

	return nil
}

func Create(in *CreateInput) error {
	return errs.UnsupportedPeople
}

func update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id string, in *UpdateItem) error {
	personal := in.IsHidden != nil || in.IsFavorite != nil || in.Color.Set || in.FeatureFaceAssetID != nil
	if in.UserID != nil && *in.UserID != a.User.ID.String() && personal {
		return errs.ErrPersonOtherUser
	}

	return denied(ctx, rt, a, permission.PersonUpdate, id)
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id string, in *UpdateInput) error {
	return update(ctx, rt, a, id, &UpdateItem{CreateInput: in.CreateInput, FeatureFaceAssetID: in.FeatureFaceAssetID, UserID: in.UserID})
}

func UpdateAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UpdateAllInput) []Result {
	out := make([]Result, 0, len(in.People))
	for i := range in.People {
		item := &in.People[i]
		if err := update(ctx, rt, a, item.ID, item); err != nil {
			out = append(out, Result{ID: item.ID, Error: bulk.Unknown})
			continue
		}
		out = append(out, Result{ID: item.ID, Success: true})
	}

	return out
}

func DeleteAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, ids []string, userID *string) error {
	if len(ids) == 0 {
		return nil
	}
	if userID != nil {
		if _, err := uuid.Parse(*userID); err != nil {
			return ErrOwnerID
		}
	}

	return denied(ctx, rt, a, permission.PersonDelete, ids...)
}

func Read(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id string) error {
	return denied(ctx, rt, a, permission.PersonRead, id)
}

func Reassign(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id string, in *ReassignInput) error {
	return denied(ctx, rt, a, permission.PersonUpdate, id)
}

func Merge(ids []string) ([]Result, error) {
	if len(ids) < 2 {
		return nil, errs.ErrMergeTooFew
	}
	unique := slices.Clone(ids)
	slices.Sort(unique)
	if len(slices.Compact(unique)) != len(ids) {
		return nil, errs.ErrMergeSelf
	}

	out := make([]Result, 0, len(ids))
	for _, id := range ids {
		out = append(out, Result{ID: id, Error: bulk.NoPermission})
	}

	return out, nil
}

func Faces(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *FaceInput) error {
	return denied(ctx, rt, a, permission.AssetRead, in.ID)
}

func CreateFace(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *FaceCreateInput) error {
	if err := denied(ctx, rt, a, permission.AssetUpdate, in.AssetID); err != nil {
		return err
	}

	return errs.UnsupportedPeople
}

func DeleteFace(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id string) error {
	return denied(ctx, rt, a, permission.FaceDelete, id)
}

func UpsertUsers(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UsersUpsertInput) error {
	if slices.Contains(in.SharedWithIDs, a.User.ID.String()) {
		return errs.ErrShareWithSelf
	}
	if err := denied(ctx, rt, a, permission.PersonUpdate, in.PersonIDs...); err != nil {
		return err
	}
	if len(in.SharedWithIDs) > 0 {
		return errs.ErrClusterGroup
	}

	return nil
}

func RemoveUsers(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in UsersDeleteInput) error {
	ids := make([]string, 0, len(in))
	for _, item := range in {
		ids = append(ids, item.PersonID)
	}

	return denied(ctx, rt, a, permission.PersonUpdate, ids...)
}
