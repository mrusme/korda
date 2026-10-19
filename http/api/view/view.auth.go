package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
)

type Login struct {
	AccessToken          string `json:"accessToken"`
	UserID               string `json:"userId"`
	UserEmail            string `json:"userEmail"`
	Name                 string `json:"name"`
	IsAdmin              bool   `json:"isAdmin"`
	ProfileImagePath     string `json:"profileImagePath"`
	ShouldChangePassword bool   `json:"shouldChangePassword"`
	IsOnboarded          bool   `json:"isOnboarded"`
}

type Logout struct {
	Successful  bool   `json:"successful"`
	RedirectURI string `json:"redirectUri"`
}

type AuthStatus struct {
	PinCode      bool           `json:"pinCode"`
	Password     bool           `json:"password"`
	IsElevated   bool           `json:"isElevated"`
	ExpiresAt    *jsontime.Time `json:"expiresAt,omitempty"`
	PinExpiresAt *jsontime.Time `json:"pinExpiresAt,omitempty"`
}

type ValidateAccessToken struct {
	AuthStatus bool `json:"authStatus"`
}

func NewLogin(result *logicauth.LoginResult) Login {
	return Login{
		AccessToken:          result.Token,
		UserID:               id(result.User.ID),
		UserEmail:            result.User.Email,
		Name:                 result.User.Name,
		IsAdmin:              result.User.IsAdmin,
		ProfileImagePath:     result.User.ProfileImagePath,
		ShouldChangePassword: result.User.ShouldChangePassword,
		IsOnboarded:          result.IsOnboarded,
	}
}

func NewLogout(result *logicauth.LogoutResult) Logout {
	return Logout{Successful: result.Successful, RedirectURI: result.RedirectURI}
}

func NewAuthStatus(status *logicauth.Status) AuthStatus {
	return AuthStatus{
		PinCode:      status.PinCode,
		Password:     status.Password,
		IsElevated:   status.IsElevated,
		ExpiresAt:    jsontime.Nullable(status.ExpiresAt),
		PinExpiresAt: jsontime.Nullable(status.PinExpiresAt),
	}
}
