package auth

import (
	"context"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type SignUpInput struct {
	Email    string `json:"email" v:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

const adminStorageLabel = "admin"

func RequireSetupAvailable(ctx context.Context, rt *runtime.Runtime) error {
	hasAdmin, err := gh.Repositories(rt).User.HasAdmin(ctx)
	if err != nil {
		return err
	}
	if hasAdmin {
		return errs.ErrAdminSetupNotAvailable
	}

	return nil
}

func SignUpAdmin(ctx context.Context, rt *runtime.Runtime, in *SignUpInput) (*user.User, error) {
	return users.Create(ctx, rt, users.CreateInput{
		Email:        in.Email,
		Password:     in.Password,
		Name:         in.Name,
		IsAdmin:      true,
		StorageLabel: optional.Of(adminStorageLabel),
	})
}
