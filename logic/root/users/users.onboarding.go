package users

import (
	"context"
	"encoding/json"
	"errors"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/usermetadata"
	"xn--gckvb8fzb.com/glides/runtime"
)

type OnboardingInput struct {
	IsOnboarded bool `json:"isOnboarded"`
}

func Onboarding(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (bool, error) {
	item, err := gh.Repositories(rt).UserMetadata.Get(ctx, a.User.ID, usermetadata.Onboarding)
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

func SetOnboarding(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *OnboardingInput) (bool, error) {
	encoded, err := json.Marshal(usermetadata.OnboardingValue{IsOnboarded: in.IsOnboarded})
	if err != nil {
		return false, err
	}

	if err = gh.Repositories(rt).UserMetadata.Upsert(ctx, a.User.ID, usermetadata.Onboarding, string(encoded), now()); err != nil {
		return false, err
	}

	return in.IsOnboarded, nil
}

func DeleteOnboarding(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) error {
	return gh.Repositories(rt).UserMetadata.Delete(ctx, a.User.ID, usermetadata.Onboarding, now())
}
