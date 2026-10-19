package auth

import (
	"context"
	"database/sql"
	"time"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/helpers/password"
	"xn--gckvb8fzb.com/glides/runtime"
)

type PinCodeSetupInput struct {
	PinCode string `json:"pinCode" v:"regex=pin"`
}

type PinCodeResetInput struct {
	PinCode  *string `json:"pinCode" v:"regex=pin"`
	Password *string `json:"password"`
}

type PinCodeChangeInput struct {
	PinCode    *string `json:"pinCode" v:"regex=pin"`
	Password   *string `json:"password"`
	NewPinCode string  `json:"newPinCode" v:"regex=pin"`
}

type SessionUnlockInput = PinCodeResetInput

func validatePinCode(u *user.User, pinCode *string, plain *string) error {
	if !u.HasPinCode() {
		return errs.ErrPinCodeMissing
	}

	switch {
	case plain != nil && *plain != "":
		if !u.HasPassword() || !checkSecret(u.Password.V, *plain) {
			return errs.ErrWrongPassword
		}
	case pinCode != nil && *pinCode != "":
		if !checkSecret(u.PinCode.V, *pinCode) {
			return errs.ErrWrongPinCode
		}
	default:
		return errs.ErrPasswordOrPinRequired
	}

	return nil
}

func SetupPinCode(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *PinCodeSetupInput) error {
	u, err := currentUser(ctx, rt, a)
	if err != nil {
		return err
	}
	if u.HasPinCode() {
		return errs.ErrPinCodeExists
	}

	hash, err := password.Hash(in.PinCode)
	if err != nil {
		return err
	}

	_, err = gh.Repositories(rt).User.Update(ctx, u.ID, repouser.Update{PinCode: optional.Of(hash)}, now())
	return err
}

func ResetPinCode(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *PinCodeResetInput) error {
	repos := gh.Repositories(rt)

	u, err := currentUser(ctx, rt, a)
	if err != nil {
		return err
	}
	if err = validatePinCode(u, in.PinCode, in.Password); err != nil {
		return err
	}

	at := now()
	if _, err = repos.User.Update(ctx, u.ID, repouser.Update{PinCode: optional.Value[string]{Set: true, Null: true}}, at); err != nil {
		return err
	}

	return repos.Session.LockAll(ctx, u.ID, at)
}

func ChangePinCode(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *PinCodeChangeInput) error {
	u, err := currentUser(ctx, rt, a)
	if err != nil {
		return err
	}
	if err = validatePinCode(u, in.PinCode, in.Password); err != nil {
		return err
	}

	hash, err := password.Hash(in.NewPinCode)
	if err != nil {
		return err
	}

	_, err = gh.Repositories(rt).User.Update(ctx, u.ID, repouser.Update{PinCode: optional.Of(hash)}, now())
	return err
}

func UnlockSession(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SessionUnlockInput) error {
	if a.Session == nil {
		return errs.ErrSessionTokenRequired
	}

	u, err := currentUser(ctx, rt, a)
	if err != nil {
		return err
	}
	if err = validatePinCode(u, in.PinCode, nil); err != nil {
		return err
	}

	at := now()
	return gh.Repositories(rt).Session.SetPinExpiresAt(ctx, a.Session.ID,
		sql.Null[time.Time]{V: at.Add(pinUnlockFor), Valid: true}, at)
}

func LockSession(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) error {
	if a.Session == nil {
		return errs.ErrSessionTokenRequired
	}

	return gh.Repositories(rt).Session.SetPinExpiresAt(ctx, a.Session.ID, sql.Null[time.Time]{}, now())
}
