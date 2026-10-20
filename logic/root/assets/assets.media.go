package assets

import (
	"context"
	"path/filepath"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Media struct {
	Path        string
	ContentType string
	FileName    string
	Redirect    MediaSize
}

func file(rt *runtime.Runtime, relative string, fileName string) (*Media, error) {
	abs, err := gh.MediaRoot(rt).Abs(relative)
	if err != nil {
		return nil, errs.ErrNotFound
	}

	return &Media{Path: abs, ContentType: mimetypes.Lookup(relative), FileName: fileName}, nil
}

func Thumbnail(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *MediaOptionsInput) (*Media, error) {
	if err := require(ctx, rt, a, permission.AssetView, id); err != nil {
		return nil, err
	}

	size := SizeThumbnail
	if in.Size != nil {
		size = *in.Size
	}
	edited := a.SharedLink != nil || in.Edited != nil && *in.Edited

	found, err := gh.Repositories(rt).Asset.ForThumbnail(ctx, id, string(size), edited)
	if err != nil {
		return nil, err
	}

	if size == SizeFullsize && mimetypes.IsWebSupportedImage(found.OriginalPath) && !edited {
		return &Media{Redirect: SizeOriginal}, nil
	}
	if size == SizeFullsize && !found.Path.Valid {
		return &Media{Redirect: SizePreview}, nil
	}
	if !found.Path.Valid {
		return nil, errs.ErrAssetMediaNotFound
	}

	base := mimetypes.Base(found.OriginalFileName)
	if a.SharedLink != nil && !a.SharedLink.ShowExif {
		base = id.String()
	}
	name := base + "_" + string(size) + filepath.Ext(found.Path.V)

	return file(rt, found.Path.V, name)
}

func Original(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *DownloadOriginalInput) (*Media, error) {
	if err := require(ctx, rt, a, permission.AssetDownload, id); err != nil {
		return nil, errs.ErrNotFound
	}

	found, err := gh.Repositories(rt).Asset.ForOriginal(ctx, id, a.SharedLink != nil || in.Edited != nil && *in.Edited)
	if err != nil {
		return nil, errs.ErrNotFound
	}

	path := found.OriginalPath
	if found.EditedPath.Valid {
		path = found.EditedPath.V
	}

	return file(rt, path, mimetypes.Base(found.OriginalFileName)+filepath.Ext(path))
}

func Playback(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Media, error) {
	if err := require(ctx, rt, a, permission.AssetView, id); err != nil {
		return nil, errs.ErrNotFound
	}

	found, err := gh.Repositories(rt).Asset.ForVideo(ctx, id)
	if err != nil {
		return nil, errs.ErrNotFound
	}

	path := found.OriginalPath
	if found.EncodedVideoPath.Valid && found.EncodedVideoPath.V != "" {
		path = found.EncodedVideoPath.V
	}

	return file(rt, path, "")
}
