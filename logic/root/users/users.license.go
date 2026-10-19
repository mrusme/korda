package users

import (
	"context"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/usermetadata"
	"xn--gckvb8fzb.com/glides/runtime"
)

type LicenseInput struct {
	LicenseKey    string `json:"licenseKey" v:"regex=license"`
	ActivationKey string `json:"activationKey"`
}

func License(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) error {
	return errs.ErrNotFound
}

func SetLicense(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *LicenseInput) error {
	return errs.ErrInvalidLicenseKey
}

func DeleteLicense(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) error {
	return gh.Repositories(rt).UserMetadata.Delete(ctx, a.User.ID, usermetadata.License, now())
}
