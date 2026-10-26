package access

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/album"
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
		return Denied(p)
	}

	return nil
}

func Denied(p permission.Permission) error {
	return fmt.Errorf("%w %s access", errs.ErrNoAccess, p)
}

type checker func(ids []uuid.UUID) ([]uuid.UUID, error)

func union(ids []uuid.UUID, checks ...checker) ([]uuid.UUID, error) {
	allowed := map[uuid.UUID]bool{}
	out := []uuid.UUID{}
	remaining := ids
	for _, check := range checks {
		if len(remaining) == 0 {
			break
		}
		found, err := check(remaining)
		if err != nil {
			return nil, err
		}
		for _, id := range found {
			if !allowed[id] {
				allowed[id] = true
				out = append(out, id)
			}
		}
		remaining = slices.DeleteFunc(slices.Clone(remaining), func(id uuid.UUID) bool { return allowed[id] })
	}

	return out, nil
}

func Allowed(
	ctx context.Context,
	rt *runtime.Runtime,
	a *auth.Auth,
	p permission.Permission,
	ids ...uuid.UUID,
) ([]uuid.UUID, error) {
	found, err := check(ctx, rt, a, p, ids)
	if found == nil && err == nil {
		found = []uuid.UUID{}
	}

	return found, err
}

func check(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, p permission.Permission, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	repos := gh.Repositories(rt)
	if a.SharedLink != nil {
		return checkSharedLink(ctx, rt, a, p, ids)
	}

	userID := a.User.ID
	assetOwner := func(elevated bool) checker {
		return func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Asset.OwnedBy(ctx, userID, ids, elevated) }
	}
	assetAlbum := func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Asset.AlbumAccess(ctx, userID, ids) }
	assetPartner := func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Asset.PartnerAccess(ctx, userID, ids) }
	albumOwner := func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Album.OwnedBy(ctx, userID, ids) }
	albumShared := func(least album.Role) checker {
		return func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Album.SharedWith(ctx, userID, ids, least) }
	}

	switch p {
	case permission.AuthDeviceDelete, permission.SessionUpdate, permission.SessionLock:
		return repos.Session.OwnedBy(ctx, userID, ids)

	case permission.AssetRead, permission.AssetView, permission.AssetDownload:
		return union(ids, assetOwner(a.Elevated), assetAlbum, assetPartner)

	case permission.AssetShare:
		return union(ids, assetOwner(false), assetPartner)

	case permission.AssetUpdate, permission.AssetDelete, permission.AssetCopy,
		permission.AssetEditGet, permission.AssetEditCreate, permission.AssetEditDelete:
		return repos.Asset.OwnedBy(ctx, userID, ids, a.Elevated)

	case permission.AlbumRead, permission.AlbumDownload:
		return union(ids, albumOwner, albumShared(album.RoleViewer))

	case permission.AlbumAssetCreate, permission.AlbumUpdate, permission.AlbumShare, permission.AlbumAssetDelete:
		return union(ids, albumOwner, albumShared(album.RoleEditor))

	case permission.AlbumDelete:
		return repos.Album.OwnedBy(ctx, userID, ids)

	case permission.ActivityCreate:
		return repos.Activity.CanCreate(ctx, userID, ids)

	case permission.ActivityDelete:
		return union(ids,
			func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Activity.OwnedBy(ctx, userID, ids) },
			func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Activity.OfOwnedAlbums(ctx, userID, ids) })

	case permission.MemoryRead, permission.MemoryUpdate, permission.MemoryDelete:
		return repos.Memory.OwnedBy(ctx, userID, ids)

	case permission.NotificationRead, permission.NotificationUpdate, permission.NotificationDelete:
		return repos.Notification.OwnedBy(ctx, userID, ids)

	case permission.PartnerUpdate:
		return repos.Partner.SharingWith(ctx, userID, ids)

	case permission.StackRead, permission.StackUpdate, permission.StackDelete:
		return repos.Stack.OwnedBy(ctx, userID, ids)

	case permission.TagAsset, permission.TagRead, permission.TagUpdate, permission.TagDelete:
		return repos.Tag.OwnedBy(ctx, userID, ids)

	case permission.DuplicateRead, permission.DuplicateDelete:
		return repos.Asset.DuplicateAccess(ctx, userID, ids)

	case permission.AssetFileRead, permission.AssetFileDelete, permission.AssetFileDownload:
		return repos.AssetFile.OwnedBy(ctx, userID, ids, a.Elevated)

	case permission.TimelineRead:
		return union(ids, self(userID),
			func(ids []uuid.UUID) ([]uuid.UUID, error) { return repos.Partner.SharingWith(ctx, userID, ids) })

	case permission.AssetUpload, permission.ArchiveRead, permission.TimelineDownload:
		return self(userID)(ids)
	}

	return nil, nil
}

func self(userID uuid.UUID) checker {
	return func(ids []uuid.UUID) ([]uuid.UUID, error) {
		if slices.Contains(ids, userID) {
			return []uuid.UUID{userID}, nil
		}
		return nil, nil
	}
}

func checkSharedLink(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, p permission.Permission, ids []uuid.UUID) ([]uuid.UUID, error) {
	repos := gh.Repositories(rt)
	link := a.SharedLink

	switch p {
	case permission.AssetRead, permission.AssetView:
		return repos.SharedLink.AssetAccess(ctx, link.ID, ids)
	case permission.AssetDownload:
		if link.AllowDownload {
			return repos.SharedLink.AssetAccess(ctx, link.ID, ids)
		}
	case permission.AssetUpload:
		if link.AllowUpload {
			return ids, nil
		}
	case permission.AlbumRead:
		return repos.SharedLink.AlbumAccess(ctx, link.ID, ids)
	case permission.AlbumDownload:
		if link.AllowDownload {
			return repos.SharedLink.AlbumAccess(ctx, link.ID, ids)
		}
	case permission.AlbumAssetCreate:
		if link.AllowUpload {
			return repos.SharedLink.AlbumAccess(ctx, link.ID, ids)
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
