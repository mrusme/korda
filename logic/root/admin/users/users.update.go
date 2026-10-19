package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/helpers/password"
	"xn--gckvb8fzb.com/glides/runtime"
)

type UpdateInput struct {
	Email                *string                          `json:"email" v:"email"`
	Password             *string                          `json:"password"`
	PinCode              optional.Value[string]           `json:"pinCode" v:"nullable,regex=pin"`
	Name                 *string                          `json:"name"`
	AvatarColor          optional.Value[user.AvatarColor] `json:"avatarColor" v:"nullable"`
	StorageLabel         optional.Value[string]           `json:"storageLabel" v:"nullable,sanitize"`
	ShouldChangePassword *bool                            `json:"shouldChangePassword"`
	QuotaSizeInBytes     optional.Value[int64]            `json:"quotaSizeInBytes" v:"nullable,min=0"`
	IsAdmin              *bool                            `json:"isAdmin"`
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*user.User, error) {
	repos := gh.Repositories(rt)

	if _, err := logicusers.Get(ctx, rt, id, false); err != nil {
		return nil, err
	}

	if in.IsAdmin != nil && *in.IsAdmin != a.User.IsAdmin && a.User.ID == id {
		return nil, errs.ErrAdminStatusSelf
	}

	if in.Email != nil && *in.Email != "" {
		duplicate, err := repos.User.GetByEmail(ctx, *in.Email)
		if err != nil && !errors.Is(err, errs.ErrNoRows) {
			return nil, err
		}
		if duplicate != nil && duplicate.ID != id {
			rt.Debug("admin users", "update", "rejected", "email in use by another account")
			return nil, errs.ErrEmailNotAvailable
		}
	}

	if in.StorageLabel.Valid() && in.StorageLabel.V != "" {
		duplicate, err := repos.User.GetByStorageLabel(ctx, in.StorageLabel.V)
		if err != nil && !errors.Is(err, errs.ErrNoRows) {
			return nil, err
		}
		if duplicate != nil && duplicate.ID != id {
			return nil, errs.ErrStorageLabelInUse
		}
	}

	update := repouser.Update{
		AvatarColor:      in.AvatarColor,
		StorageLabel:     in.StorageLabel,
		QuotaSizeInBytes: in.QuotaSizeInBytes,
	}
	if in.Email != nil {
		update.Email = optional.Of(*in.Email)
	}
	if in.Name != nil {
		update.Name = optional.Of(*in.Name)
	}
	if in.ShouldChangePassword != nil {
		update.ShouldChangePassword = optional.Of(*in.ShouldChangePassword)
	}
	if in.IsAdmin != nil {
		update.IsAdmin = optional.Of(*in.IsAdmin)
	}
	if in.Password != nil && *in.Password != "" {
		hash, err := password.Hash(*in.Password)
		if err != nil {
			return nil, err
		}
		update.Password = optional.Of(hash)
	} else if in.Password != nil {
		update.Password = optional.Of("")
	}
	if in.PinCode.Valid() && in.PinCode.V != "" {
		hash, err := password.Hash(in.PinCode.V)
		if err != nil {
			return nil, err
		}
		update.PinCode = optional.Of(hash)
	} else {
		update.PinCode = in.PinCode
	}
	if update.StorageLabel.Valid() && update.StorageLabel.V == "" {
		update.StorageLabel = optional.Value[string]{Set: true, Null: true}
	}

	updated, err := repos.User.Update(ctx, id, update, now())
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return updated, nil
}
