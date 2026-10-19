package access

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Require(
	ctx context.Context,
	rt *runtime.Runtime,
	a *auth.Auth,
	p permission.Permission,
	ids ...uuid.UUID,
) error {
	allowed, err := check(ctx, rt, a, p, ids)
	if err != nil {
		return err
	}

	if !sameSet(allowed, ids) {
		return fmt.Errorf("%w %s access", errs.ErrNoAccess, p)
	}

	return nil
}

func check(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, p permission.Permission, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	switch p {
	case permission.AuthDeviceDelete, permission.SessionUpdate, permission.SessionLock:
		return gh.Repositories(rt).Session.OwnedBy(ctx, a.User.ID, ids)

	case permission.AssetRead, permission.AssetView, permission.AssetDownload,
		permission.AssetUpdate, permission.AssetDelete, permission.AssetCopy,
		permission.AssetEditGet, permission.AssetEditCreate, permission.AssetEditDelete:
		return gh.Repositories(rt).Asset.OwnedBy(ctx, a.User.ID, ids, a.Elevated)

	case permission.AssetShare:
		return gh.Repositories(rt).Asset.OwnedBy(ctx, a.User.ID, ids, false)

	case permission.AssetUpload:
		if slices.Contains(ids, a.User.ID) {
			return []uuid.UUID{a.User.ID}, nil
		}
	}

	return nil, nil
}

func sameSet(a []uuid.UUID, b []uuid.UUID) bool {
	set := make(map[uuid.UUID]bool, len(a))
	for _, id := range a {
		set[id] = true
	}

	other := make(map[uuid.UUID]bool, len(b))
	for _, id := range b {
		if !set[id] {
			return false
		}
		other[id] = true
	}

	return len(set) == len(other)
}
