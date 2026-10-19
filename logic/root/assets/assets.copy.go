package assets

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/services/jobs"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

func copyFile(rt *runtime.Runtime, from string, to string) error {
	root := gh.MediaRoot(rt)
	src, err := root.Abs(from)
	if err != nil {
		return err
	}
	dst, err := root.Abs(to)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}

func Copy(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CopyInput) error {
	sourceID, targetID := uuid.MustParse(in.SourceID), uuid.MustParse(in.TargetID)
	if err := require(ctx, rt, a, permission.AssetCopy, sourceID, targetID); err != nil {
		return err
	}

	repos := gh.Repositories(rt)
	source, err := repos.Asset.Get(ctx, sourceID)
	if errors.Is(err, errs.ErrNoRows) {
		return errs.ErrBothAssetsMustExist
	}
	if err != nil {
		return err
	}
	target, err := repos.Asset.Get(ctx, targetID)
	if errors.Is(err, errs.ErrNoRows) {
		return errs.ErrBothAssetsMustExist
	}
	if err != nil {
		return err
	}
	if sourceID == targetID {
		return errs.ErrSameSourceAndTarget
	}

	at := now()
	if in.Favorite == nil || *in.Favorite {
		if _, err = repos.Asset.Update(ctx, targetID, repoasset.Update{IsFavorite: optional.Of(source.IsFavorite)}, at); err != nil {
			return err
		}
	}

	if in.Sidecar == nil || *in.Sidecar {
		files, err := repos.AssetFile.List(ctx, sourceID)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.Type != assetfile.TypeSidecar {
				continue
			}

			targetFiles, err := repos.AssetFile.List(ctx, targetID)
			if err != nil {
				return err
			}
			for _, tf := range targetFiles {
				if tf.Type == assetfile.TypeSidecar {
					if err = gh.MediaRoot(rt).Remove(tf.Path); err != nil {
						rt.Warn("assets", "copy", "remove sidecar", tf.Path, "error", err)
					}
				}
			}

			path := target.OriginalPath + ".xmp"
			if err = copyFile(rt, f.Path, path); err != nil {
				return err
			}
			if _, err = repos.AssetFile.Upsert(ctx, NewFile(targetID, assetfile.TypeSidecar, path, at)); err != nil {
				return err
			}
			if err = gh.Jobs(rt).Metadata(jobs.AssetPayload{AssetID: targetID}); err != nil {
				return err
			}
			break
		}
	}

	return nil
}
