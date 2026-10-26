package assetfiles

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	repoassetfile "tty.fail/mrus/korda/services/repositories/assetfile"
	"xn--gckvb8fzb.com/glides/runtime"
)

type SearchInput struct {
	AssetID       string          `json:"assetId" v:"uuid"`
	Type          *assetfile.Type `json:"type"`
	IsEdited      *bool           `json:"isEdited"`
	IsProgressive *bool           `json:"isProgressive"`
	IsTransparent *bool           `json:"isTransparent"`
}

type Download struct {
	Abs         string
	FileName    string
	ContentType string
}

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]assetfile.File, error) {
	assetID := uuid.MustParse(in.AssetID)
	if err := access.Require(ctx, rt, a, permission.AssetRead, assetID); err != nil {
		return nil, err
	}

	return gh.Repositories(rt).AssetFile.Search(ctx, repoassetfile.Filter{
		AssetID: assetID, Type: in.Type, IsEdited: in.IsEdited, IsProgressive: in.IsProgressive, IsTransparent: in.IsTransparent,
	})
}

func find(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, p permission.Permission, id uuid.UUID) (*assetfile.File, error) {
	if err := access.Require(ctx, rt, a, p, id); err != nil {
		return nil, err
	}

	f, err := gh.Repositories(rt).AssetFile.ByID(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrAssetFileNotFound
	}

	return f, err
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*assetfile.File, error) {
	return find(ctx, rt, a, permission.AssetFileRead, id)
}

func Prepare(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Download, error) {
	f, err := find(ctx, rt, a, permission.AssetFileDownload, id)
	if err != nil {
		return nil, err
	}

	abs, err := gh.MediaRoot(rt).Abs(f.Path)
	if err != nil {
		return nil, err
	}

	return &Download{Abs: abs, FileName: filepath.Base(f.Path), ContentType: mimetypes.Lookup(f.Path)}, nil
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	f, err := find(ctx, rt, a, permission.AssetFileDelete, id)
	if err != nil {
		return err
	}
	if f.Type == assetfile.TypeSidecar {
		return errs.ErrSidecarNotDeletable
	}

	if err = gh.Repositories(rt).AssetFile.DeleteByID(ctx, id); err != nil {
		return err
	}
	if err = gh.MediaRoot(rt).Remove(f.Path); err != nil {
		rt.Warn("assetfiles", "delete", "path", f.Path, "error", err)
	}

	return nil
}
