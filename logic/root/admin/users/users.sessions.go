package users

import (
	"context"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/preferences"
	"tty.fail/mrus/korda/models/session"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Sessions(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) ([]session.Session, error) {
	return gh.Repositories(rt).Session.ListByUser(ctx, id, now())
}

func Preferences(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (preferences.Preferences, error) {
	if _, err := logicusers.Get(ctx, rt, id, true); err != nil {
		return preferences.Preferences{}, err
	}

	return logicusers.Preferences(ctx, rt, id)
}

func UpdatePreferences(ctx context.Context, rt *runtime.Runtime, id uuid.UUID, in *preferences.Update) (preferences.Preferences, error) {
	if _, err := logicusers.Get(ctx, rt, id, false); err != nil {
		return preferences.Preferences{}, err
	}

	return logicusers.UpdatePreferences(ctx, rt, id, in)
}
