package auth

import (
	"time"

	"tty.fail/mrus/korda/models/permission"
)

const (
	LoginURL          = "/auth/login?autoLaunch=0"
	sessionTouchAfter = time.Hour
	pinUnlockFor      = 15 * time.Minute
	pinExtendWithin   = 5 * time.Minute
)

type Credentials struct {
	ShareKey     string
	ShareSlug    string
	SessionToken string
	APIKey       string
	UserAgent    string
	DeviceModel  string
	DeviceType   string
}

type Requirement struct {
	Permission   permission.Permission
	NoPermission bool
	Admin        bool
}

func (r Requirement) permission() permission.Permission {
	if r.Permission == "" {
		return permission.All
	}

	return r.Permission
}

func now() time.Time {
	return time.Now().UTC()
}
