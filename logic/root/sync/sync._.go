package sync

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	modelsync "tty.fail/mrus/korda/models/sync"
	"tty.fail/mrus/korda/models/synccheckpoint"
	reposync "tty.fail/mrus/korda/services/repositories/sync"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	pageSize = 1000
	maxDays  = 30
)

var ErrInvalidCheckpoint = errors.New("sync checkpoint is not a UUID")

type Line struct {
	Type modelsync.EntityType
	Data any
	Ack  string
}

type Plan struct {
	reset     bool
	nowID     string
	userID    uuid.UUID
	sessionID uuid.UUID
	types     []modelsync.RequestType
	acks      map[modelsync.EntityType]modelsync.Ack
}

var checkpointTypes = map[modelsync.RequestType][]modelsync.EntityType{
	modelsync.ReqAuthUsersV1:     {modelsync.AuthUserV1},
	modelsync.ReqAuthUsersV2:     {modelsync.AuthUserV2},
	modelsync.ReqUsersV1:         {modelsync.UserDeleteV1, modelsync.UserV1},
	modelsync.ReqPartnersV1:      {modelsync.PartnerDeleteV1, modelsync.PartnerV1},
	modelsync.ReqAssetsV2:        {modelsync.AssetDeleteV1, modelsync.AssetV2},
	modelsync.ReqStacksV1:        {modelsync.StackDeleteV1, modelsync.StackV1},
	modelsync.ReqPartnerAssetsV2: {modelsync.PartnerAssetDeleteV1, modelsync.PartnerAssetBackfillV2, modelsync.PartnerAssetV2},
	modelsync.ReqPartnerStacksV1: {modelsync.PartnerStackDeleteV1, modelsync.PartnerStackBackfillV1, modelsync.PartnerStackV1},
	modelsync.ReqAlbumAssetsV2:   {modelsync.AlbumAssetBackfillV2, modelsync.AlbumAssetUpdateV2, modelsync.AlbumAssetCreateV2},
	modelsync.ReqAlbumsV1:        {modelsync.AlbumDeleteV1, modelsync.AlbumV1},
	modelsync.ReqAlbumsV2:        {modelsync.AlbumDeleteV1, modelsync.AlbumV2},
	modelsync.ReqAlbumUsersV1:    {modelsync.AlbumUserDeleteV1, modelsync.AlbumUserBackfillV1, modelsync.AlbumUserV1},
	modelsync.ReqAlbumToAssetsV1: {modelsync.AlbumToAssetDeleteV1, modelsync.AlbumToAssetBackfillV1, modelsync.AlbumToAssetV1},
	modelsync.ReqAssetExifsV1:    {modelsync.AssetExifV1},
	modelsync.ReqAlbumAssetExifsV1: {modelsync.AlbumAssetExifBackfillV1, modelsync.AlbumAssetExifUpdateV1,
		modelsync.AlbumAssetExifCreateV1},
	modelsync.ReqAssetOcrV1:          {modelsync.AssetOcrDeleteV1, modelsync.AssetOcrV1},
	modelsync.ReqPartnerAssetExifsV1: {modelsync.PartnerAssetExifBackfillV1, modelsync.PartnerAssetExifV1},
	modelsync.ReqMemoriesV1:          {modelsync.MemoryDeleteV1, modelsync.MemoryV1},
	modelsync.ReqMemoriesV2:          {modelsync.MemoryDeleteV1, modelsync.MemoryV2},
	modelsync.ReqMemoryToAssetsV1:    {modelsync.MemoryToAssetDeleteV1, modelsync.MemoryToAssetV1},
	modelsync.ReqMemoryToAssetsV2:    {modelsync.MemoryToAssetDeleteV1, modelsync.MemoryToAssetV2},
	modelsync.ReqPeopleV1:            {modelsync.PersonDeleteV1, modelsync.PersonV1},
	modelsync.ReqAssetFacesV2:        {modelsync.AssetFaceDeleteV1, modelsync.AssetFaceV2},
	modelsync.ReqAssetFacesV3:        {modelsync.AssetFaceDeleteV1, modelsync.AssetFaceV3},
	modelsync.ReqUserMetadataV1:      {modelsync.UserMetadataDeleteV1, modelsync.UserMetadataV1},
	modelsync.ReqAssetMetadataV1:     {modelsync.AssetMetadataDeleteV1, modelsync.AssetMetadataV1},
	modelsync.ReqAssetEditsV1:        {modelsync.AssetEditDeleteV1, modelsync.AssetEditV1},
}

