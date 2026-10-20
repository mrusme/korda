package search

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/checksum"
	"tty.fail/mrus/korda/helpers/collation"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/search"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	defaultSize      = 250
	defaultSmartSize = 100
	exploreFields    = 12
)

type Page struct {
	Items      []asset.Asset
	WithExif   bool
	NextPage   *string
	NextCursor *string
}

func userIDs(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, v *asset.Visibility) ([]uuid.UUID, error) {
	if v != nil && *v == asset.VisibilityLocked {
		return []uuid.UUID{a.User.ID}, nil
	}

	partners, err := gh.Repositories(rt).Partner.InTimeline(ctx, a.User.ID)
	if err != nil {
		return nil, err
	}

	return append([]uuid.UUID{a.User.ID}, partners...), nil
}

func size(s *int64, fallback int) int {
	if s == nil {
		return fallback
	}

	return int(*s)
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

func newShape(filter *search.Filter, orderBy *search.Order, cursor *string) bool {
	return filter != nil || orderBy != nil || cursor != nil
}

func visibility(a *auth.Auth, v *asset.Visibility) string {
	switch {
	case v != nil:
		return string(*v)
	case a.Elevated:
		return ""
	}

	return "not-locked"
}

func flat(a *auth.Auth, b *search.Base) repoasset.Flat {
	f := repoasset.Flat{
		Visibility:    visibility(a, b.Visibility),
		AlbumIDs:      parseIDs(b.AlbumIDs),
		PersonIDs:     parseIDs(b.PersonIDs),
		CreatedBefore: b.CreatedBefore,
		CreatedAfter:  b.CreatedAfter,
		UpdatedBefore: b.UpdatedBefore,
		UpdatedAfter:  b.UpdatedAfter,
		TrashedBefore: b.TrashedBefore,
		TrashedAfter:  b.TrashedAfter,
		TakenBefore:   b.TakenBefore,
		TakenAfter:    b.TakenAfter,
		City:          b.City,
		State:         b.State,
		Country:       b.Country,
		Make:          b.Make,
		Model:         b.Model,
		LensModel:     b.LensModel,
		Rating:        b.Rating,
		Type:          b.Type,
		IsFavorite:    b.IsFavorite,
		IsOffline:     b.IsOffline,
		IsEncoded:     b.IsEncoded,
		IsMotion:      b.IsMotion,
		IsNotInAlbum:  b.IsNotInAlbum != nil && *b.IsNotInAlbum,
	}
	if b.TagIDs.Valid() {
		f.TagIDs = parseIDs(b.TagIDs.V)
	}
	if b.TagIDs.Set && b.TagIDs.Null {
		f.NoTags = true
	}
	if b.LibraryID.Valid() {
		if id, err := uuid.Parse(b.LibraryID.V); err == nil {
			f.LibraryID = &id
		}
	}
	if b.Ocr != nil {
		f.OCR = *b.Ocr
	}

	return f
}

func requireLocked(a *auth.Auth, v *asset.Visibility) error {
	if v != nil && *v == asset.VisibilityLocked && !a.Elevated {
		return errs.ErrElevatedRequired
	}

	return nil
}

func canMatchLocked(f *search.VisibilityFilter) bool {
	locked := asset.VisibilityLocked
	return (f.Eq == nil || *f.Eq == locked) &&
		(f.Ne == nil || *f.Ne != locked) &&
		(f.In == nil || slices.Contains(f.In, locked)) &&
		(f.NotIn == nil || !slices.Contains(f.NotIn, locked))
}

func lockedPolicy(a *auth.Auth, f *search.Filter) error {
	if a.Elevated {
		return nil
	}

	var deciding []*search.VisibilityFilter
	if f.Visibility != nil {
		deciding = append(deciding, f.Visibility)
	} else {
		for i := range f.Or {
			if f.Or[i].Visibility != nil {
				deciding = append(deciding, f.Or[i].Visibility)
			}
		}
	}
	if slices.ContainsFunc(deciding, canMatchLocked) {
		return errs.ErrElevatedRequired
	}

	if f.Visibility == nil {
		locked := asset.VisibilityLocked
		f.Visibility = &search.VisibilityFilter{Ne: &locked}
	}

	return nil
}

func scopeV3(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, filter *search.Filter) (*search.Filter, repoasset.Scope, error) {
	f := &search.Filter{}
	if filter != nil {
		copied := *filter
		f = &copied
	}

	if err := lockedPolicy(a, f); err != nil {
		return nil, repoasset.Scope{}, err
	}

	ids := map[string]bool{}
	branches := append([]search.Conditions{f.Conditions}, conditions(f.Or)...)
	for _, b := range branches {
		if b.AlbumIDs == nil {
			continue
		}
		for _, list := range [][]string{b.AlbumIDs.Any, b.AlbumIDs.All, b.AlbumIDs.None} {
			for _, id := range list {
				ids[id] = true
			}
		}
	}
	if len(ids) > 0 {
		var albumIDs []uuid.UUID
		for id := range ids {
			albumIDs = append(albumIDs, uuid.MustParse(id))
		}
		if err := access.Require(ctx, rt, a, permission.AlbumRead, albumIDs...); err != nil {
			return nil, repoasset.Scope{}, err
		}
	}

	confined := f.AlbumConfined()
	if !confined && len(f.Or) > 0 {
		confined = true
		for i := range f.Or {
			confined = confined && f.Or[i].AlbumConfined()
		}
	}
	if a.SharedLink != nil && !confined {
		return nil, repoasset.Scope{}, errs.ErrSharedLinkSearch
	}

	users := []uuid.UUID{a.User.ID}
	if !confined {
		var err error
		if users, err = userIDs(ctx, rt, a, nil); err != nil {
			return nil, repoasset.Scope{}, err
		}
	}

	return f, repoasset.Scope{UserIDs: users, OwnerID: a.User.ID}, nil
}

func conditions(branches []search.Branch) []search.Conditions {
	out := make([]search.Conditions, 0, len(branches))
	for _, b := range branches {
		out = append(out, b.Conditions)
	}

	return out
}

type cursorPayload struct {
	Offset *json.Number `json:"offset"`
}

func DecodeCursor(cursor *string) (int, error) {
	if cursor == nil {
		return 0, nil
	}

	raw := strings.TrimRight(*cursor, "=")
	data, err := base64.RawURLEncoding.DecodeString(strings.NewReplacer("+", "-", "/", "_").Replace(raw))
	if err != nil {
		return 0, errs.ErrInvalidCursor
	}

	var p cursorPayload
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err = dec.Decode(&p); err != nil || p.Offset == nil {
		return 0, errs.ErrInvalidCursor
	}
	offset, err := strconv.ParseInt(p.Offset.String(), 10, 64)
	if err != nil || offset < 0 {
		return 0, errs.ErrInvalidCursor
	}

	return int(offset), nil
}

func EncodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(`{"offset":` + strconv.Itoa(offset) + `}`))
}

