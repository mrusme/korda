package auth

import (
	"context"

	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"xn--gckvb8fzb.com/glides/runtime"
)

type LogoutResult struct {
	Successful  bool
	RedirectURI string
}

func Logout(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*LogoutResult, error) {
	if a.Session != nil {
		if err := gh.Repositories(rt).Session.Delete(ctx, a.Session.ID); err != nil {
			return nil, err
		}
		gh.Emit(ctx, rt, event.SessionDeleted{SessionID: a.Session.ID, At: now()})
	}

	return &LogoutResult{Successful: true, RedirectURI: LoginURL}, nil
}
