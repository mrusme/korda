package auth

import (
	"context"
	"errors"
	"fmt"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/secret"
	"tty.fail/mrus/korda/helpers/useragent"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Authenticate(
	ctx context.Context,
	rt *runtime.Runtime,
	creds Credentials,
	req Requirement,
	uri string,
) (a *auth.Auth, err error) {
	if a, err = validate(ctx, rt, creds); err != nil {
		return nil, err
	}

	if req.Admin && !a.User.IsAdmin {
		rt.Warn("auth", "denied", "reason", "admin only route", "uri", uri)
		return nil, errs.ErrForbidden
	}

	if a.APIKey != nil && !req.NoPermission &&
		!permission.IsGranted([]permission.Permission{req.permission()}, a.APIKey.Permissions) {
		return nil, fmt.Errorf("%w: %s", errs.ErrMissingPermission, req.permission())
	}

	return a, nil
}

func validate(ctx context.Context, rt *runtime.Runtime, creds Credentials) (*auth.Auth, error) {
	switch {
	case creds.ShareKey != "":
		return nil, errs.ErrInvalidShareKey
	case creds.ShareSlug != "":
		return nil, errs.ErrInvalidShareSlug
	case creds.SessionToken != "":
		return validateSession(ctx, rt, creds)
	case creds.APIKey != "":
		return validateAPIKey(ctx, rt, creds.APIKey)
	}

	return nil, errs.ErrAuthenticationRequired
}

func validateSession(ctx context.Context, rt *runtime.Runtime, creds Credentials) (*auth.Auth, error) {
	repos := gh.Repositories(rt)
	at := now()

	s, err := repos.Session.GetByToken(ctx, secret.Hash(creds.SessionToken), at)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrInvalidUserToken
	}
	if err != nil {
		return nil, err
	}

	u, err := repos.User.Get(ctx, s.UserID, false)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrInvalidUserToken
	}
	if err != nil {
		return nil, err
	}

	details := useragent.Parse(creds.UserAgent, creds.DeviceModel, creds.DeviceType)
	if at.Sub(s.UpdatedAt) > sessionTouchAfter || details.AppVersion != s.AppVersion {
		if err = repos.Session.Touch(ctx, s.ID, details.AppVersion, details.DeviceOS, details.DeviceType, at); err != nil {
			return nil, err
		}
		s.AppVersion, s.DeviceOS, s.DeviceType, s.UpdatedAt = details.AppVersion, details.DeviceOS, details.DeviceType, at
	}

	elevated := false
	if s.PinExpiresAt.Valid {
		elevated = s.PinExpiresAt.V.After(at)
		if elevated && at.Add(pinExtendWithin).After(s.PinExpiresAt.V) {
			s.PinExpiresAt.V = at.Add(pinExtendWithin)
			if err = repos.Session.SetPinExpiresAt(ctx, s.ID, s.PinExpiresAt, at); err != nil {
				return nil, err
			}
		}
	}

	return &auth.Auth{User: u, Session: s, Elevated: elevated}, nil
}

func validateAPIKey(ctx context.Context, rt *runtime.Runtime, key string) (*auth.Auth, error) {
	repos := gh.Repositories(rt)

	k, err := repos.APIKey.GetByKey(ctx, secret.Hash(key))
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrInvalidAPIKey
	}
	if err != nil {
		return nil, err
	}

	u, err := repos.User.Get(ctx, k.UserID, false)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrInvalidAPIKey
	}
	if err != nil {
		return nil, err
	}

	return &auth.Auth{User: u, APIKey: k}, nil
}
