package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/secret"
	"tty.fail/mrus/korda/helpers/useragent"
	"tty.fail/mrus/korda/models/session"
	"tty.fail/mrus/korda/models/user"
	"tty.fail/mrus/korda/models/usermetadata"
	"xn--gckvb8fzb.com/glides/helpers/password"
	"xn--gckvb8fzb.com/glides/runtime"
)

type LoginInput struct {
	Email    string `json:"email" v:"email"`
	Password string `json:"password"`
}

type LoginDetails struct {
	ClientIP    string
	UserAgent   string
	DeviceModel string
	DeviceType  string
}

type LoginResult struct {
	Token       string
	User        *user.User
	IsOnboarded bool
}

var (
	decoyOnce sync.Once
	decoyHash string
)

func decoy() string {
	decoyOnce.Do(func() {
		decoyHash, _ = password.HashRandom()
	})

	return decoyHash
}

func checkSecret(encoded string, plain string) bool {
	match, _, err := password.Check(encoded, plain)
	return err == nil && match
}

func Login(ctx context.Context, rt *runtime.Runtime, in *LoginInput, details LoginDetails) (*LoginResult, error) {
	repos := gh.Repositories(rt)

	u, err := repos.User.GetByEmail(ctx, in.Email)
	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		return nil, err
	}

	stored := decoy()
	if u != nil && u.HasPassword() {
		stored = u.Password.V
	}
	authenticated := checkSecret(stored, in.Password)

	if u == nil || !u.HasPassword() || !authenticated {
		rt.Warn("auth", "failed login attempt", "email", in.Email, "ip", details.ClientIP)
		return nil, errs.ErrIncorrectCredentials
	}

	token, err := createSession(ctx, rt, u, details)
	if err != nil {
		return nil, err
	}

	onboarded, err := IsOnboarded(ctx, rt, u)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, User: u, IsOnboarded: onboarded}, nil
}

func createSession(ctx context.Context, rt *runtime.Runtime, u *user.User, details LoginDetails) (string, error) {
	agent := useragent.Parse(details.UserAgent, details.DeviceModel, details.DeviceType)
	token := secret.Token()
	at := now()

	_, err := gh.Repositories(rt).Session.Create(ctx, &session.Session{
		ID:         uuid.New(),
		Token:      secret.Hash(token),
		UserID:     u.ID,
		DeviceType: agent.DeviceType,
		DeviceOS:   agent.DeviceOS,
		AppVersion: agent.AppVersion,
		CreatedAt:  at,
		UpdatedAt:  at,
		UpdateID:   uuid.Must(uuid.NewV7()),
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

func IsOnboarded(ctx context.Context, rt *runtime.Runtime, u *user.User) (bool, error) {
	item, err := gh.Repositories(rt).UserMetadata.Get(ctx, u.ID, usermetadata.Onboarding)
	if errors.Is(err, errs.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var value usermetadata.OnboardingValue
	if err = json.Unmarshal([]byte(item.Value), &value); err != nil {
		return false, err
	}

	return value.IsOnboarded, nil
}
