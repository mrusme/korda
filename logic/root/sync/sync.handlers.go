package sync

import (
	"context"

	modelsync "tty.fail/mrus/korda/models/sync"
	reposync "tty.fail/mrus/korda/services/repositories/sync"
)

type handler func(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error

var handlers = map[modelsync.RequestType]handler{
	modelsync.ReqAuthUsersV1:     authUsers(modelsync.AuthUserV1),
	modelsync.ReqAuthUsersV2:     authUsers(modelsync.AuthUserV2),
	modelsync.ReqUsersV1:         users,
	modelsync.ReqAssetsV2:        assets,
	modelsync.ReqAssetExifsV1:    assetExifs,
	modelsync.ReqUserMetadataV1:  userMetadata,
	modelsync.ReqAssetMetadataV1: assetMetadata,
}

func authUsers(t modelsync.EntityType) handler {
	return func(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
		return page(t, p, func(b reposync.Bounds) ([]reposync.AuthUser, error) {
			return repo.AuthUsers(ctx, p.userID, b)
		}, func(r reposync.AuthUser) string { return r.UpdateID }, identity, emit)
	}
}

func users(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.UserDeleteV1, p, func(b reposync.Bounds) ([]reposync.UserDelete, error) {
		return repo.UserDeletes(ctx, b)
	}, func(r reposync.UserDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.UserV1, p, func(b reposync.Bounds) ([]reposync.User, error) {
		return repo.Users(ctx, b)
	}, func(r reposync.User) string { return r.UpdateID }, identity, emit)
}

func assets(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AssetDeleteV1, p, func(b reposync.Bounds) ([]reposync.AssetDelete, error) {
		return repo.AssetDeletes(ctx, p.userID, b)
	}, func(r reposync.AssetDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.AssetV2, p, func(b reposync.Bounds) ([]reposync.Asset, error) {
		return repo.Assets(ctx, p.userID, b)
	}, func(r reposync.Asset) string { return r.UpdateID }, identity, emit)
}

func assetExifs(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	return page(modelsync.AssetExifV1, p, func(b reposync.Bounds) ([]reposync.AssetExif, error) {
		return repo.AssetExifs(ctx, p.userID, b)
	}, func(r reposync.AssetExif) string { return r.UpdateID }, identity, emit)
}

func userMetadata(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.UserMetadataDeleteV1, p, func(b reposync.Bounds) ([]reposync.UserMetadataDelete, error) {
		return repo.UserMetadataDeletes(ctx, p.userID, b)
	}, func(r reposync.UserMetadataDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.UserMetadataV1, p, func(b reposync.Bounds) ([]reposync.UserMetadata, error) {
		return repo.UserMetadata(ctx, p.userID, b)
	}, func(r reposync.UserMetadata) string { return r.UpdateID }, identity, emit)
}

func assetMetadata(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AssetMetadataDeleteV1, p, func(b reposync.Bounds) ([]reposync.AssetMetadataDelete, error) {
		return repo.AssetMetadataDeletes(ctx, p.userID, b)
	}, func(r reposync.AssetMetadataDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.AssetMetadataV1, p, func(b reposync.Bounds) ([]reposync.AssetMetadata, error) {
		return repo.AssetMetadata(ctx, p.userID, b)
	}, func(r reposync.AssetMetadata) string { return r.UpdateID }, identity, emit)
}
