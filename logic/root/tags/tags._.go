package tags

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/collation"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/tag"
	repoexif "tty.fail/mrus/korda/services/repositories/assetexif"
	repotag "tty.fail/mrus/korda/services/repositories/tag"
	"xn--gckvb8fzb.com/glides/runtime"
)

type CreateInput struct {
	Name     string                 `json:"name" v:"regex=tagName"`
	ParentID optional.Value[string] `json:"parentId" v:"nullable,uuid"`
	Color    optional.Value[string] `json:"color" v:"nullable,regex=hexColor"`
}

type UpdateInput struct {
	Name  *string                `json:"name" v:"regex=tagName"`
	Color optional.Value[string] `json:"color" v:"nullable,regex=hexColor"`
}

type UpsertInput struct {
	Tags []string `json:"tags"`
}

type BulkAssetsInput struct {
	TagIDs   []string `json:"tagIds" vi:"uuid"`
	AssetIDs []string `json:"assetIds" vi:"uuid"`
}

type BulkIDsInput struct {
	IDs []string `json:"ids" vi:"uuid"`
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

func color(v optional.Value[string]) optional.Value[string] {
	if v.Set && !v.Null && !strings.HasPrefix(v.V, "#") {
		v.V = "#" + v.V
	}

	return v
}

func find(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*tag.Tag, error) {
	t, err := gh.Repositories(rt).Tag.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrTagNotFound
	}

	return t, err
}

func All(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]tag.Tag, error) {
	list, err := gh.Repositories(rt).Tag.All(ctx, a.User.ID)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(list, func(x, y tag.Tag) int { return collation.Compare(x.Value, y.Value) })

	return list, nil
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*tag.Tag, error) {
	if err := access.Require(ctx, rt, a, permission.TagRead, id); err != nil {
		return nil, err
	}

	return find(ctx, rt, id)
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*tag.Tag, error) {
	repos := gh.Repositories(rt)
	value := in.Name
	var parentID uuid.NullUUID
	if in.ParentID.Set && !in.ParentID.Null && in.ParentID.V != "" {
		id := uuid.MustParse(in.ParentID.V)
		if err := access.Require(ctx, rt, a, permission.TagRead, id); err != nil {
			return nil, err
		}
		parent, err := find(ctx, rt, id)
		if err != nil {
			return nil, err
		}
		value = parent.Value + "/" + in.Name
		parentID = uuid.NullUUID{UUID: parent.ID, Valid: true}
	}

	if _, err := repos.Tag.ByValue(ctx, a.User.ID, value); err == nil {
		return nil, errs.ErrTagExists
	} else if !errors.Is(err, errs.ErrNoRows) {
		return nil, err
	}

	var c sql.Null[string]
	if v := color(in.Color); v.Set && !v.Null {
		c = sql.Null[string]{V: v.V, Valid: true}
	}

	return repos.Tag.Create(ctx, a.User.ID, value, c, parentID, now())
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*tag.Tag, error) {
	if err := access.Require(ctx, rt, a, permission.TagUpdate, id); err != nil {
		return nil, err
	}

	existing, err := find(ctx, rt, id)
	if err != nil {
		return nil, err
	}

	value := existing.Value
	if in.Name != nil && *in.Name != "" {
		parts := strings.Split(existing.Value, "/")
		parts[len(parts)-1] = *in.Name
		value = strings.Join(parts, "/")
	}

	repos := gh.Repositories(rt)
	var assetIDs []uuid.UUID
	if value != existing.Value {
		if assetIDs, err = repos.Tag.AssetIDsDeep(ctx, id); err != nil {
			return nil, err
		}
	}

	updated, err := repos.Tag.Update(ctx, id, value, color(in.Color), now())
	if err != nil {
		return nil, err
	}

	return updated, syncAssetTags(ctx, rt, assetIDs)
}

func Upsert(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UpsertInput) ([]tag.Tag, error) {
	return UpsertValues(ctx, rt, a.User.ID, in.Tags)
}

func UpsertValues(ctx context.Context, rt *runtime.Runtime, userID uuid.UUID, values []string) ([]tag.Tag, error) {
	repos := gh.Repositories(rt)
	at := now()
	seen := map[string]bool{}
	out := []tag.Tag{}
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true

		var parent *tag.Tag
		for part := range strings.SplitSeq(v, "/") {
			if part == "" {
				continue
			}
			value := part
			var parentID uuid.NullUUID
			if parent != nil {
				value = parent.Value + "/" + part
				parentID = uuid.NullUUID{UUID: parent.ID, Valid: true}
			}
			next, err := repos.Tag.UpsertValue(ctx, nil, userID, value, parentID, at)
			if err != nil {
				return nil, err
			}
			parent = next
		}
		if parent != nil {
			out = append(out, *parent)
		}
	}

	return out, nil
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.TagDelete, id); err != nil {
		return err
	}

	repos := gh.Repositories(rt)
	assetIDs, err := repos.Tag.AssetIDsDeep(ctx, id)
	if err != nil {
		return err
	}
	if err = repos.Tag.Delete(ctx, id); err != nil {
		return err
	}

	return syncAssetTags(ctx, rt, assetIDs)
}

