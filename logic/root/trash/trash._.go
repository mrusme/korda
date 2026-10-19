package trash

import (
	"context"
	"time"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

type BulkIDsInput struct {
	IDs []string `json:"ids" vi:"uuid"`
}

func now() time.Time {
	return time.Now().UTC()
}

func Empty(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (int64, error) {
	count, err := gh.Repositories(rt).Asset.EmptyTrash(ctx, a.User.ID, now())
	if err != nil {
		return 0, err
	}
	if count > 0 {
		if err = gh.Jobs(rt).EmptyTrash(); err != nil {
			return 0, err
		}
	}

	return count, nil
}

func Restore(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (int64, error) {
	return gh.Repositories(rt).Asset.RestoreTrash(ctx, a.User.ID, now())
}

func RestoreAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BulkIDsInput) (int64, error) {
	if len(in.IDs) == 0 {
		return 0, nil
	}

	ids := make([]uuid.UUID, 0, len(in.IDs))
	for _, id := range in.IDs {
		if parsed, err := uuid.Parse(id); err == nil {
			ids = append(ids, parsed)
		}
	}

	if err := access.Require(ctx, rt, a, permission.AssetDelete, ids...); err != nil {
		return 0, err
	}
	if err := gh.Repositories(rt).Asset.RestoreAll(ctx, ids, now()); err != nil {
		return 0, err
	}

	return int64(len(in.IDs)), nil
}
