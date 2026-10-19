package auth

import (
	"context"
	"database/sql"
	"time"

	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Status struct {
	PinCode      bool
	Password     bool
	IsElevated   bool
	ExpiresAt    sql.Null[time.Time]
	PinExpiresAt sql.Null[time.Time]
}

func GetStatus(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*Status, error) {
	u, err := currentUser(ctx, rt, a)
	if err != nil {
		return nil, err
	}

	status := &Status{
		PinCode:    u.HasPinCode(),
		Password:   u.HasPassword(),
		IsElevated: a.Session != nil && a.Elevated,
	}

	if a.Session != nil {
		s, err := gh.Repositories(rt).Session.Get(ctx, a.Session.ID)
		if err != nil {
			return nil, err
		}
		status.ExpiresAt = s.ExpiresAt
		status.PinExpiresAt = s.PinExpiresAt
	}

	return status, nil
}
