package assets

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

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

func NewFile(assetID uuid.UUID, kind assetfile.Type, path string, at time.Time) *assetfile.File {
	return &assetfile.File{
		ID:        uuid.New(),
		AssetID:   assetID,
		Type:      kind,
		Path:      path,
		CreatedAt: at,
		UpdatedAt: at,
		UpdateID:  uuid.Must(uuid.NewV7()),
	}
}

func find(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*asset.Asset, error) {
	a, err := gh.Repositories(rt).Asset.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrAssetNotFound
	}

	return a, err
}

func require(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, p permission.Permission, ids ...uuid.UUID) error {
	return access.Require(ctx, rt, a, p, ids...)
}
