package users

import (
	"context"
	"errors"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/helpers/password"
	"xn--gckvb8fzb.com/glides/runtime"
)

type UpdateMeInput struct {
	Email       *string                          `json:"email" v:"email"`
	Password    *string                          `json:"password"`
	Name        *string                          `json:"name"`
	AvatarColor optional.Value[user.AvatarColor] `json:"avatarColor" v:"nullable"`
}

func UpdateMe(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UpdateMeInput) (*user.User, error) {
	repos := gh.Repositories(rt)

	if in.Email != nil && *in.Email != "" {
		duplicate, err := repos.User.GetByEmail(ctx, *in.Email)
		if err != nil && !errors.Is(err, errs.ErrNoRows) {
			return nil, err
		}
		if duplicate != nil && duplicate.ID != a.User.ID {
			rt.Warn("users", "update", "rejected", "email in use by another account")
			return nil, errs.ErrEmailNotAvailable
		}
	}

	var update repouser.Update
	if in.Email != nil {
		update.Email = optional.Of(*in.Email)
	}
	if in.Name != nil {
		update.Name = optional.Of(*in.Name)
	}
	update.AvatarColor = in.AvatarColor

	if in.Password != nil && *in.Password != "" {
		hash, err := password.Hash(*in.Password)
		if err != nil {
			return nil, err
		}
		update.Password = optional.Of(hash)
		update.ShouldChangePassword = optional.Of(false)
	}

	updated, err := repos.User.Update(ctx, a.User.ID, update, now())
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrUserNotFound
	}
	if err != nil {
		return nil, uniqueError(err)
	}

	return updated, nil
}
