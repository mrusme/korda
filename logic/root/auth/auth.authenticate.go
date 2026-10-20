package auth

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/secret"
	"tty.fail/mrus/korda/helpers/useragent"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/sharedlink"
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

	if a.SharedLink != nil && !req.SharedLink {
		rt.Warn("auth", "denied", "reason", "non-shared route", "uri", uri)
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
		return validateSharedLink(ctx, rt, creds.ShareKey, true)
	case creds.ShareSlug != "":
		return validateSharedLink(ctx, rt, creds.ShareSlug, false)
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

func DecodeShareKey(key string) ([]byte, bool) {
	if len(key) == 100 {
		decoded, err := hex.DecodeString(key)
		return decoded, err == nil
	}

	var b strings.Builder
	for _, r := range key {
		if r == '=' {
			break
		}
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == '+':
			b.WriteRune('-')
		case r == '/':
			b.WriteRune('_')
		}
	}
	cleaned := b.String()
	if len(cleaned)%4 == 1 {
		cleaned = cleaned[:len(cleaned)-1]
	}

	decoded, err := base64.RawURLEncoding.DecodeString(cleaned)
	return decoded, err == nil
}

func validateSharedLink(ctx context.Context, rt *runtime.Runtime, value string, byKey bool) (*auth.Auth, error) {
	repos := gh.Repositories(rt)
	invalid := errs.ErrInvalidShareSlug
	var (
		link *sharedlink.SharedLink
		err  error
	)
	if byKey {
		invalid = errs.ErrInvalidShareKey
		key, ok := DecodeShareKey(value)
		if !ok {
			return nil, invalid
		}
		link, err = repos.SharedLink.ByKey(ctx, key)
	} else {
		link, err = repos.SharedLink.BySlug(ctx, value)
	}
	if errors.Is(err, errs.ErrNoRows) {
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}
	if link.ExpiresAt.Valid && !link.ExpiresAt.V.After(now()) {
		return nil, invalid
	}

	u, err := repos.User.Get(ctx, link.UserID, true)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}

	return &auth.Auth{User: u, SharedLink: link}, nil
}
