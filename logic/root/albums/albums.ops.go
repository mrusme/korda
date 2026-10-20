package albums

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	repoalbum "tty.fail/mrus/korda/services/repositories/album"
	"xn--gckvb8fzb.com/glides/runtime"
)

var (
	ErrUserIDMe       = errors.New("album user id me is not a UUID")
	ErrNoAlbumMembers = errors.New("album has no members")
)

type Statistics struct {
	Owned     int
	Shared    int
	NotShared int
}

func List(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *ListInput) ([]Details, error) {
	repos := gh.Repositories(rt)
	if err := repos.Album.UpdateThumbnails(ctx, now()); err != nil {
		return nil, err
	}

	var (
		list        []album.Album
		err         error
		sharedLinks = true
	)
	if in.AssetID != nil {
		sharedLinks = false
		list, err = repos.Album.ByAsset(ctx, a.User.ID, uuid.MustParse(*in.AssetID))
	} else {
		f := repoalbum.Filter{IsOwned: in.IsOwned, IsShared: in.IsShared}
		if in.ID != nil {
			f.ID = uuid.NullUUID{UUID: uuid.MustParse(*in.ID), Valid: true}
		}
		if in.Name != nil {
			f.Name = *in.Name
		}
		list, err = repos.Album.List(ctx, a.User.ID, f)
	}
	if err != nil || len(list) == 0 {
		return []Details{}, err
	}

	out, err := withMembers(ctx, rt, list, a.User.ID, sharedLinks)
	if err != nil {
		return nil, err
	}

	return out, addMetadata(ctx, rt, out)
}

func Stats(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*Statistics, error) {
	repos := gh.Repositories(rt)
	yes, no := true, false
	owned, err := repos.Album.List(ctx, a.User.ID, repoalbum.Filter{IsOwned: &yes})
	if err != nil {
		return nil, err
	}
	shared, err := repos.Album.List(ctx, a.User.ID, repoalbum.Filter{IsShared: &yes})
	if err != nil {
		return nil, err
	}
	notShared, err := repos.Album.List(ctx, a.User.ID, repoalbum.Filter{IsOwned: &yes, IsShared: &no})
	if err != nil {
		return nil, err
	}

	return &Statistics{Owned: len(owned), Shared: len(shared), NotShared: len(notShared)}, nil
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.AlbumRead, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	if err := repos.Album.UpdateThumbnails(ctx, now()); err != nil {
		return nil, err
	}
	found, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	list, err := withMembers(ctx, rt, []album.Album{*found}, a.User.ID, true)
	if err != nil {
		return nil, err
	}
	if err = addMetadata(ctx, rt, list); err != nil {
		return nil, err
	}

	d := &list[0]
	if len(d.Members) > 1 || d.HasSharedLink {
		counts, err := repos.Album.Contributors(ctx, id)
		if err != nil {
			return nil, err
		}
		d.Contributors = &counts
	}

	return d, nil
}