func BulkTag(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BulkAssetsInput) (int, error) {
	repos := gh.Repositories(rt)
	tagIDs, err := repos.Tag.OwnedBy(ctx, a.User.ID, dedupe(parseIDs(in.TagIDs)))
	if err != nil {
		return 0, err
	}
	assetIDs, err := access.Allowed(ctx, rt, a, permission.AssetUpdate, dedupe(parseIDs(in.AssetIDs))...)
	if err != nil {
		return 0, err
	}

	pairs := make([]repotag.Pair, 0, len(tagIDs)*len(assetIDs))
	for _, t := range tagIDs {
		for _, asset := range assetIDs {
			pairs = append(pairs, repotag.Pair{AssetID: asset, TagID: t})
		}
	}

	inserted, err := repos.Tag.UpsertAssets(ctx, pairs)
	if err != nil {
		return 0, err
	}

	done := map[uuid.UUID]bool{}
	for _, p := range inserted {
		if done[p.AssetID] {
			continue
		}
		done[p.AssetID] = true
		if err = updateTags(ctx, rt, p.AssetID); err != nil {
			return 0, err
		}
	}

	return len(inserted), nil
}

func dedupe(ids []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}

	return out
}

func AddAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *BulkIDsInput) ([]bulk.Result, error) {
	if err := access.Require(ctx, rt, a, permission.TagAsset, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	ids := parseIDs(in.IDs)
	present, err := repos.Tag.AssetIDs(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	allowed, err := access.Allowed(ctx, rt, a, permission.AssetUpdate, bulk.Missing(ids, present)...)
	if err != nil {
		return nil, err
	}

	results, added := bulk.Add(ids, present, allowed)
	if len(added) > 0 {
		if err = repos.Tag.AddAssets(ctx, id, added); err != nil {
			return nil, err
		}
	}
	for _, assetID := range added {
		if err = updateTags(ctx, rt, assetID); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func RemoveAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *BulkIDsInput) ([]bulk.Result, error) {
	if err := access.Require(ctx, rt, a, permission.TagAsset, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	ids := parseIDs(in.IDs)
	present, err := repos.Tag.AssetIDs(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	allowed := present
	canAlways, err := access.Allowed(ctx, rt, a, permission.TagDelete, id)
	if err != nil {
		return nil, err
	}
	if len(canAlways) == 0 {
		if allowed, err = access.Allowed(ctx, rt, a, permission.AssetShare, present...); err != nil {
			return nil, err
		}
	}

	results, removed := bulk.Remove(ids, present, allowed)
	if len(removed) > 0 {
		if err = repos.Tag.RemoveAssets(ctx, id, removed); err != nil {
			return nil, err
		}
	}
	for _, assetID := range removed {
		if err = updateTags(ctx, rt, assetID); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func updateTags(ctx context.Context, rt *runtime.Runtime, assetID uuid.UUID) error {
	repos := gh.Repositories(rt)
	byAsset, err := repos.Tag.ForAssets(ctx, []uuid.UUID{assetID})
	if err != nil {
		return err
	}

	values := make([]string, 0, len(byAsset[assetID]))
	for _, t := range byAsset[assetID] {
		values = append(values, t.Value)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return err
	}

	return repos.AssetExif.Write(ctx, nil, assetID, []repoexif.Write{{Column: "tags", Value: string(encoded)}},
		repoexif.Append, []string{"tags"}, now())
}

func syncAssetTags(ctx context.Context, rt *runtime.Runtime, assetIDs []uuid.UUID) error {
	for _, id := range assetIDs {
		if err := updateTags(ctx, rt, id); err != nil {
			return err
		}
	}

	return nil
}

func ApplyKeywords(ctx context.Context, rt *runtime.Runtime, assetID uuid.UUID, ownerID uuid.UUID, keywords []string) error {
	list, err := UpsertValues(ctx, rt, ownerID, keywords)
	if err != nil {
		return err
	}

	ids := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		ids = append(ids, t.ID)
	}

	return gh.Repositories(rt).Tag.ReplaceAssetTags(ctx, nil, assetID, ids)
}
