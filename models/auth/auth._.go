package auth

import (
	"tty.fail/mrus/korda/models/apikey"
	"tty.fail/mrus/korda/models/session"
	"tty.fail/mrus/korda/models/user"
)

type Auth struct {
	User     *user.User
	Session  *session.Session
	Elevated bool
	APIKey   *apikey.APIKey
}

func (a *Auth) SessionID() string {
	if a == nil || a.Session == nil {
		return ""
	}

	return a.Session.ID.String()
}
