package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/user"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/helpers/password"
	"xn--gckvb8fzb.com/glides/runtime"
)

type ChangePasswordInput struct {
	Password           string `json:"password"`
	NewPassword        string `json:"newPassword" v:"min=8"`
	InvalidateSessions *bool  `json:"invalidateSessions"`
}

func currentUser(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*user.User, error) {
	u, err := gh.Repositories(rt).User.Get(ctx, a.User.ID, false)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrUnauthorized
	}

	return u, err
}

func ChangePassword(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *ChangePasswordInput) (*user.User, error) {
	repos := gh.Repositories(rt)

	u, err := currentUser(ctx, rt, a)
	if err != nil {
		return nil, err
	}
	if !u.HasPassword() || !checkSecret(u.Password.V, in.Password) {
		return nil, errs.ErrWrongPassword
	}

	hash, err := password.Hash(in.NewPassword)
	if err != nil {
		return nil, err
	}

	updated, err := repos.User.Update(ctx, u.ID, repouser.Update{
		Password:             optional.Of(hash),
		ShouldChangePassword: optional.Of(false),
	}, now())
	if err != nil {
		return nil, err
	}

	var except uuid.NullUUID
	if a.Session != nil {
		except = uuid.NullUUID{UUID: a.Session.ID, Valid: true}
	}
	ids, err := repos.Session.DeleteAllOfUser(ctx, u.ID, except)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		gh.Emit(ctx, rt, event.SessionsRevoked{SessionIDs: ids})
	}

	return updated, nil
}
