package memories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/jsonb"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/memory"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/systemmetadata"
	repomemory "tty.fail/mrus/korda/services/repositories/memory"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	days         = 3
	keepUnsaved  = 30 * 24 * time.Hour
	stateLayout  = "2006-01-02T15:04:05.000Z"
	searchLayout = time.DateOnly
)

type SearchInput struct {
	ID         *string       `json:"id" v:"uuid"`
	Type       *memory.Type  `json:"type"`
	For        *string       `json:"for" v:"date"`
	IsTrashed  *bool         `json:"isTrashed"`
	IsSaved    *bool         `json:"isSaved"`
	IsUpcoming *bool         `json:"isUpcoming"`
	Size       *int64        `json:"size" v:"coerce,min=1"`
	Page       *int64        `json:"page" v:"coerce,min=1"`
	Order      *memory.Order `json:"order"`
}

type DataInput struct {
	Year       int64   `json:"year" v:"min=1000,max=9999"`
	PersonID   *string `json:"personId" v:"uuid"`
	PersonName *string `json:"personName"`
}

type CreateInput struct {
	Type     memory.Type `json:"type"`
	Data     DataInput   `json:"data"`
	MemoryAt time.Time   `json:"memoryAt" v:"datetime"`
	AssetIDs []string    `json:"assetIds" v:"optional" vi:"uuid"`
	IsSaved  *bool       `json:"isSaved"`
	SeenAt   *time.Time  `json:"seenAt" v:"datetime"`
	ShowAt   *time.Time  `json:"showAt" v:"datetime"`
	HideAt   *time.Time  `json:"hideAt" v:"datetime"`
}

func (CreateInput) Refine(in refine.Fields) []refine.Issue {
	if kind, _ := in.Get("type"); kind != string(memory.TypeBirthday) {
		return nil
	}

	data, _ := in.Get("data")
	fields, _ := data.(refine.Fields)
	var out []refine.Issue
	for _, key := range []string{"personId", "personName"} {
		if fields == nil || !fields.Has(key) {
			out = append(out, refine.At([]any{"data", key}, "Required for birthday memories"))
		}
	}

	return out
}

type UpdateInput struct {
	IsSaved  *bool      `json:"isSaved"`
	SeenAt   *time.Time `json:"seenAt" v:"datetime"`
	MemoryAt *time.Time `json:"memoryAt" v:"datetime"`
}

func (UpdateInput) Refine(in refine.Fields) []refine.Issue {
	if slices.ContainsFunc([]string{"isSaved", "seenAt", "memoryAt"}, in.Has) {
		return nil
	}

	return []refine.Issue{refine.Root("At least one of the following fields is required: isSaved, seenAt, memoryAt")}
}

type BulkIDsInput struct {
	IDs []string `json:"ids" vi:"uuid"`
}

type Details struct {
	Memory *memory.Memory
	Assets []asset.Asset
}

type state struct {
	LastOnThisDayDate string `json:"lastOnThisDayDate"`
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

func filter(in *SearchInput) (repomemory.Filter, error) {
	f := repomemory.Filter{Type: in.Type, IsSaved: in.IsSaved, IsUpcoming: in.IsUpcoming, Now: now(), Size: in.Size,
		Order: memory.OrderDesc}
	if in.ID != nil {
		f.ID = uuid.NullUUID{UUID: uuid.MustParse(*in.ID), Valid: true}
	}
	if in.For != nil {
		t, err := time.Parse(searchLayout, *in.For)
		if err != nil {
			return f, err
		}
		f.For = &t
	}
	f.IsTrashed = in.IsTrashed != nil && *in.IsTrashed
	if in.Size != nil {
		f.Page = in.Page
	}
	if in.Order != nil {
		f.Order = *in.Order
	}

	return f, nil
}

func withAssets(ctx context.Context, rt *runtime.Runtime, list []memory.Memory) ([]Details, error) {
	ids := make([]uuid.UUID, 0, len(list))
	for _, m := range list {
		ids = append(ids, m.ID)
	}

	assets, err := gh.Repositories(rt).Memory.Assets(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]Details, 0, len(list))
	for i := range list {
		out = append(out, Details{Memory: &list[i], Assets: assets[list[i].ID]})
	}

	return out, nil
}

