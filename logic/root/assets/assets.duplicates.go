package assets

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/tag"
	repoalbum "tty.fail/mrus/korda/services/repositories/album"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	repoexif "tty.fail/mrus/korda/services/repositories/assetexif"
	"xn--gckvb8fzb.com/glides/runtime"
)

type DuplicateGroup struct {
	DuplicateID uuid.UUID
	Assets      []asset.Asset
	Exif        map[uuid.UUID]*assetexif.Exif
	Tags        map[uuid.UUID][]tag.Tag
}

type ResolveGroupInput struct {
	DuplicateID   string   `json:"duplicateId" v:"uuid"`
	KeepAssetIDs  []string `json:"keepAssetIds" vi:"uuid"`
	TrashAssetIDs []string `json:"trashAssetIds" vi:"uuid"`
}

type ResolveInput struct {
	Groups []ResolveGroupInput `json:"groups" v:"min=1"`
}

type ResolveResult struct {
	ID           uuid.UUID
	Success      bool
	Error        bulk.Reason
	ErrorMessage string
}

const reasonValidation bulk.Reason = "validation"

func Duplicates(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]DuplicateGroup, error) {
	repos := gh.Repositories(rt)
	if err := repos.Asset.ClearSingletonDuplicates(ctx, a.User.ID, now()); err != nil {
		return nil, err
	}

	list, err := repos.Asset.Duplicates(ctx, a.User.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(list))
	for _, x := range list {
		ids = append(ids, x.ID)
	}
	exif, err := repos.AssetExif.ByAssets(ctx, ids)
	if err != nil {
		return nil, err
	}
	tags, err := repos.Tag.ForAssets(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := []DuplicateGroup{}
	for _, x := range list {
		if n := len(out); n == 0 || out[n-1].DuplicateID != x.DuplicateID.UUID {
			out = append(out, DuplicateGroup{DuplicateID: x.DuplicateID.UUID, Exif: exif, Tags: tags})
		}
		out[len(out)-1].Assets = append(out[len(out)-1].Assets, x)
	}

	return slices.DeleteFunc(out, func(g DuplicateGroup) bool { return len(g.Assets) < 2 }), nil
}

func DeleteDuplicates(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, ids []uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.DuplicateDelete, ids...); err != nil {
		return err
	}

	return gh.Repositories(rt).Asset.ClearDuplicates(ctx, a.User.ID, ids, now())
}

func ResolveDuplicates(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *ResolveInput) ([]ResolveResult, error) {
	ids := make([]uuid.UUID, 0, len(in.Groups))
	for _, g := range in.Groups {
		ids = append(ids, uuid.MustParse(g.DuplicateID))
	}
	if err := access.Require(ctx, rt, a, permission.DuplicateDelete, ids...); err != nil {
		return nil, err
	}

	out := make([]ResolveResult, 0, len(in.Groups))
	for i, g := range in.Groups {
		r, err := resolve(ctx, rt, a, ids[i], g)
		if err != nil {
			rt.Error("duplicates", "resolve", "group", ids[i].String(), "error", err)
			r = ResolveResult{ID: ids[i], Error: bulk.Unknown}
		}
		out = append(out, r)
	}

	return out, nil
}

func uniqueLines(values []string) []string {
	var out []string
	for _, v := range values {
		for line := range strings.SplitSeq(v, "\n") {
			if line = strings.TrimSpace(line); line != "" && !slices.Contains(out, line) {
				out = append(out, line)
			}
		}
	}

	return out
}

func resolve(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, g ResolveGroupInput) (ResolveResult, error) {
	repos := gh.Repositories(rt)
	members, err := repos.Asset.DuplicateGroup(ctx, id)
	if err != nil {
		return ResolveResult{}, err
	}
	if len(members) == 0 {
		return ResolveResult{ID: id, Error: bulk.NotFound}, nil
	}

	inGroup := map[uuid.UUID]bool{}
	for _, m := range members {
		inGroup[m.ID] = true
	}
	pick := func(list []string) []uuid.UUID {
		var out []uuid.UUID
		for _, s := range list {
			if u := uuid.MustParse(s); inGroup[u] {
				out = append(out, u)
			}
		}
		return out
	}
	keep, trash := pick(g.KeepAssetIDs), pick(g.TrashAssetIDs)

	for _, m := range members {
		kept, trashed := slices.Contains(keep, m.ID), slices.Contains(trash, m.ID)
		if kept && trashed {
			return ResolveResult{ID: id, Error: reasonValidation, ErrorMessage: "An asset cannot be in both keepAssetIds and trashAssetIds"}, nil
		}
		if !kept && !trashed {
			return ResolveResult{ID: id, Error: reasonValidation, ErrorMessage: "Every asset must be in either keepAssetIds or trashAssetIds"}, nil
		}
	}

	if len(trash) > 0 {
		allowed, err := access.Allowed(ctx, rt, a, permission.AssetDelete, trash...)
		if err != nil {
			return ResolveResult{}, err
		}
		if len(allowed) != len(trash) {
			return ResolveResult{ID: id, Error: bulk.NoPermission, ErrorMessage: "No permission to delete assets"}, nil
		}
	}

	at := now()
	update := repoasset.Update{DuplicateID: optional.Value[uuid.UUID]{Set: true, Null: true}}
	if len(keep) == 1 && len(trash) > 0 {
		if err = merge(ctx, rt, a, members, keep, &update); err != nil {
			return ResolveResult{}, err
		}
	}
	if len(keep) > 0 {
		if err = repos.Asset.UpdateAll(ctx, keep, update, at); err != nil {
			return ResolveResult{}, err
		}
	}

	if len(trash) > 0 {
		cfg, err := gh.ServerConfigFrom(rt.Config())
		if err != nil {
			return ResolveResult{}, err
		}
		status := asset.StatusTrashed
		if cfg.TrashDays == 0 {
			status = asset.StatusDeleted
		}
		if err = repos.Asset.UpdateAll(ctx, trash, repoasset.Update{
			DeletedAt: optional.Of(at), Status: optional.Of(status), DuplicateID: optional.Value[uuid.UUID]{Set: true, Null: true},
		}, at); err != nil {
			return ResolveResult{}, err
		}
		if status == asset.StatusDeleted {
			if err = gh.Jobs(rt).EmptyTrash(); err != nil {
				return ResolveResult{}, err
			}
		} else {
			gh.Emit(ctx, rt, event.AssetsTrashed{UserID: a.User.ID, AssetIDs: trash})
		}
	}

	return ResolveResult{ID: id, Success: true}, nil
}