var backfillTypes = map[modelsync.EntityType]bool{
	modelsync.PartnerAssetBackfillV2:     true,
	modelsync.PartnerAssetExifBackfillV1: true,
	modelsync.PartnerStackBackfillV1:     true,
	modelsync.AlbumAssetBackfillV2:       true,
	modelsync.AlbumAssetExifBackfillV1:   true,
	modelsync.AlbumUserBackfillV1:        true,
	modelsync.AlbumToAssetBackfillV1:     true,
}

func now() time.Time {
	return time.Now().UTC()
}

func uuidV7At(t time.Time) uuid.UUID {
	var u uuid.UUID
	_, _ = rand.Read(u[:])
	ms := uint64(t.UnixMilli())
	for i := range 6 {
		u[i] = byte(ms >> (40 - 8*i))
	}
	u[6] = u[6]&0x0f | 0x70
	u[8] = u[8]&0x3f | 0x80

	return u
}

func ackMillis(id string) (int64, bool) {
	digits := strings.ReplaceAll(id, "-", "")
	if len(digits) > 12 {
		digits = digits[:12]
	}
	digits = strings.TrimLeft(digits, " \t\n\v\f\r")
	if strings.HasPrefix(digits, "0x") || strings.HasPrefix(digits, "0X") {
		digits = digits[2:]
	}
	end := 0
	for end < len(digits) && strings.ContainsRune("0123456789abcdefABCDEF", rune(digits[end])) {
		end++
	}
	if end == 0 {
		return 0, false
	}
	ms, err := strconv.ParseInt(digits[:end], 16, 64)

	return ms, err == nil
}

func requireSession(a *auth.Auth) error {
	if a.Session == nil {
		return errs.ErrSyncAPIKey
	}

	return nil
}

func Prepare(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *StreamInput) (*Plan, error) {
	if err := requireSession(a); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	sessionID := a.Session.ID
	if in.Reset != nil && *in.Reset {
		if err := repos.Session.ResetSync(ctx, sessionID, now()); err != nil {
			return nil, err
		}
	}

	session, err := repos.Session.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.IsPendingSyncReset {
		return &Plan{reset: true}, nil
	}

	stored, err := repos.SyncCheckpoint.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	acks := make(map[modelsync.EntityType]modelsync.Ack, len(stored))
	for _, c := range stored {
		acks[modelsync.EntityType(c.Type)] = modelsync.ParseAck(c.Ack)
	}

	if complete, ok := acks[modelsync.SyncCompleteV1]; ok {
		if complete.UpdateID == nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCheckpoint, modelsync.SyncCompleteV1)
		}
		if ms, ok := ackMillis(*complete.UpdateID); ok && time.UnixMilli(ms).Before(time.Now().AddDate(0, 0, -maxDays)) {
			return &Plan{reset: true}, nil
		}
	}

	for _, t := range modelsync.Order {
		replacement, deprecated := modelsync.Deprecated[t]
		if deprecated && slices.Contains(in.Types, t) {
			return nil, fmt.Errorf("%w%s is deprecated, use SyncRequestType.%s instead", errs.ErrSyncTypeDeprecated, t, replacement)
		}
	}

	plan := &Plan{
		nowID:     uuidV7At(time.Now().Add(-time.Millisecond)).String(),
		userID:    a.User.ID,
		sessionID: sessionID,
		acks:      make(map[modelsync.EntityType]modelsync.Ack),
	}
	for _, t := range modelsync.Order {
		if !slices.Contains(in.Types, t) {
			continue
		}
		plan.types = append(plan.types, t)
		for _, e := range checkpointTypes[t] {
			ack, ok := acks[e]
			if !ok {
				continue
			}
			checked, err := checkAck(e, ack)
			if err != nil {
				return nil, err
			}
			plan.acks[e] = checked
		}
	}

	return plan, nil
}

func canonical(id *string, optional bool, extra ...string) (*string, bool) {
	if id == nil {
		return nil, optional
	}
	if slices.Contains(extra, *id) {
		return id, true
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return nil, false
	}
	s := parsed.String()

	return &s, true
}

func checkAck(t modelsync.EntityType, ack modelsync.Ack) (modelsync.Ack, error) {
	backfill := backfillTypes[t]
	updateID, ok := canonical(ack.UpdateID, backfill)
	if !ok {
		return ack, fmt.Errorf("%w: %s", ErrInvalidCheckpoint, t)
	}
	extraID := ack.ExtraID
	if backfill {
		if extraID, ok = canonical(ack.ExtraID, true, modelsync.CompleteID); !ok {
			return ack, fmt.Errorf("%w: %s", ErrInvalidCheckpoint, t)
		}
	}

	return modelsync.Ack{Type: ack.Type, UpdateID: updateID, ExtraID: extraID}, nil
}