func Metadata(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *search.Metadata) (*Page, error) {
	repos := gh.Repositories(rt)
	n := size(in.Size, defaultSize)
	withExif := in.WithExif != nil && *in.WithExif

	if newShape(in.Filter, in.OrderBy, in.Cursor) {
		filter, scope, err := scopeV3(ctx, rt, a, in.Filter)
		if err != nil {
			return nil, err
		}
		offset, err := DecodeCursor(in.Cursor)
		if err != nil {
			return nil, err
		}

		items, err := repos.Asset.SearchTree(ctx, filter, scope, in.WithStacked, in.OrderBy, n+1, offset)
		if err != nil {
			return nil, err
		}

		page := &Page{Items: items, WithExif: withExif}
		if len(items) > n {
			page.Items = items[:n]
			next := EncodeCursor(offset + n)
			page.NextCursor = &next
		}
		return page, nil
	}

	if err := requireLocked(a, in.Visibility); err != nil {
		return nil, err
	}

	f := flat(a, &in.Base)
	switch {
	case len(f.AlbumIDs) > 0:
		if err := access.Require(ctx, rt, a, permission.AlbumRead, f.AlbumIDs...); err != nil {
			return nil, err
		}
	case a.SharedLink != nil:
		return nil, errs.ErrSharedLinkSearch
	default:
		users, err := userIDs(ctx, rt, a, in.Visibility)
		if err != nil {
			return nil, err
		}
		f.UserIDs = users
	}

	if in.Checksum != nil {
		f.Checksum = checksum.Decode(*in.Checksum)
	}
	if in.ID != nil {
		if id, err := uuid.Parse(*in.ID); err == nil {
			f.ID = &id
		}
	}
	for dst, src := range map[*string]*string{
		&f.Description: in.Description, &f.OriginalFileName: in.OriginalFileName,
		&f.OriginalPath: in.OriginalPath, &f.EncodedVideoPath: in.EncodedVideoPath,
	} {
		if src != nil {
			*dst = *src
		}
	}
	f.WithDeleted = in.WithDeleted != nil && *in.WithDeleted
	f.WithExif = withExif
	f.WithStacked = in.WithStacked

	page := 1
	if in.Page != nil {
		page = int(*in.Page)
	}
	desc := in.Order == nil || *in.Order == search.Desc

	items, err := repos.Asset.SearchFlat(ctx, f, desc, n+1, (page-1)*n)
	if err != nil {
		return nil, err
	}

	out := &Page{Items: items, WithExif: withExif}
	if len(items) > n {
		out.Items = items[:n]
		next := strconv.Itoa(page + 1)
		out.NextPage = &next
	}

	return out, nil
}