func merge(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, members []asset.Asset, keep []uuid.UUID, update *repoasset.Update) error {
	repos := gh.Repositories(rt)
	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.ID)
	}
	exif, err := repos.AssetExif.ByAssets(ctx, ids)
	if err != nil {
		return err
	}

	favorite := slices.ContainsFunc(members, func(m asset.Asset) bool { return m.IsFavorite })
	update.IsFavorite = optional.Of(favorite)
	for _, v := range []asset.Visibility{asset.VisibilityLocked, asset.VisibilityArchive, asset.VisibilityTimeline, asset.VisibilityHidden} {
		if slices.ContainsFunc(members, func(m asset.Asset) bool { return m.Visibility == v }) {
			update.Visibility = optional.Of(v)
			break
		}
	}

	var writes []repoexif.Write
	var rating int64
	var descriptions []string
	latitudes, longitudes := map[float64]bool{}, map[float64]bool{}
	for _, id := range ids {
		e := exif[id]
		if e == nil {
			continue
		}
		if e.Rating.Valid && e.Rating.V > rating {
			rating = e.Rating.V
		}
		descriptions = append(descriptions, e.Description)
		if e.Latitude.Valid {
			latitudes[e.Latitude.V] = true
		}
		if e.Longitude.Valid {
			longitudes[e.Longitude.V] = true
		}
	}
	if rating > 0 {
		writes = append(writes, repoexif.Write{Column: "rating", Value: rating})
	}
	if lines := uniqueLines(descriptions); len(lines) > 0 {
		writes = append(writes, repoexif.Write{Column: "description", Value: strings.Join(lines, "\n")})
	}
	if len(latitudes) == 1 && len(longitudes) == 1 {
		for lat := range latitudes {
			writes = append(writes, repoexif.Write{Column: "latitude", Value: lat})
		}
		for lon := range longitudes {
			writes = append(writes, repoexif.Write{Column: "longitude", Value: lon})
		}
	}

	var albumIDs []uuid.UUID
	for _, id := range ids {
		albums, err := repos.Album.ByAsset(ctx, a.User.ID, id)
		if err != nil {
			return err
		}
		for _, al := range albums {
			if !slices.Contains(albumIDs, al.ID) {
				albumIDs = append(albumIDs, al.ID)
			}
		}
	}
	if len(albumIDs) > 0 {
		allowedAlbums, err := access.Allowed(ctx, rt, a, permission.AlbumAssetCreate, albumIDs...)
		if err != nil {
			return err
		}
		allowedKeep, err := access.Allowed(ctx, rt, a, permission.AssetShare, keep...)
		if err != nil {
			return err
		}
		var pairs []repoalbum.Pair
		for _, albumID := range allowedAlbums {
			for _, assetID := range allowedKeep {
				pairs = append(pairs, repoalbum.Pair{AlbumID: albumID, AssetID: assetID})
			}
		}
		if len(pairs) > 0 {
			if err = repos.Album.AddAssets(ctx, pairs, now()); err != nil {
				return err
			}
		}
	}

	tags, err := repos.Tag.ForAssets(ctx, ids)
	if err != nil {
		return err
	}
	var tagIDs []uuid.UUID
	for _, id := range ids {
		for _, t := range tags[id] {
			if !slices.Contains(tagIDs, t.ID) {
				tagIDs = append(tagIDs, t.ID)
			}
		}
	}
	if len(tagIDs) > 0 {
		allowed, err := access.Allowed(ctx, rt, a, permission.TagAsset, tagIDs...)
		if err != nil {
			return err
		}
		if len(allowed) > 0 {
			for _, id := range keep {
				if err = repos.Tag.ReplaceAssetTags(ctx, nil, id, allowed); err != nil {
					return err
				}
			}
		}
	}

	if len(writes) > 0 {
		if err = repos.AssetExif.UpdateAll(ctx, keep, writes, repoexif.Names(writes), now()); err != nil {
			return err
		}
	}
	if len(writes) > 0 || len(tagIDs) > 0 {
		sidecarWrite(rt, keep...)
	}

	return nil
}
