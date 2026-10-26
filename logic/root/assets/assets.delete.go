package assets

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/permission"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

func DeleteAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BulkDeleteInput) error {
	ids := parseIDs(in.IDs)
	if err := require(ctx, rt, a, permission.AssetDelete, ids...); err != nil {
		return err
	}

	force := in.Force != nil && *in.Force
	status := asset.StatusTrashed
	if force {
		status = asset.StatusDeleted
	}

	at := now()
	if err := gh.Repositories(rt).Asset.UpdateAll(ctx, ids, repoasset.Update{
		DeletedAt: optional.Of(at),
		Status:    optional.Of(status),
	}, at); err != nil {
		return err
	}

	if force {
		return gh.Jobs(rt).EmptyTrash()
	}
	gh.Emit(ctx, rt, event.AssetsTrashed{UserID: a.User.ID, AssetIDs: ids})

	return nil
}

func QueueDeleted(ctx context.Context, rt *runtime.Runtime) error {
	ids, err := gh.Repositories(rt).Asset.Deleted(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err = gh.Jobs(rt).DeleteAsset(id, true); err != nil {
			return err
		}
	}

	return nil
}

func QueueExpired(ctx context.Context, rt *runtime.Runtime, trashDays int) error {
	before := time.Now().UTC().AddDate(0, 0, -trashDays)
	expired, err := gh.Repositories(rt).Asset.Expired(ctx, before)
	if err != nil {
		return err
	}

	for _, e := range expired {
		if err = gh.Jobs(rt).DeleteAsset(e.ID, !e.IsOffline); err != nil {
			return err
		}
	}
	if len(expired) > 0 {
		rt.Info("assets", "trash", "queued expired assets", len(expired))
	}

	return nil
}

func Purge(ctx context.Context, rt *runtime.Runtime, id uuid.UUID, deleteOnDisk bool) error {
	repos := gh.Repositories(rt)

	removed, err := repos.Asset.Remove(ctx, id, now())
	if errors.Is(err, errs.ErrNoRows) {
		rt.Warn("assets", "purge", "asset not found", id.String())
		return nil
	}
	if err != nil {
		return err
	}
	gh.Emit(ctx, rt, event.AssetDeleted{UserID: removed.Asset.OwnerID, AssetID: id})

	if motion := removed.Asset.LivePhotoVideoID; motion.Valid {
		count, err := repos.Asset.LivePhotoCount(ctx, motion.UUID)
		if err != nil {
			return err
		}
		if count == 0 {
			if err = gh.Jobs(rt).DeleteAsset(motion.UUID, deleteOnDisk); err != nil {
				return err
			}
		}
	}

	var paths []string
	for _, f := range removed.Files {
		if f.Type != assetfile.TypeSidecar {
			paths = append(paths, f.Path)
		}
	}
	if deleteOnDisk && !removed.Asset.IsOffline {
		for _, f := range removed.Files {
			if f.Type == assetfile.TypeSidecar {
				paths = append(paths, f.Path)
			}
		}
		paths = append(paths, removed.Asset.OriginalPath)
	}

	if err = gh.MediaRoot(rt).Remove(paths...); err != nil {
		rt.Warn("assets", "purge", "asset", id.String(), "error", err)
	}

	return nil
}