func Statistics(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *search.Statistics) (int64, error) {
	repos := gh.Repositories(rt)

	if in.Filter != nil {
		filter, scope, err := scopeV3(ctx, rt, a, in.Filter)
		if err != nil {
			return 0, err
		}
		return repos.Asset.CountTree(ctx, filter, scope)
	}

	if err := requireLocked(a, in.Visibility); err != nil {
		return 0, err
	}

	f := flat(a, &in.Base)
	users, err := userIDs(ctx, rt, a, in.Visibility)
	if err != nil {
		return 0, err
	}
	f.UserIDs = users
	if in.Description != nil {
		f.Description = *in.Description
	}

	return repos.Asset.CountFlat(ctx, f)
}

func Random(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *search.Random) ([]asset.Asset, error) {
	repos := gh.Repositories(rt)
	n := size(in.Size, defaultSize)

	if in.Filter != nil {
		filter, scope, err := scopeV3(ctx, rt, a, in.Filter)
		if err != nil {
			return nil, err
		}
		return repos.Asset.RandomTree(ctx, filter, scope, in.WithStacked, n)
	}

	if err := requireLocked(a, in.Visibility); err != nil {
		return nil, err
	}

	f := flat(a, &in.Base)
	users, err := userIDs(ctx, rt, a, in.Visibility)
	if err != nil {
		return nil, err
	}
	f.UserIDs = users
	f.WithDeleted = in.WithDeleted != nil && *in.WithDeleted
	f.WithExif = in.WithExif != nil && *in.WithExif
	f.WithStacked = in.WithStacked

	return repos.Asset.RandomFlat(ctx, f, n)
}

func Large(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *search.Large) ([]asset.Asset, error) {
	if err := requireLocked(a, in.Visibility); err != nil {
		return nil, err
	}

	f := flat(a, &in.Base)
	users, err := userIDs(ctx, rt, a, in.Visibility)
	if err != nil {
		return nil, err
	}
	f.UserIDs = users
	f.WithDeleted = in.WithDeleted != nil && *in.WithDeleted
	f.WithExif = true

	var minimum int64
	if in.MinFileSize != nil {
		minimum = *in.MinFileSize
	}

	return gh.Repositories(rt).Asset.LargeFlat(ctx, f, size(in.Size, defaultSize), minimum)
}

func Smart(a *auth.Auth, in *search.Smart) error {
	if in.Filter == nil {
		if err := requireLocked(a, in.Visibility); err != nil {
			return err
		}
	}

	return errs.ErrSmartSearchDisabled
}

func Recent(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]asset.Asset, error) {
	return gh.Repositories(rt).Asset.RecentlyCreated(ctx, a.User.ID, exploreFields)
}

func Cities(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]asset.Asset, error) {
	users, err := userIDs(ctx, rt, a, nil)
	if err != nil {
		return nil, err
	}

	return gh.Repositories(rt).Asset.ByCity(ctx, users)
}

var suggestionColumns = map[search.SuggestionType]string{
	search.SuggestCountry:   "country",
	search.SuggestState:     "state",
	search.SuggestCity:      "city",
	search.SuggestMake:      "make",
	search.SuggestModel:     "model",
	search.SuggestLensModel: "lens_model",
}

func value(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func Suggestions(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *search.Suggestions) ([]*string, error) {
	narrow := map[string]string{}
	switch in.Type {
	case search.SuggestState:
		narrow["country"] = value(in.Country)
	case search.SuggestCity:
		narrow["country"], narrow["state"] = value(in.Country), value(in.State)
	case search.SuggestMake:
		narrow["model"], narrow["lens_model"] = value(in.Model), value(in.LensModel)
	case search.SuggestModel:
		narrow["make"], narrow["lens_model"] = value(in.Make), value(in.LensModel)
	case search.SuggestLensModel:
		narrow["make"], narrow["model"] = value(in.Make), value(in.Model)
	}

	users, err := userIDs(ctx, rt, a, nil)
	if err != nil {
		return nil, err
	}
	values, err := gh.Repositories(rt).Asset.Suggestions(ctx, users, suggestionColumns[in.Type], narrow)
	if err != nil {
		return nil, err
	}

	collation.Sort(values)

	out := make([]*string, 0, len(values)+1)
	for i := range values {
		out = append(out, &values[i])
	}
	if in.IncludeNull != nil && *in.IncludeNull {
		out = append(out, nil)
	}

	return out, nil
}
