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
	reset       bool
	nowID       string
	userID      uuid.UUID
	types       []modelsync.RequestType
	checkpoints map[modelsync.EntityType]string
}

var checkpointTypes = map[modelsync.RequestType][]modelsync.EntityType{
	modelsync.ReqAuthUsersV1:         {modelsync.AuthUserV1},
	modelsync.ReqAuthUsersV2:         {modelsync.AuthUserV2},
	modelsync.ReqUsersV1:             {modelsync.UserDeleteV1, modelsync.UserV1},
	modelsync.ReqPartnersV1:          {modelsync.PartnerDeleteV1, modelsync.PartnerV1},
	modelsync.ReqAssetsV2:            {modelsync.AssetDeleteV1, modelsync.AssetV2},
	modelsync.ReqStacksV1:            {modelsync.StackDeleteV1, modelsync.StackV1},
	modelsync.ReqPartnerAssetsV2:     {modelsync.PartnerAssetDeleteV1, modelsync.PartnerAssetV2},
	modelsync.ReqPartnerStacksV1:     {modelsync.PartnerStackDeleteV1, modelsync.PartnerStackV1},
	modelsync.ReqAlbumsV1:            {modelsync.AlbumDeleteV1, modelsync.AlbumV1},
	modelsync.ReqAlbumsV2:            {modelsync.AlbumDeleteV1, modelsync.AlbumV2},
	modelsync.ReqAlbumUsersV1:        {modelsync.AlbumUserDeleteV1, modelsync.AlbumUserV1},
	modelsync.ReqAlbumToAssetsV1:     {modelsync.AlbumToAssetDeleteV1, modelsync.AlbumToAssetV1},
	modelsync.ReqAssetExifsV1:        {modelsync.AssetExifV1},
	modelsync.ReqAssetOcrV1:          {modelsync.AssetOcrDeleteV1, modelsync.AssetOcrV1},
	modelsync.ReqPartnerAssetExifsV1: {modelsync.PartnerAssetExifV1},
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
		nowID:       uuidV7At(time.Now().Add(-time.Millisecond)).String(),
		userID:      a.User.ID,
		checkpoints: make(map[modelsync.EntityType]string),
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
			if ack.UpdateID == nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidCheckpoint, e)
			}
			id, err := uuid.Parse(*ack.UpdateID)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidCheckpoint, e)
			}
			plan.checkpoints[e] = id.String()
		}
	}

	return plan, nil
}

func (p *Plan) after(t modelsync.EntityType) *string {
	if id, ok := p.checkpoints[t]; ok {
		return &id
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

func (p *Plan) Stream(ctx context.Context, rt *runtime.Runtime, emit func(Line) error) error {
	if p.reset {
		return emit(Line{Type: modelsync.SyncResetV1, Ack: modelsync.FormatAck(modelsync.SyncResetV1, modelsync.ResetID)})
	}

	for _, t := range p.types {
		if handler, ok := handlers[t]; ok {
			if err := handler(ctx, gh.Repositories(rt).Sync, p, emit); err != nil {
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
