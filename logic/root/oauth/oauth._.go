package oauth

import (
	"context"
	"strings"
	"time"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

const MobileRedirect = "app.immich:///oauth-callback"

type AuthorizeInput struct {
	RedirectURI   string  `json:"redirectUri"`
	State         *string `json:"state"`
	CodeChallenge *string `json:"codeChallenge"`
}

type CallbackInput struct {
	URL          string  `json:"url" v:"min=1"`
	State        *string `json:"state"`
	CodeVerifier *string `json:"codeVerifier"`
}

type LogoutInput struct {
	LogoutToken string `json:"logout_token"`
}

type DiscoveryError struct{}

func (DiscoveryError) Error() string {
	return "Error in OAuth discovery: TypeError: Invalid URL"
}

func Authorize(in *AuthorizeInput) error {
	return errs.ErrOAuthDisabled
}

func Callback(in *CallbackInput) error {
	return errs.ErrOAuthDisabled
}

func Link(in *CallbackInput, stateCookie string, verifierCookie string) error {
	state, verifier := stateCookie, verifierCookie
	if in.State != nil {
		state = *in.State
	}
	if in.CodeVerifier != nil {
		verifier = *in.CodeVerifier
	}
	if state == "" {
		return errs.ErrOAuthState
	}
	if verifier == "" {
		return errs.ErrOAuthVerifier
	}

	return DiscoveryError{}
}

func Unlink(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*user.User, error) {
	return gh.Repositories(rt).User.UnlinkOAuth(ctx, a.User.ID, now())
}

func UnlinkAll(ctx context.Context, rt *runtime.Runtime) error {
	return gh.Repositories(rt).User.UnlinkAllOAuth(ctx, now())
}

func Logout(in *LogoutInput) error {
	return errs.ErrOAuthLogout
}

func encodeURL(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			b.WriteString(s[i : i+3])
			i += 2
			continue
		}
		if allowed(c) && c != '%' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte("0123456789ABCDEF"[c>>4])
		b.WriteByte("0123456789ABCDEF"[c&0xf])
	}

	return b.String()
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func allowed(c byte) bool {
	switch {
	case c == 0x21, c >= 0x23 && c <= 0x3B, c == 0x3D, c >= 0x3F && c <= 0x5F, c >= 0x61 && c <= 0x7A, c == 0x7C, c == 0x7E:
		return true
	}

	return false
}

func MobileLocation(requestURI string) string {
	query := ""
	if parts := strings.SplitN(requestURI, "?", 3); len(parts) > 1 {
		query = parts[1]
	}

	return encodeURL(MobileRedirect + "?" + query)
}

func now() time.Time {
	return time.Now().UTC()
}
