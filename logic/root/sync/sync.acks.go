package sync

import (
	"context"
	"errors"
	"fmt"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	modelsync "tty.fail/mrus/korda/models/sync"
	"tty.fail/mrus/korda/models/synccheckpoint"
	"xn--gckvb8fzb.com/glides/runtime"
)

var (
	ErrNoAcks  = errors.New("no acks to store")
	ErrNoTypes = errors.New("no ack types to delete")
)

func Acks(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]synccheckpoint.Checkpoint, error) {
	if err := requireSession(a); err != nil {
		return nil, err
	}

	return gh.Repositories(rt).SyncCheckpoint.List(ctx, a.Session.ID)
}

func SetAcks(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *AckSetInput) error {
	if err := requireSession(a); err != nil {
		return err
	}

	repos := gh.Repositories(rt)
	var order []string
	byType := map[string]string{}
	for _, ack := range in.Acks {
		t := modelsync.ParseAck(ack).Type
		if t == string(modelsync.SyncResetV1) {
			return repos.Session.ResetSync(ctx, a.Session.ID, now())
		}
		if !modelsync.IsEntityType(t) {
			return fmt.Errorf("%w %s", errs.ErrInvalidAckType, t)
		}
		if _, seen := byType[t]; !seen {
			order = append(order, t)
		}
		byType[t] = ack
	}
	if len(order) == 0 {
		return ErrNoAcks
	}

	checkpoints := make([]synccheckpoint.Checkpoint, 0, len(order))
	for _, t := range order {
		checkpoints = append(checkpoints, synccheckpoint.Checkpoint{Type: t, Ack: byType[t]})
	}

	return repos.SyncCheckpoint.Upsert(ctx, a.Session.ID, checkpoints, now())
}

func DeleteAcks(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *AckDeleteInput) error {
	if err := requireSession(a); err != nil {
		return err
	}
	if in.Types != nil && len(in.Types) == 0 {
		return ErrNoTypes
	}

	var types []string
	if in.Types != nil {
		types = make([]string, 0, len(in.Types))
		for _, t := range in.Types {
			types = append(types, string(t))
		}
	}

	return gh.Repositories(rt).SyncCheckpoint.Delete(ctx, a.Session.ID, types)
}
