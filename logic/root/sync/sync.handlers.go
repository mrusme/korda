package sync

import (
	"context"

	"github.com/google/uuid"
	modelsync "tty.fail/mrus/korda/models/sync"
	reposync "tty.fail/mrus/korda/services/repositories/sync"
	"xn--gckvb8fzb.com/glides/runtime"
)

type handler func(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error

var handlers = map[modelsync.RequestType]handler{
	modelsync.ReqAuthUsersV1:         authUsers(modelsync.AuthUserV1),
	modelsync.ReqAuthUsersV2:         authUsers(modelsync.AuthUserV2),
	modelsync.ReqUsersV1:             users,
	modelsync.ReqPartnersV1:          partners,
	modelsync.ReqAssetsV2:            assets,
	modelsync.ReqStacksV1:            stacks,
	modelsync.ReqPartnerAssetsV2:     partnerAssets,
	modelsync.ReqPartnerStacksV1:     partnerStacks,
	modelsync.ReqAlbumAssetsV2:       albumAssets,
	modelsync.ReqAlbumsV1:            albumsV1,
	modelsync.ReqAlbumsV2:            albumsV2,
	modelsync.ReqAlbumUsersV1:        albumUsers,
	modelsync.ReqAlbumToAssetsV1:     albumToAssets,
	modelsync.ReqAssetExifsV1:        assetExifs,
	modelsync.ReqAlbumAssetExifsV1:   albumAssetExifs,
	modelsync.ReqPartnerAssetExifsV1: partnerAssetExifs,
	modelsync.ReqMemoriesV1:          memories(modelsync.MemoryV1, true),
	modelsync.ReqMemoriesV2:          memories(modelsync.MemoryV2, false),
	modelsync.ReqMemoryToAssetsV1:    memoryAssets(modelsync.MemoryToAssetV1, true),
	modelsync.ReqMemoryToAssetsV2:    memoryAssets(modelsync.MemoryToAssetV2, false),
	modelsync.ReqUserMetadataV1:      userMetadata,
	modelsync.ReqAssetMetadataV1:     assetMetadata,
	modelsync.ReqAssetEditsV1:        assetEdits,
}

func authUsers(t modelsync.EntityType) handler {
	return func(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
		return page(t, p, func(b reposync.Bounds) ([]reposync.AuthUser, error) {
			return repo.AuthUsers(ctx, p.userID, b)
		}, func(r reposync.AuthUser) string { return r.UpdateID }, identity, emit)
	}
}

func users(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.UserDeleteV1, p, func(b reposync.Bounds) ([]reposync.UserDelete, error) {
		return repo.UserDeletes(ctx, b)
	}, func(r reposync.UserDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.UserV1, p, func(b reposync.Bounds) ([]reposync.User, error) {
		return repo.Users(ctx, b)
	}, func(r reposync.User) string { return r.UpdateID }, identity, emit)
}

func partners(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.PartnerDeleteV1, p, func(b reposync.Bounds) ([]reposync.PartnerDelete, error) {
		return repo.PartnerDeletes(ctx, p.userID, b)
	}, func(r reposync.PartnerDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.PartnerV1, p, func(b reposync.Bounds) ([]reposync.Partner, error) {
		return repo.Partners(ctx, p.userID, b)
	}, func(r reposync.Partner) string { return r.UpdateID }, identity, emit)
}

func assets(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AssetDeleteV1, p, func(b reposync.Bounds) ([]reposync.AssetDelete, error) {
		return repo.AssetDeletes(ctx, p.userID, b)
	}, func(r reposync.AssetDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.AssetV2, p, func(b reposync.Bounds) ([]reposync.Asset, error) {
		return repo.Assets(ctx, p.userID, b)
	}, func(r reposync.Asset) string { return r.UpdateID }, identity, emit)
}

func stacks(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.StackDeleteV1, p, func(b reposync.Bounds) ([]reposync.StackDelete, error) {
		return repo.StackDeletes(ctx, p.userID, b)
	}, func(r reposync.StackDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.StackV1, p, func(b reposync.Bounds) ([]reposync.Stack, error) {
		return repo.Stacks(ctx, p.userID, b)
	}, func(r reposync.Stack) string { return r.UpdateID }, identity, emit)
}

func partnerMembers(ctx context.Context, repo *reposync.Repository, p *Plan) func(*string) ([]reposync.Membership, error) {
	return func(after *string) ([]reposync.Membership, error) {
		return repo.PartnersCreatedAfter(ctx, p.userID, after, p.nowID)
	}
}

func albumMembers(ctx context.Context, repo *reposync.Repository, p *Plan) func(*string) ([]reposync.Membership, error) {
	return func(after *string) ([]reposync.Membership, error) {
		return repo.AlbumsCreatedAfter(ctx, p.userID, after, p.nowID)
	}
}

func partnerAssets(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.PartnerAssetDeleteV1, p, func(b reposync.Bounds) ([]reposync.AssetDelete, error) {
		return repo.PartnerAssetDeletes(ctx, p.userID, b)
	}, func(r reposync.AssetDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	if err := (backfill[reposync.PartnerAsset]{
		kind:    modelsync.PartnerAssetBackfillV2,
		bound:   modelsync.PartnerAssetV2,
		members: partnerMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.PartnerAsset, error) {
			return repo.PartnerAssetsBackfill(ctx, m.ID, b)
		},
		key: func(r reposync.PartnerAsset) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	return page(modelsync.PartnerAssetV2, p, func(b reposync.Bounds) ([]reposync.PartnerAsset, error) {
		return repo.PartnerAssets(ctx, p.userID, b)
	}, func(r reposync.PartnerAsset) string { return r.UpdateID }, identity, emit)
}

func partnerStacks(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.PartnerStackDeleteV1, p, func(b reposync.Bounds) ([]reposync.StackDelete, error) {
		return repo.PartnerStackDeletes(ctx, p.userID, b)
	}, func(r reposync.StackDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	if err := (backfill[reposync.Stack]{
		kind:    modelsync.PartnerStackBackfillV1,
		bound:   modelsync.PartnerStackV1,
		members: partnerMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.Stack, error) {
			return repo.PartnerStacksBackfill(ctx, m.ID, b)
		},
		key: func(r reposync.Stack) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	return page(modelsync.PartnerStackV1, p, func(b reposync.Bounds) ([]reposync.Stack, error) {
		return repo.PartnerStacks(ctx, p.userID, b)
	}, func(r reposync.Stack) string { return r.UpdateID }, identity, emit)
}

func partnerAssetExifs(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := (backfill[reposync.AssetExif]{
		kind:    modelsync.PartnerAssetExifBackfillV1,
		bound:   modelsync.PartnerAssetExifV1,
		members: partnerMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.AssetExif, error) {
			return repo.PartnerAssetExifsBackfill(ctx, m.ID, b)
		},
		key: func(r reposync.AssetExif) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	return page(modelsync.PartnerAssetExifV1, p, func(b reposync.Bounds) ([]reposync.AssetExif, error) {
		return repo.PartnerAssetExifs(ctx, p.userID, b)
	}, func(r reposync.AssetExif) string { return r.UpdateID }, identity, emit)
}

func pageCursor[T any](
	t modelsync.EntityType,
	p *Plan,
	fetch func(b reposync.Bounds, c *reposync.Cursor) ([]T, error),
	cursor func(T) reposync.Cursor,
	emit func(Line) error,
) error {
	b := reposync.Bounds{After: p.after(t), Before: p.nowID, Limit: pageSize}
	var c *reposync.Cursor
	for {
		rows, err := fetch(b, c)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err = emit(Line{Type: t, Data: row, Ack: modelsync.FormatAck(t, cursor(row).UpdateID)}); err != nil {
				return err
			}
		}
		if len(rows) < pageSize {
			return nil
		}
		last := cursor(rows[len(rows)-1])
		c = &last
	}
}

func albumAssets(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := (backfill[reposync.AlbumAsset]{
		kind:    modelsync.AlbumAssetBackfillV2,
		bound:   modelsync.AlbumAssetCreateV2,
		members: albumMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.AlbumAsset, error) {
			return repo.AlbumAssetsBackfill(ctx, p.userID, m.ID, b)
		},
		key: func(r reposync.AlbumAsset) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	if created := p.after(modelsync.AlbumAssetCreateV2); created != nil {
		if err := pageCursor(modelsync.AlbumAssetUpdateV2, p,
			func(b reposync.Bounds, c *reposync.Cursor) ([]reposync.AlbumAssetUpdate, error) {
				return repo.AlbumAssetUpdates(ctx, p.userID, *created, b, c)
			},
			func(r reposync.AlbumAssetUpdate) reposync.Cursor {
				return reposync.Cursor{UpdateID: r.UpdateID, AlbumID: r.PageAlbumID}
			}, emit); err != nil {
			return err
		}
	}

	first := Line{Type: modelsync.SyncAckV1, Ack: modelsync.FormatAck(modelsync.AlbumAssetUpdateV2, p.nowID)}

	return page(modelsync.AlbumAssetCreateV2, p, func(b reposync.Bounds) ([]reposync.AlbumAsset, error) {
		return repo.AlbumAssetCreates(ctx, p.userID, b)
	}, func(r reposync.AlbumAsset) string { return r.UpdateID }, identity, firstOnly(emit, first))
}

func albumAssetExifs(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := (backfill[reposync.AssetExif]{
		kind:    modelsync.AlbumAssetExifBackfillV1,
		bound:   modelsync.AlbumAssetExifCreateV1,
		members: albumMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.AssetExif, error) {
			return repo.AlbumAssetExifsBackfill(ctx, m.ID, b)
		},
		key: func(r reposync.AssetExif) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	if created := p.after(modelsync.AlbumAssetExifCreateV1); created != nil {
		if err := pageCursor(modelsync.AlbumAssetExifUpdateV1, p,
			func(b reposync.Bounds, c *reposync.Cursor) ([]reposync.AlbumAssetExifUpdate, error) {
				return repo.AlbumAssetExifUpdates(ctx, p.userID, *created, b, c)
			},
			func(r reposync.AlbumAssetExifUpdate) reposync.Cursor {
				return reposync.Cursor{UpdateID: r.UpdateID, AlbumID: r.PageAlbumID}
			}, emit); err != nil {
			return err
		}
	}

	first := Line{Type: modelsync.SyncAckV1, Ack: modelsync.FormatAck(modelsync.AlbumAssetExifUpdateV1, p.nowID)}

	return page(modelsync.AlbumAssetExifCreateV1, p, func(b reposync.Bounds) ([]reposync.AssetExif, error) {
		return repo.AlbumAssetExifCreates(ctx, p.userID, b)
	}, func(r reposync.AssetExif) string { return r.UpdateID }, identity, firstOnly(emit, first))
}

func albumDeletes(ctx context.Context, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	return page(modelsync.AlbumDeleteV1, p, func(b reposync.Bounds) ([]reposync.AlbumDelete, error) {
		return repo.AlbumDeletes(ctx, p.userID, b)
	}, func(r reposync.AlbumDelete) string { return r.ID }, identity, emit)
}

func albumsV1(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := albumDeletes(ctx, repo, p, emit); err != nil {
		return err
	}

	owners := map[uuid.UUID]uuid.UUID{}
	return page(modelsync.AlbumV1, p, func(b reposync.Bounds) ([]reposync.AlbumV1, error) {
		albums, err := repo.Albums(ctx, p.userID, b)
		if err != nil {
			return nil, err
		}
		out := make([]reposync.AlbumV1, 0, len(albums))
		for _, a := range albums {
			owner, ok := owners[a.ID]
			if !ok {
				if owner, err = repo.AlbumOwner(ctx, a.ID); err != nil {
					return nil, err
				}
				owners[a.ID] = owner
			}
			out = append(out, reposync.AlbumV1{Album: a, OwnerID: owner})
		}
		return out, nil
	}, func(r reposync.AlbumV1) string { return r.UpdateID }, identity, emit)
}

func albumsV2(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := albumDeletes(ctx, repo, p, emit); err != nil {
		return err
	}

	return page(modelsync.AlbumV2, p, func(b reposync.Bounds) ([]reposync.Album, error) {
		return repo.Albums(ctx, p.userID, b)
	}, func(r reposync.Album) string { return r.UpdateID }, identity, emit)
}

func albumUsers(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AlbumUserDeleteV1, p, func(b reposync.Bounds) ([]reposync.AlbumUserDelete, error) {
		return repo.AlbumUserDeletes(ctx, p.userID, b)
	}, func(r reposync.AlbumUserDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	if err := (backfill[reposync.AlbumUser]{
		kind:    modelsync.AlbumUserBackfillV1,
		bound:   modelsync.AlbumUserV1,
		members: albumMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.AlbumUser, error) {
			return repo.AlbumUsersBackfill(ctx, m.ID, b)
		},
		key: func(r reposync.AlbumUser) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	return page(modelsync.AlbumUserV1, p, func(b reposync.Bounds) ([]reposync.AlbumUser, error) {
		return repo.AlbumUsers(ctx, p.userID, b)
	}, func(r reposync.AlbumUser) string { return r.UpdateID }, identity, emit)
}

func albumToAssets(ctx context.Context, rt *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AlbumToAssetDeleteV1, p, func(b reposync.Bounds) ([]reposync.AlbumToAssetDelete, error) {
		return repo.AlbumToAssetDeletes(ctx, p.userID, b)
	}, func(r reposync.AlbumToAssetDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	if err := (backfill[reposync.AlbumToAsset]{
		kind:    modelsync.AlbumToAssetBackfillV1,
		bound:   modelsync.AlbumToAssetV1,
		members: albumMembers(ctx, repo, p),
		rows: func(m reposync.Membership, b reposync.Backfill) ([]reposync.AlbumToAsset, error) {
			return repo.AlbumToAssetsBackfill(ctx, m.ID, b)
		},
		key: func(r reposync.AlbumToAsset) string { return r.UpdateID },
	}).run(ctx, rt, p, emit); err != nil {
		return err
	}

	return page(modelsync.AlbumToAssetV1, p, func(b reposync.Bounds) ([]reposync.AlbumToAsset, error) {
		return repo.AlbumToAssets(ctx, p.userID, b)
	}, func(r reposync.AlbumToAsset) string { return r.UpdateID }, identity, emit)
}

func assetExifs(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	return page(modelsync.AssetExifV1, p, func(b reposync.Bounds) ([]reposync.AssetExif, error) {
		return repo.AssetExifs(ctx, p.userID, b)
	}, func(r reposync.AssetExif) string { return r.UpdateID }, identity, emit)
}

func memories(t modelsync.EntityType, onThisDayOnly bool) handler {
	return func(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
		if err := page(modelsync.MemoryDeleteV1, p, func(b reposync.Bounds) ([]reposync.MemoryDelete, error) {
			return repo.MemoryDeletes(ctx, p.userID, b)
		}, func(r reposync.MemoryDelete) string { return r.ID }, identity, emit); err != nil {
			return err
		}

		return page(t, p, func(b reposync.Bounds) ([]reposync.Memory, error) {
			return repo.Memories(ctx, p.userID, onThisDayOnly, b)
		}, func(r reposync.Memory) string { return r.UpdateID }, identity, emit)
	}
}

func memoryAssets(t modelsync.EntityType, onThisDayOnly bool) handler {
	return func(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
		if err := page(modelsync.MemoryToAssetDeleteV1, p, func(b reposync.Bounds) ([]reposync.MemoryAssetDelete, error) {
			return repo.MemoryAssetDeletes(ctx, p.userID, b)
		}, func(r reposync.MemoryAssetDelete) string { return r.ID }, identity, emit); err != nil {
			return err
		}

		return page(t, p, func(b reposync.Bounds) ([]reposync.MemoryAsset, error) {
			return repo.MemoryAssets(ctx, p.userID, onThisDayOnly, b)
		}, func(r reposync.MemoryAsset) string { return r.UpdateID }, identity, emit)
	}
}

func userMetadata(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.UserMetadataDeleteV1, p, func(b reposync.Bounds) ([]reposync.UserMetadataDelete, error) {
		return repo.UserMetadataDeletes(ctx, p.userID, b)
	}, func(r reposync.UserMetadataDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.UserMetadataV1, p, func(b reposync.Bounds) ([]reposync.UserMetadata, error) {
		return repo.UserMetadata(ctx, p.userID, b)
	}, func(r reposync.UserMetadata) string { return r.UpdateID }, identity, emit)
}

func assetMetadata(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AssetMetadataDeleteV1, p, func(b reposync.Bounds) ([]reposync.AssetMetadataDelete, error) {
		return repo.AssetMetadataDeletes(ctx, p.userID, b)
	}, func(r reposync.AssetMetadataDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.AssetMetadataV1, p, func(b reposync.Bounds) ([]reposync.AssetMetadata, error) {
		return repo.AssetMetadata(ctx, p.userID, b)
	}, func(r reposync.AssetMetadata) string { return r.UpdateID }, identity, emit)
}

func assetEdits(ctx context.Context, _ *runtime.Runtime, repo *reposync.Repository, p *Plan, emit func(Line) error) error {
	if err := page(modelsync.AssetEditDeleteV1, p, func(b reposync.Bounds) ([]reposync.AssetEditDelete, error) {
		return repo.AssetEditDeletes(ctx, p.userID, b)
	}, func(r reposync.AssetEditDelete) string { return r.ID }, identity, emit); err != nil {
		return err
	}

	return page(modelsync.AssetEditV1, p, func(b reposync.Bounds) ([]reposync.AssetEdit, error) {
		return repo.AssetEdits(ctx, p.userID, b)
	}, func(r reposync.AssetEdit) string { return r.UpdateID }, identity, emit)
}