func MapMarkers(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) ([]repoalbum.Marker, error) {
	if err := access.Require(ctx, rt, a, permission.AlbumRead, id); err != nil {
		return nil, err
	}
	if a.SharedLink != nil && !a.SharedLink.ShowExif {
		return []repoalbum.Marker{}, nil
	}

	return gh.Repositories(rt).Album.MapMarkers(ctx, id)
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*Details, error) {
	repos := gh.Repositories(rt)
	members := []repoalbum.NewMember{{UserID: a.User.ID, Role: album.RoleOwner}}
	for _, m := range in.AlbumUsers {
		userID := uuid.MustParse(m.UserID)
		if userID == a.User.ID {
			continue
		}
		if _, err := repos.User.Get(ctx, userID, false); errors.Is(err, errs.ErrNoRows) {
			return nil, errs.ErrInvalidUser
		} else if err != nil {
			return nil, err
		}
		members = append(members, repoalbum.NewMember{UserID: userID, Role: m.Role})
	}

	assetIDs, err := access.Allowed(ctx, rt, a, permission.AssetShare, parseIDs(in.AssetIDs)...)
	if err != nil {
		return nil, err
	}

	prefs, err := logicusers.Preferences(ctx, rt, a.User.ID)
	if err != nil {
		return nil, err
	}

	c := repoalbum.Create{
		Name:     in.AlbumName,
		Order:    album.Order(prefs.Albums.DefaultAssetOrder),
		Members:  members,
		AssetIDs: assetIDs,
	}
	if in.Description.Set && !in.Description.Null && in.Description.V != "" {
		c.Description = sql.Null[string]{V: in.Description.V, Valid: true}
	}
	if len(assetIDs) > 0 {
		c.Thumbnail = uuid.NullUUID{UUID: assetIDs[0], Valid: true}
	}

	created, err := repos.Album.Create(ctx, c, now())
	if err != nil {
		return nil, err
	}

	return load(ctx, rt, created, a.User.ID, false)
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.AlbumUpdate, id); err != nil {
		return nil, err
	}

	found, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}
	before := Details{Album: found}
	if err = addAggregated(ctx, rt, &before); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	u := repoalbum.Update{Name: in.AlbumName, IsActivityEnabled: in.IsActivityEnabled, Order: in.Order}
	if in.Description.Set {
		u.Description = in.Description
		if !in.Description.Null && in.Description.V == "" {
			u.Description.Null = true
		}
	}
	if in.AlbumThumbnailAssetID != nil {
		thumbnail := uuid.MustParse(*in.AlbumThumbnailAssetID)
		present, err := repos.Album.AssetIDs(ctx, id, []uuid.UUID{thumbnail})
		if err != nil {
			return nil, err
		}
		if len(present) == 0 {
			return nil, errs.ErrInvalidAlbumThumbnail
		}
		u.Thumbnail = uuid.NullUUID{UUID: thumbnail, Valid: true}
	}

	updated, err := repos.Album.Update(ctx, id, u, now())
	if err != nil {
		return nil, err
	}

	list, err := withMembers(ctx, rt, []album.Album{*updated}, a.User.ID, true)
	if err != nil {
		return nil, err
	}
	d := &list[0]
	d.StartDate, d.EndDate, d.AssetCount = before.StartDate, before.EndDate, before.AssetCount

	return d, nil
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.AlbumDelete, id); err != nil {
		return err
	}

	return gh.Repositories(rt).Album.Delete(ctx, id, now())
}

func AddAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *BulkIDsInput) ([]bulk.Result, error) {
	found, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}
	if err = access.Require(ctx, rt, a, permission.AlbumAssetCreate, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	ids := parseIDs(in.IDs)
	present, err := repos.Album.AssetIDs(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	allowed, err := access.Allowed(ctx, rt, a, permission.AssetShare, bulk.Missing(ids, present)...)
	if err != nil {
		return nil, err
	}

	results, added := bulk.Add(ids, present, allowed)
	if len(added) == 0 {
		return results, nil
	}

	at := now()
	pairs := make([]repoalbum.Pair, 0, len(added))
	for _, assetID := range added {
		pairs = append(pairs, repoalbum.Pair{AlbumID: id, AssetID: assetID})
	}
	if err = repos.Album.AddAssets(ctx, pairs, at); err != nil {
		return nil, err
	}

	thumbnail := found.AlbumThumbnailAssetID
	if !thumbnail.Valid {
		thumbnail = uuid.NullUUID{UUID: added[0], Valid: true}
	}

	return results, repos.Album.Touch(ctx, id, thumbnail, at)
}

func AddToAlbums(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *AddToAlbumsInput) (bool, bulk.Reason, error) {
	albumIDs, err := access.Allowed(ctx, rt, a, permission.AlbumAssetCreate, parseIDs(in.AlbumIDs)...)
	if err != nil {
		return false, "", err
	}
	if len(albumIDs) == 0 {
		return false, bulk.NoPermission, nil
	}
	assetIDs, err := access.Allowed(ctx, rt, a, permission.AssetShare, parseIDs(in.AssetIDs)...)
	if err != nil {
		return false, "", err
	}
	if len(assetIDs) == 0 {
		return false, bulk.NoPermission, nil
	}

	repos := gh.Repositories(rt)
	at := now()
	success := false
	var pairs []repoalbum.Pair
	for _, albumID := range albumIDs {
		present, err := repos.Album.AssetIDs(ctx, albumID, assetIDs)
		if err != nil {
			return false, "", err
		}
		missing := bulk.Missing(assetIDs, present)
		if len(missing) == 0 {
			continue
		}
		found, err := find(ctx, rt, albumID)
		if err != nil {
			return false, "", err
		}
		success = true
		for _, assetID := range missing {
			pairs = append(pairs, repoalbum.Pair{AlbumID: albumID, AssetID: assetID})
		}
		thumbnail := found.AlbumThumbnailAssetID
		if !thumbnail.Valid {
			thumbnail = uuid.NullUUID{UUID: missing[0], Valid: true}
		}
		if err = repos.Album.Touch(ctx, albumID, thumbnail, at); err != nil {
			return false, "", err
		}
	}

	if err = repos.Album.AddAssets(ctx, pairs, at); err != nil {
		return false, "", err
	}
	if !success {
		return false, bulk.Duplicate, nil
	}

	return true, "", nil
}

func RemoveAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *BulkIDsInput) ([]bulk.Result, error) {
	if err := access.Require(ctx, rt, a, permission.AlbumAssetDelete, id); err != nil {
		return nil, err
	}

	found, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	canAlways, err := access.Allowed(ctx, rt, a, permission.AlbumDelete, id)
	if err != nil {
		return nil, err
	}
	ids := parseIDs(in.IDs)
	present, err := repos.Album.AssetIDs(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	allowed := present
	if len(canAlways) == 0 {
		if allowed, err = access.Allowed(ctx, rt, a, permission.AssetShare, present...); err != nil {
			return nil, err
		}
	}

	results, removed := bulk.Remove(ids, present, allowed)
	if len(removed) == 0 {
		return results, nil
	}

	at := now()
	if err = repos.Album.RemoveAssets(ctx, id, removed, at); err != nil {
		return nil, err
	}
	if found.AlbumThumbnailAssetID.Valid && slices.Contains(removed, found.AlbumThumbnailAssetID.UUID) {
		if err = repos.Album.UpdateThumbnails(ctx, at); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func AddUsers(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *AddUsersInput) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.AlbumShare, id); err != nil {
		return nil, err
	}

	if _, err := find(ctx, rt, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	members, err := repos.Album.Members(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	for _, m := range in.AlbumUsers {
		role := album.RoleEditor
		if m.Role != nil {
			role = *m.Role
		}
		if role == album.RoleOwner {
			return nil, errs.ErrAnotherOwner
		}

		userID := uuid.MustParse(m.UserID)
		if slices.ContainsFunc(members[id], func(x repoalbum.Member) bool { return x.ID == userID }) {
			continue
		}
		if _, err = repos.User.Get(ctx, userID, false); errors.Is(err, errs.ErrNoRows) {
			return nil, errs.ErrInvalidUser
		} else if err != nil {
			return nil, err
		}
		if err = repos.Album.AddUser(ctx, id, repoalbum.NewMember{UserID: userID, Role: role}, now()); err != nil {
			return nil, err
		}
	}

	found, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	return load(ctx, rt, found, a.User.ID, true)
}

func UpdateUser(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UserParams, role album.Role) error {
	id := uuid.MustParse(in.ID)
	if err := access.Require(ctx, rt, a, permission.AlbumShare, id); err != nil {
		return err
	}
	if in.UserID == "me" {
		return ErrUserIDMe
	}

	if _, err := find(ctx, rt, id); err != nil {
		return err
	}

	userID := uuid.MustParse(in.UserID)
	members, err := gh.Repositories(rt).Album.Members(ctx, []uuid.UUID{id})
	if err != nil {
		return err
	}
	list := members[id]
	if len(list) == 0 {
		return ErrNoAlbumMembers
	}
	sortMembers(list, userID)
	if list[0].ID == userID {
		return errs.ErrUserIsOwner
	}

	return gh.Repositories(rt).Album.UpdateRole(ctx, id, userID, role, now())
}

func RemoveUser(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UserParams) error {
	id := uuid.MustParse(in.ID)
	userID := a.User.ID
	if in.UserID != "me" {
		userID = uuid.MustParse(in.UserID)
	}

	if _, err := find(ctx, rt, id); err != nil {
		return err
	}

	members, err := gh.Repositories(rt).Album.Members(ctx, []uuid.UUID{id})
	if err != nil {
		return err
	}
	idx := slices.IndexFunc(members[id], func(m repoalbum.Member) bool { return m.ID == userID })
	if idx < 0 {
		return errs.ErrAlbumNotShared
	}
	owners := 0
	for _, m := range members[id] {
		if m.Role == album.RoleOwner {
			owners++
		}
	}
	if members[id][idx].Role == album.RoleOwner && owners == 1 {
		return errs.ErrLastAlbumOwner
	}
	if userID != a.User.ID {
		if err = access.Require(ctx, rt, a, permission.AlbumShare, id); err != nil {
			return err
		}
	}

	return gh.Repositories(rt).Album.RemoveUser(ctx, id, userID, now())
}