func load(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*Details, error) {
	m, err := gh.Repositories(rt).Memory.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrMemoryNotFound
	}
	if err != nil {
		return nil, err
	}

	list, err := withAssets(ctx, rt, []memory.Memory{*m})
	if err != nil {
		return nil, err
	}

	return &list[0], nil
}

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]Details, error) {
	f, err := filter(in)
	if err != nil {
		return nil, err
	}

	list, err := gh.Repositories(rt).Memory.Search(ctx, a.User.ID, f)
	if err != nil {
		return nil, err
	}

	all, err := withAssets(ctx, rt, list)
	if err != nil {
		return nil, err
	}

	out := make([]Details, 0, len(all))
	for _, d := range all {
		if len(d.Assets) > 0 {
			out = append(out, d)
		}
	}

	return out, nil
}

func Statistics(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) (int64, error) {
	f, err := filter(in)
	if err != nil {
		return 0, err
	}

	return gh.Repositories(rt).Memory.Count(ctx, a.User.ID, f)
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.MemoryRead, id); err != nil {
		return nil, err
	}

	return load(ctx, rt, id)
}

func nullTime(t *time.Time) sql.Null[time.Time] {
	if t == nil {
		return sql.Null[time.Time]{}
	}

	return sql.Null[time.Time]{V: t.UTC(), Valid: true}
}

func encodeData(in *DataInput) (string, error) {
	data := map[string]any{"year": in.Year}
	if in.PersonID != nil {
		data["personId"] = *in.PersonID
	}
	if in.PersonName != nil {
		data["personName"] = *in.PersonName
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	normalized, err := jsonb.Normalize(encoded)

	return string(normalized), err
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*Details, error) {
	assetIDs, err := access.Allowed(ctx, rt, a, permission.AssetUpdate, parseIDs(in.AssetIDs)...)
	if err != nil {
		return nil, err
	}

	data, err := encodeData(&in.Data)
	if err != nil {
		return nil, err
	}

	id, err := gh.Repositories(rt).Memory.Create(ctx, repomemory.Create{
		OwnerID: a.User.ID, Type: in.Type, Data: data, IsSaved: in.IsSaved != nil && *in.IsSaved,
		MemoryAt: in.MemoryAt.UTC(), SeenAt: nullTime(in.SeenAt), ShowAt: nullTime(in.ShowAt), HideAt: nullTime(in.HideAt),
		AssetIDs: assetIDs,
	}, now())
	if err != nil {
		return nil, err
	}

	return load(ctx, rt, id)
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*Details, error) {
	if err := access.Require(ctx, rt, a, permission.MemoryUpdate, id); err != nil {
		return nil, err
	}

	u := repomemory.Update{IsSaved: in.IsSaved, SeenAt: in.SeenAt, MemoryAt: in.MemoryAt}
	if err := gh.Repositories(rt).Memory.Update(ctx, id, u, now()); err != nil {
		return nil, err
	}

	return load(ctx, rt, id)
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.MemoryDelete, id); err != nil {
		return err
	}

	return gh.Repositories(rt).Memory.Delete(ctx, id, now())
}

func AddAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *BulkIDsInput) ([]bulk.Result, error) {
	if err := access.Require(ctx, rt, a, permission.MemoryRead, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	ids := parseIDs(in.IDs)
	present, err := repos.Memory.AssetIDs(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	allowed, err := access.Allowed(ctx, rt, a, permission.AssetUpdate, bulk.Missing(ids, present)...)
	if err != nil {
		return nil, err
	}

	results, added := bulk.Add(ids, present, allowed)
	if len(added) == 0 {
		return results, nil
	}

	at := now()
	if err = repos.Memory.AddAssets(ctx, id, added, at); err != nil {
		return nil, err
	}

	return results, repos.Memory.Update(ctx, id, repomemory.Update{}, at)
}

func RemoveAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *BulkIDsInput) ([]bulk.Result, error) {
	if err := access.Require(ctx, rt, a, permission.MemoryUpdate, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	ids := parseIDs(in.IDs)
	canAlways, err := access.Allowed(ctx, rt, a, permission.MemoryDelete, id)
	if err != nil {
		return nil, err
	}
	present, err := repos.Memory.AssetIDs(ctx, id, ids)
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
	if err = repos.Memory.RemoveAssets(ctx, id, removed, at); err != nil {
		return nil, err
	}

	return results, repos.Memory.Update(ctx, id, repomemory.Update{}, at)
}

func Cleanup(ctx context.Context, rt *runtime.Runtime) error {
	at := now()

	return gh.Repositories(rt).Memory.Cleanup(ctx, at.Add(-keepUnsaved), at)
}

func Generate(ctx context.Context, rt *runtime.Runtime) error {
	return GenerateAt(ctx, rt, now())
}

func GenerateAt(ctx context.Context, rt *runtime.Runtime, at time.Time) error {
	repos := gh.Repositories(rt)
	list, err := repos.User.List(ctx, uuid.NullUUID{}, false)
	if err != nil {
		return err
	}
	users := make([]uuid.UUID, 0, len(list))
	for _, u := range list {
		users = append(users, u.ID)
	}

	var last *time.Time
	if raw, err := repos.SystemMetadata.Get(ctx, systemmetadata.MemoriesState); err == nil {
		var s state
		if err = json.Unmarshal([]byte(raw), &s); err == nil && s.LastOnThisDayDate != "" {
			last = parseState(s.LastOnThisDayDate)
		}
	} else if !errors.Is(err, errs.ErrNoRows) {
		return err
	}

	at = at.UTC()
	start := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -days)
	if last == nil {
		last = &start
	}

	for i := 0; i <= days*2; i++ {
		target := start.AddDate(0, 0, i)
		if !last.Before(target) {
			continue
		}

		var failed error
		for _, owner := range users {
			if err = onThisDay(ctx, rt, owner, target); err != nil && failed == nil {
				failed = err
			}
		}
		if failed != nil {
			rt.Error("memories", "generate", "target", target.Format(stateLayout), "error", failed)
		}

		encoded, err := json.Marshal(state{LastOnThisDayDate: target.Format(stateLayout)})
		if err != nil {
			return err
		}
		if err = repos.SystemMetadata.Set(ctx, systemmetadata.MemoriesState, string(encoded)); err != nil {
			return err
		}
	}

	return nil
}

func parseState(value string) *time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.DateOnly} {
		if t, err := time.Parse(layout, value); err == nil {
			return &t
		}
	}

	return &time.Time{}
}

func onThisDay(ctx context.Context, rt *runtime.Runtime, owner uuid.UUID, target time.Time) error {
	repos := gh.Repositories(rt)
	years, err := repos.Memory.OnThisDay(ctx, owner, target.Month(), target.Day(), target.Year())
	if err != nil {
		return err
	}

	showAt := target
	hideAt := target.Add(24*time.Hour - time.Millisecond)
	for _, y := range years {
		data, err := encodeData(&DataInput{Year: int64(y.Year)})
		if err != nil {
			return err
		}
		memoryAt := time.Date(y.Year, target.Month(), target.Day(), 0, 0, 0, 0, time.UTC)
		if _, err = repos.Memory.Create(ctx, repomemory.Create{
			OwnerID: owner, Type: memory.TypeOnThisDay, Data: data, MemoryAt: memoryAt,
			ShowAt: sql.Null[time.Time]{V: showAt, Valid: true}, HideAt: sql.Null[time.Time]{V: hideAt, Valid: true},
			AssetIDs: y.Assets,
		}, now()); err != nil {
			return err
		}
	}

	return nil
}
