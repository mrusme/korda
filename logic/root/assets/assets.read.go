package assets

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/tag"
	"tty.fail/mrus/korda/models/user"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Details struct {
	Asset *asset.Asset
	Exif  *assetexif.Exif
	Owner *user.User
	Stack *repoasset.StackSummary
	Tags  []tag.Tag
}

func Exif(ctx context.Context, rt *runtime.Runtime, assetID uuid.UUID) (*assetexif.Exif, error) {
	e, err := gh.Repositories(rt).AssetExif.Get(ctx, assetID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, nil
	}

	return e, err
}

func Load(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*Details, error) {
	a, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	d := &Details{Asset: a}
	if d.Exif, err = Exif(ctx, rt, id); err != nil {
		return nil, err
	}

	d.Owner, err = gh.Repositories(rt).User.Get(ctx, a.OwnerID, true)
	if errors.Is(err, errs.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return nil, err
	}

	if a.StackID.Valid {
		d.Stack, err = gh.Repositories(rt).Asset.StackSummary(ctx, a.StackID.UUID)
		if errors.Is(err, errs.ErrNoRows) {
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}

	tags, err := gh.Repositories(rt).Tag.ForAssets(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	d.Tags = tags[id]

	return d, nil
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Details, error) {
	if err := require(ctx, rt, a, permission.AssetRead, id); err != nil {
		return nil, err
	}

	return Load(ctx, rt, id)
}

func Statistics(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *StatisticsInput) (*repoasset.Statistics, error) {
	if in.Visibility != nil && *in.Visibility == asset.VisibilityLocked && !a.Elevated {
		return nil, errs.ErrElevatedRequired
	}

	return gh.Repositories(rt).Asset.Statistics(ctx, a.User.ID, repoasset.StatisticsFilter{
		Visibility: in.Visibility,
		IsFavorite: in.IsFavorite,
		IsTrashed:  in.IsTrashed,
	})
}
