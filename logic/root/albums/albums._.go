package albums

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/collation"
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/album"
	repoalbum "tty.fail/mrus/korda/services/repositories/album"
	"xn--gckvb8fzb.com/glides/runtime"
)

type ListInput struct {
	ID       *string `json:"id" v:"uuid"`
	Name     *string `json:"name"`
	IsOwned  *bool   `json:"isOwned"`
	IsShared *bool   `json:"isShared"`
	AssetID  *string `json:"assetId" v:"uuid"`
}

type MemberInput struct {
	UserID string     `json:"userId" v:"uuid"`
	Role   album.Role `json:"role"`
}

type CreateInput struct {
	AlbumName   string                 `json:"albumName"`
	Description optional.Value[string] `json:"description" v:"nullable"`
	AlbumUsers  []MemberInput          `json:"albumUsers" v:"optional"`
	AssetIDs    []string               `json:"assetIds" v:"optional" vi:"uuid"`
}

type UpdateInput struct {
	AlbumName             *string                `json:"albumName"`
	Description           optional.Value[string] `json:"description" v:"nullable"`
	AlbumThumbnailAssetID *string                `json:"albumThumbnailAssetId" v:"uuid"`
	IsActivityEnabled     *bool                  `json:"isActivityEnabled"`
	Order                 *album.Order           `json:"order"`
}

type AddMemberInput struct {
	UserID string      `json:"userId" v:"uuid"`
	Role   *album.Role `json:"role"`
}

type AddUsersInput struct {
	AlbumUsers []AddMemberInput `json:"albumUsers" v:"min=1"`
}

type UpdateUserInput struct {
	Role album.Role `json:"role"`
}

type UserParams struct {
	ID     string `json:"id" v:"uuid"`
	UserID string `json:"userId" v:"refine=albumUserId"`
}

type BulkIDsInput struct {
	IDs []string `json:"ids" vi:"uuid"`
}

type AddToAlbumsInput struct {
	AlbumIDs []string `json:"albumIds" vi:"uuid"`
	AssetIDs []string `json:"assetIds" vi:"uuid"`
}

type Details struct {
	Album         *album.Album
	Members       []repoalbum.Member
	HasSharedLink bool
	StartDate     *string
	EndDate       *string
	AssetCount    int64
	LastModified  *time.Time
	Contributors  *[]repoalbum.Contributor
}

func now() time.Time {
	return time.Now().UTC()
}

func parseIDs(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if parsed, err := uuid.Parse(id); err == nil {
			out = append(out, parsed)
		}
	}

	return out
}

var roleOrder = map[album.Role]int{album.RoleOwner: 0, album.RoleEditor: 1, album.RoleViewer: 2}

func sortMembers(members []repoalbum.Member, caller uuid.UUID) {
	slices.SortStableFunc(members, func(a, b repoalbum.Member) int {
		if c := cmp.Compare(roleOrder[a.Role], roleOrder[b.Role]); c != 0 {
			return c
		}
		if a.ID == caller && b.ID != caller {
			return -1
		}
		if b.ID == caller && a.ID != caller {
			return 1
		}
		if c := collation.Compare(a.Name, b.Name); c != 0 {
			return c
		}

		return cmp.Compare(a.ID.String(), b.ID.String())
	})
}

func find(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*album.Album, error) {
	a, err := gh.Repositories(rt).Album.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrAlbumNotFound
	}

	return a, err
}

func midnight(date string) *string {
	if date == "" {
		return nil
	}
	s := date + "T00:00:00.000Z"

	return &s
}

func withMembers(ctx context.Context, rt *runtime.Runtime, list []album.Album, caller uuid.UUID, sharedLinks bool) ([]Details, error) {
	repos := gh.Repositories(rt)
	ids := make([]uuid.UUID, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}

	members, err := repos.Album.Members(ctx, ids)
	if err != nil {
		return nil, err
	}
	links := map[uuid.UUID]bool{}
	if sharedLinks {
		if links, err = repos.Album.WithSharedLinks(ctx, ids); err != nil {
			return nil, err
		}
	}

	out := make([]Details, 0, len(list))
	for i := range list {
		m := members[list[i].ID]
		sortMembers(m, caller)
		out = append(out, Details{Album: &list[i], Members: m, HasSharedLink: links[list[i].ID]})
	}

	return out, nil
}

func addMetadata(ctx context.Context, rt *runtime.Runtime, list []Details) error {
	ids := make([]uuid.UUID, 0, len(list))
	for _, d := range list {
		ids = append(ids, d.Album.ID)
	}

	meta, err := gh.Repositories(rt).Album.Metadata(ctx, ids)
	if err != nil {
		return err
	}
	for i := range list {
		m, ok := meta[list[i].Album.ID]
		if !ok {
			continue
		}
		modified := m.LastModified
		list[i].StartDate, list[i].EndDate = midnight(m.StartDate), midnight(m.EndDate)
		list[i].AssetCount = m.AssetCount
		list[i].LastModified = &modified
	}

	return nil
}

func addAggregated(ctx context.Context, rt *runtime.Runtime, d *Details) error {
	assets, err := gh.Repositories(rt).Album.CountedAssets(ctx, d.Album.ID)
	if err != nil {
		return err
	}

	d.AssetCount = int64(len(assets))
	if len(assets) == 0 {
		return nil
	}

	start := jsontime.FormatPostgres(assets[0].LocalDateTime)
	end := jsontime.FormatPostgres(assets[len(assets)-1].LocalDateTime)
	if start > end {
		start, end = end, start
	}
	d.StartDate, d.EndDate = &start, &end

	return nil
}

func load(ctx context.Context, rt *runtime.Runtime, a *album.Album, caller uuid.UUID, sharedLinks bool) (*Details, error) {
	list, err := withMembers(ctx, rt, []album.Album{*a}, caller, sharedLinks)
	if err != nil {
		return nil, err
	}
	if err = addAggregated(ctx, rt, &list[0]); err != nil {
		return nil, err
	}

	return &list[0], nil
}
