package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/preferences"
	"tty.fail/mrus/korda/models/usermetadata"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Preferences(ctx context.Context, rt *runtime.Runtime, userID uuid.UUID) (preferences.Preferences, error) {
	item, err := gh.Repositories(rt).UserMetadata.Get(ctx, userID, usermetadata.Preferences)
	if errors.Is(err, errs.ErrNoRows) {
		return preferences.Defaults(), nil
	}
	if err != nil {
		return preferences.Preferences{}, err
	}

	return preferences.FromStored(item.Value)
}

func UpdatePreferences(ctx context.Context, rt *runtime.Runtime, userID uuid.UUID, in *preferences.Update) (preferences.Preferences, error) {
	current, err := Preferences(ctx, rt, userID)
	if err != nil {
		return current, err
	}

	merged := preferences.Merge(current, *in)

	stored, err := preferences.Stored(merged)
	if err != nil {
		return merged, err
	}

	if err = gh.Repositories(rt).UserMetadata.Upsert(ctx, userID, usermetadata.Preferences, stored, now()); err != nil {
		return merged, err
	}

	return merged, nil
}
