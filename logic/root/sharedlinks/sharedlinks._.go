package sharedlinks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/sharedlink"
	"tty.fail/mrus/korda/models/user"
	reposharedlink "tty.fail/mrus/korda/services/repositories/sharedlink"
	"xn--gckvb8fzb.com/glides/runtime"
)

const keyLength = 50

type SearchInput struct {
	AlbumID *string `json:"albumId" v:"uuid"`
	ID      *string `json:"id" v:"uuid"`
}

type CreateInput struct {
	Type          sharedlink.Type `json:"type"`
	AssetIDs      []string        `json:"assetIds" v:"optional" vi:"uuid"`
	AlbumID       *string         `json:"albumId" v:"uuid"`
	Description   *string         `json:"description" v:"nullable"`
	Password      *string         `json:"password" v:"nullable"`
	Slug          *string         `json:"slug" v:"nullable"`
	ExpiresAt     *time.Time      `json:"expiresAt" v:"datetime,nullable"`
	AllowUpload   *bool           `json:"allowUpload"`
	AllowDownload *bool           `json:"allowDownload"`
	ShowMetadata  *bool           `json:"showMetadata"`
}

func (CreateInput) Refine(in refine.Fields) []refine.Issue {
	raw, _ := in.Get("type")
	kind, _ := raw.(string)
	raw, _ = in.Get("albumId")
	albumID, _ := raw.(string)
	raw, _ = in.Get("assetIds")
	assetIDs, _ := raw.([]any)

	var out []refine.Issue
	switch sharedlink.Type(kind) {
	case sharedlink.TypeAlbum:
		if albumID == "" {
			out = append(out, refine.Added("albumId is required for type ALBUM"))
		}
		if len(assetIDs) > 0 {
			out = append(out, refine.Added("assetIds can only be used with type INDIVIDUAL"))
		}
	case sharedlink.TypeIndividual:
		if len(assetIDs) == 0 {
			out = append(out, refine.Added("assetIds are required for type INDIVIDUAL"))
		}
		if albumID != "" {
			out = append(out, refine.Added("albumId can only be used with type ALBUM"))
		}
	}

	return out
}

type UpdateInput struct {
	Description   optional.Value[string]    `json:"description" v:"nullable"`
	Password      optional.Value[string]    `json:"password" v:"nullable"`
	Slug          optional.Value[string]    `json:"slug" v:"nullable"`
	ExpiresAt     optional.Value[time.Time] `json:"expiresAt" v:"datetime,nullable"`
	AllowUpload   *bool                     `json:"allowUpload"`
	AllowDownload *bool                     `json:"allowDownload"`
	ShowMetadata  *bool                     `json:"showMetadata"`
}

type LoginInput struct {
	Password string `json:"password"`
}

type AssetIDsInput struct {
	AssetIDs []string `json:"assetIds" vi:"uuid"`
}

type Result struct {
	AssetID string
	Success bool
	Error   bulk.Reason
}

type Album struct {
	Album  *album.Album
	Owners []user.User
	Assets []asset.Asset
}

type Details struct {
	Link   *sharedlink.SharedLink
	Assets []asset.Asset
	Exif   map[uuid.UUID]*assetexif.Exif
	Album  *Album
	Strip  bool
}

func now() time.Time {
	return time.Now().UTC()
}

func nullString(v *string) sql.Null[string] {
	if v == nil || *v == "" {
		return sql.Null[string]{}
	}

	return sql.Null[string]{V: *v, Valid: true}
}

func saveError(err error) error {
	if errors.Is(err, errs.ErrUniqueViolationOn) && strings.Contains(err.Error(), "shared_links.slug") {
		return errs.ErrSharedLinkSave
	}

	return err
}

func findOrFail(ctx context.Context, rt *runtime.Runtime, userID uuid.UUID, id uuid.UUID) (*sharedlink.SharedLink, error) {
	link, err := gh.Repositories(rt).SharedLink.Get(ctx, userID, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrSharedLinkNotFound
	}

	return link, err
}