func (p *Plan) after(t modelsync.EntityType) *string {
	if ack, ok := p.acks[t]; ok {
		return ack.UpdateID
	}

	return nil
}

func page[T any](t modelsync.EntityType, p *Plan, fetch func(b reposync.Bounds) ([]T, error), key func(T) string,
	data func(T) any, emit func(Line) error) error {
	b := reposync.Bounds{After: p.after(t), Before: p.nowID, Limit: pageSize}
	for {
		rows, err := fetch(b)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err = emit(Line{Type: t, Data: data(row), Ack: modelsync.FormatAck(t, key(row))}); err != nil {
				return err
			}
		}
		if len(rows) < pageSize {
			return nil
		}
		last := key(rows[len(rows)-1])
		b.After = &last
	}
}

func identity[T any](row T) any {
	return row
}

func firstOnly(emit func(Line) error, first Line) func(Line) error {
	sent := false

	return func(l Line) error {
		if !sent {
			sent = true
			if err := emit(first); err != nil {
				return err
			}
		}

		return emit(l)
	}
}

type backfill[T any] struct {
	kind    modelsync.EntityType
	bound   modelsync.EntityType
	members func(after *string) ([]reposync.Membership, error)
	rows    func(m reposync.Membership, b reposync.Backfill) ([]T, error)
	key     func(T) string
}

func (bf backfill[T]) run(ctx context.Context, rt *runtime.Runtime, p *Plan, emit func(Line) error) error {
	checkpoint, hasCheckpoint := p.acks[bf.kind]
	members, err := bf.members(checkpoint.UpdateID)
	if err != nil {
		return err
	}

	bound, hasBound := p.acks[bf.bound]
	if !hasBound {
		if len(members) == 0 {
			return nil
		}
		last := members[len(members)-1].CreateID
		return gh.Repositories(rt).SyncCheckpoint.Upsert(ctx, p.sessionID, []synccheckpoint.Checkpoint{{
			Type: string(bf.kind), Ack: modelsync.FormatAck(bf.kind, last, modelsync.CompleteID),
		}}, now())
	}

	for _, m := range members {
		named := hasCheckpoint && checkpoint.UpdateID != nil && *checkpoint.UpdateID == m.CreateID
		var start *string
		if named {
			if checkpoint.ExtraID != nil && *checkpoint.ExtraID == modelsync.CompleteID {
				continue
			}
			start = checkpoint.ExtraID
		}

		b := reposync.Backfill{After: start, End: *bound.UpdateID, Before: p.nowID, Limit: pageSize}
		for {
			rows, err := bf.rows(m, b)
			if err != nil {
				return err
			}
			for _, row := range rows {
				if err = emit(Line{Type: bf.kind, Data: row, Ack: modelsync.FormatAck(bf.kind, m.CreateID, bf.key(row))}); err != nil {
					return err
				}
			}
			if len(rows) < pageSize {
				break
			}
			last := bf.key(rows[len(rows)-1])
			b.After = &last
		}

		if err = emit(Line{Type: modelsync.SyncAckV1, Ack: modelsync.FormatAck(bf.kind, m.CreateID, modelsync.CompleteID)}); err != nil {
			return err
		}
	}

	return nil
}

func (p *Plan) Stream(ctx context.Context, rt *runtime.Runtime, emit func(Line) error) error {
	if p.reset {
		return emit(Line{Type: modelsync.SyncResetV1, Ack: modelsync.FormatAck(modelsync.SyncResetV1, modelsync.ResetID)})
	}

	for _, t := range p.types {
		if handler, ok := handlers[t]; ok {
			if err := handler(ctx, rt, gh.Repositories(rt).Sync, p, emit); err != nil {
				return err
			}
		}
	}

	return emit(Line{Type: modelsync.SyncCompleteV1, Ack: modelsync.FormatAck(modelsync.SyncCompleteV1, p.nowID)})
}

func CleanupAudit(ctx context.Context, rt *runtime.Runtime) error {
	n, err := gh.Repositories(rt).Sync.CleanupAudit(ctx, now().AddDate(0, 0, -(maxDays+1)))
	if err != nil {
		return err
	}
	if n > 0 {
		rt.Info("sync", "audit cleanup", "deleted rows", n)
	}

	return nil
}
