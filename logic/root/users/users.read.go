package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]user.User, error) {
	cfg, err := gh.ServerConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	if a.User.IsAdmin || cfg.PublicUsers {
		return repos.User.List(ctx, uuid.NullUUID{}, false)
	}

	u, err := repos.User.Get(ctx, a.User.ID, false)
	if errors.Is(err, errs.ErrNoRows) {
		return []user.User{}, nil
	}
	if err != nil {
		return nil, err
	}

	return []user.User{*u}, nil
}

func Get(ctx context.Context, rt *runtime.Runtime, id uuid.UUID, withDeleted bool) (*user.User, error) {
	u, err := gh.Repositories(rt).User.Get(ctx, id, withDeleted)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrUserNotFound
	}

	return u, err
}
