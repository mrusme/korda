package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/helpers/password"
	"xn--gckvb8fzb.com/glides/runtime"
)

type CreateInput struct {
	Email                string
	Password             string
	Name                 string
	IsAdmin              bool
	AvatarColor          optional.Value[user.AvatarColor]
	PinCode              optional.Value[string]
	StorageLabel         optional.Value[string]
	QuotaSizeInBytes     optional.Value[int64]
	ShouldChangePassword optional.Value[bool]
}

func Create(ctx context.Context, rt *runtime.Runtime, in CreateInput) (*user.User, error) {
	repos := gh.Repositories(rt)

	if _, err := repos.User.GetByEmail(ctx, in.Email); err == nil {
		rt.Debug("users", "create", "rejected", "email exists")
		return nil, errs.ErrEmailNotAvailable
	} else if !errors.Is(err, errs.ErrNoRows) {
		return nil, err
	}

	if !in.IsAdmin {
		if _, err := repos.User.GetAdmin(ctx); errors.Is(err, errs.ErrNoRows) {
			return nil, errs.ErrFirstUserMustBeAdmin
		} else if err != nil {
			return nil, err
		}
	}

	at := now()
	u := &user.User{
		ID:                   uuid.New(),
		Email:                in.Email,
		Name:                 in.Name,
		IsAdmin:              in.IsAdmin,
		ShouldChangePassword: true,
		ProfileChangedAt:     at,
		Status:               user.StatusActive,
		ClusterGroupID:       uuid.New(),
		CreatedAt:            at,
		UpdatedAt:            at,
		UpdateID:             uuid.Must(uuid.NewV7()),
	}

	if in.Password != "" {
		hash, err := password.Hash(in.Password)
		if err != nil {
			return nil, err
		}
		u.Password = sql.Null[string]{V: hash, Valid: true}
	}
	if in.PinCode.Valid() {
		hash, err := password.Hash(in.PinCode.V)
		if err != nil {
			return nil, err
		}
		u.PinCode = sql.Null[string]{V: hash, Valid: true}
	}
	if in.AvatarColor.Valid() {
		u.AvatarColor = sql.Null[user.AvatarColor]{V: in.AvatarColor.V, Valid: true}
	}
	if in.StorageLabel.Valid() && in.StorageLabel.V != "" {
		u.StorageLabel = sql.Null[string]{V: in.StorageLabel.V, Valid: true}
	}
	if in.QuotaSizeInBytes.Valid() {
		u.QuotaSizeInBytes = sql.Null[int64]{V: in.QuotaSizeInBytes.V, Valid: true}
	}
	if in.ShouldChangePassword.Valid() {
		u.ShouldChangePassword = in.ShouldChangePassword.V
	}

	created, err := repos.User.Create(ctx, u)
	if err != nil {
		return nil, uniqueError(err)
	}

	return created, nil
}

func uniqueError(err error) error {
	if !errors.Is(err, errs.ErrUniqueViolationOn) {
		return err
	}

	switch {
	case strings.Contains(err.Error(), "users.email"):
		return errs.ErrEmailNotAvailable
	case strings.Contains(err.Error(), "users.storage_label"):
		return errs.ErrStorageLabelInUse
	}

	return err
}