func loadAlbum(ctx context.Context, rt *runtime.Runtime, id uuid.NullUUID, withAssets bool) (*Album, error) {
	if !id.Valid {
		return nil, nil
	}

	repos := gh.Repositories(rt)
	found, err := repos.Album.Get(ctx, id.UUID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	members, err := repos.Album.Members(ctx, []uuid.UUID{id.UUID})
	if err != nil {
		return nil, err
	}
	out := &Album{Album: found, Owners: []user.User{}}
	for _, m := range members[id.UUID] {
		if m.Role == album.RoleOwner {
			out.Owners = append(out.Owners, m.User)
		}
	}

	if withAssets {
		if out.Assets, err = repos.SharedLink.AlbumAssets(ctx, id.UUID); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func withExif(ctx context.Context, rt *runtime.Runtime, list []asset.Asset) (map[uuid.UUID]*assetexif.Exif, error) {
	ids := make([]uuid.UUID, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}

	return gh.Repositories(rt).AssetExif.ByAssets(ctx, ids)
}

func full(ctx context.Context, rt *runtime.Runtime, link *sharedlink.SharedLink, strip bool) (*Details, error) {
	assets, err := gh.Repositories(rt).SharedLink.Assets(ctx, link.ID)
	if err != nil {
		return nil, err
	}
	exif, err := withExif(ctx, rt, assets)
	if err != nil {
		return nil, err
	}
	linked, err := loadAlbum(ctx, rt, link.AlbumID, true)
	if err != nil {
		return nil, err
	}

	return &Details{Link: link, Assets: assets, Exif: exif, Album: linked, Strip: strip}, nil
}

func written(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*Details, error) {
	repos := gh.Repositories(rt)
	link, err := repos.SharedLink.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	assets, err := repos.SharedLink.Linked(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Details{Link: link, Assets: assets}, nil
}

func All(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]Details, error) {
	f := reposharedlink.Filter{}
	if in.AlbumID != nil {
		f.AlbumID = uuid.NullUUID{UUID: uuid.MustParse(*in.AlbumID), Valid: true}
	}
	if in.ID != nil {
		f.ID = uuid.NullUUID{UUID: uuid.MustParse(*in.ID), Valid: true}
	}

	repos := gh.Repositories(rt)
	links, err := repos.SharedLink.All(ctx, a.User.ID, f)
	if err != nil {
		return nil, err
	}

	out := make([]Details, 0, len(links))
	for i := range links {
		cover, err := repos.SharedLink.Cover(ctx, links[i].ID)
		if err != nil {
			return nil, err
		}
		linked, err := loadAlbum(ctx, rt, links[i].AlbumID, false)
		if err != nil {
			return nil, err
		}
		out = append(out, Details{Link: &links[i], Assets: cover, Album: linked})
	}

	return out, nil
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Details, error) {
	link, err := findOrFail(ctx, rt, a.User.ID, id)
	if err != nil {
		return nil, err
	}

	return full(ctx, rt, link, false)
}

func Mine(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*Details, error) {
	if a.SharedLink == nil {
		return nil, errs.ErrForbidden
	}

	link, err := findOrFail(ctx, rt, a.User.ID, a.SharedLink.ID)
	if err != nil {
		return nil, err
	}

	return full(ctx, rt, link, !link.ShowExif)
}

func Login(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *LoginInput) (*Details, string, error) {
	if a.SharedLink == nil {
		return nil, "", errs.ErrForbidden
	}

	link, err := findOrFail(ctx, rt, a.User.ID, a.SharedLink.ID)
	if err != nil {
		return nil, "", err
	}
	if !link.HasPassword() {
		return nil, "", errs.ErrSharedLinkNoPassword
	}
	if link.Password.V != in.Password {
		return nil, "", errs.ErrSharedLinkPassword
	}

	d, err := full(ctx, rt, link, !link.ShowExif)
	if err != nil {
		return nil, "", err
	}

	return d, link.Token(), nil
}

func parseIDs(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		out = append(out, uuid.MustParse(id))
	}

	return out
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*Details, error) {
	link := &sharedlink.SharedLink{
		ID: uuid.New(), UserID: a.User.ID, Type: in.Type, Description: nullString(in.Description),
		CreatedAt: now(), AllowUpload: in.AllowUpload == nil || *in.AllowUpload,
		AllowDownload: in.AllowDownload == nil || *in.AllowDownload,
		ShowExif:      in.ShowMetadata == nil || *in.ShowMetadata, Slug: nullString(in.Slug),
	}
	if !link.ShowExif {
		link.AllowDownload = false
	}
	if in.Password != nil {
		link.Password = sql.Null[string]{V: *in.Password, Valid: true}
	}
	if in.ExpiresAt != nil {
		link.ExpiresAt = sql.Null[time.Time]{V: in.ExpiresAt.UTC(), Valid: true}
	}

	var assetIDs []uuid.UUID
	switch in.Type {
	case sharedlink.TypeAlbum:
		albumID := uuid.MustParse(*in.AlbumID)
		if err := access.Require(ctx, rt, a, permission.AlbumShare, albumID); err != nil {
			return nil, err
		}
		link.AlbumID = uuid.NullUUID{UUID: albumID, Valid: true}
	case sharedlink.TypeIndividual:
		assetIDs = parseIDs(in.AssetIDs)
		if err := access.Require(ctx, rt, a, permission.AssetShare, assetIDs...); err != nil {
			return nil, err
		}
	}

	link.Key = make([]byte, keyLength)
	if _, err := rand.Read(link.Key); err != nil {
		return nil, err
	}

	if err := gh.Repositories(rt).SharedLink.Create(ctx, link, assetIDs); err != nil {
		return nil, saveError(err)
	}

	return written(ctx, rt, link.ID)
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*Details, error) {
	if _, err := findOrFail(ctx, rt, a.User.ID, id); err != nil {
		return nil, err
	}

	u := reposharedlink.Update{
		Description: in.Description, Password: in.Password, ExpiresAt: in.ExpiresAt, AllowUpload: in.AllowUpload,
		AllowDownload: in.AllowDownload, ShowExif: in.ShowMetadata,
	}
	if in.Slug.Valid() && in.Slug.V != "" {
		u.Slug = sql.Null[string]{V: in.Slug.V, Valid: true}
	}
	if err := gh.Repositories(rt).SharedLink.Update(ctx, id, a.User.ID, u); err != nil {
		return nil, saveError(err)
	}

	return written(ctx, rt, id)
}

func Remove(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	link, err := findOrFail(ctx, rt, a.User.ID, id)
	if err != nil {
		return err
	}

	return gh.Repositories(rt).SharedLink.Remove(ctx, link.ID)
}

func individual(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*sharedlink.SharedLink, error) {
	link, err := findOrFail(ctx, rt, a.User.ID, id)
	if err != nil {
		return nil, err
	}
	if link.Type != sharedlink.TypeIndividual {
		return nil, errs.ErrSharedLinkType
	}

	return link, nil
}

func AddAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *AssetIDsInput) ([]Result, error) {
	link, err := individual(ctx, rt, a, id)
	if err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	current, err := repos.SharedLink.Assets(ctx, link.ID)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]bool, len(current))
	for _, c := range current {
		existing[c.ID.String()] = true
	}

	missing := []uuid.UUID{}
	for _, raw := range in.AssetIDs {
		if !existing[raw] {
			missing = append(missing, uuid.MustParse(raw))
		}
	}
	allowed, err := access.Allowed(ctx, rt, a, permission.AssetShare, missing...)
	if err != nil {
		return nil, err
	}
	permitted := make(map[string]bool, len(allowed))
	for _, id := range allowed {
		permitted[id.String()] = true
	}

	results := make([]Result, 0, len(in.AssetIDs))
	added := []uuid.UUID{}
	for _, raw := range in.AssetIDs {
		switch {
		case existing[raw]:
			results = append(results, Result{AssetID: raw, Error: bulk.Duplicate})
		case !permitted[raw]:
			results = append(results, Result{AssetID: raw, Error: bulk.NoPermission})
		default:
			results = append(results, Result{AssetID: raw, Success: true})
			added = append(added, uuid.MustParse(raw))
		}
	}

	return results, repos.SharedLink.InsertAssets(ctx, link.ID, added)
}

func RemoveAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *AssetIDsInput) ([]Result, error) {
	link, err := individual(ctx, rt, a, id)
	if err != nil {
		return nil, err
	}

	removed, err := gh.Repositories(rt).SharedLink.RemoveAssets(ctx, link.ID, parseIDs(in.AssetIDs))
	if err != nil {
		return nil, err
	}
	gone := make(map[string]bool, len(removed))
	for _, id := range removed {
		gone[id.String()] = true
	}

	results := make([]Result, 0, len(in.AssetIDs))
	for _, raw := range in.AssetIDs {
		if gone[raw] {
			results = append(results, Result{AssetID: raw, Success: true})
			continue
		}
		results = append(results, Result{AssetID: raw, Error: bulk.NotFound})
	}

	return results, nil
}
