package apikeys

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/secret"
	"tty.fail/mrus/korda/models/apikey"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/runtime"
)

type CreateInput struct {
	Name        *string                 `json:"name"`
	Permissions []permission.Permission `json:"permissions" v:"min=1"`
}

type UpdateInput struct {
	Name        *string                 `json:"name"`
	Permissions []permission.Permission `json:"permissions" v:"optional,min=1"`
}

func now() time.Time {
	return time.Now().UTC()
}

func notFound(err error) error {
	if errors.Is(err, errs.ErrNoRows) {
		return errs.ErrAPIKeyNotFound
	}

	return err
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*apikey.APIKey, string, error) {
	if a.APIKey != nil && !permission.IsGranted(in.Permissions, a.APIKey.Permissions) {
		return nil, "", errs.ErrCannotGrantPermissions
	}

	name := apikey.DefaultName
	if in.Name != nil && *in.Name != "" {
		name = *in.Name
	}

	token := secret.Token()
	at := now()
	created, err := gh.Repositories(rt).APIKey.Create(ctx, &apikey.APIKey{
		ID:          uuid.New(),
		Name:        name,
		Key:         secret.Hash(token),
		UserID:      a.User.ID,
		Permissions: permission.List(in.Permissions),
		CreatedAt:   at,
		UpdatedAt:   at,
		UpdateID:    uuid.Must(uuid.NewV7()),
	})
	if err != nil {
		return nil, "", err
	}

	return created, token, nil
}

func List(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]apikey.APIKey, error) {
	return gh.Repositories(rt).APIKey.ListByUser(ctx, a.User.ID)
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*apikey.APIKey, error) {
	k, err := gh.Repositories(rt).APIKey.Get(ctx, a.User.ID, id)
	return k, notFound(err)
}

func Mine(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) (*apikey.APIKey, error) {
	if a.APIKey == nil {
		return nil, errs.ErrNotAPIKey
	}

	return Get(ctx, rt, a, a.APIKey.ID)
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*apikey.APIKey, error) {
	repos := gh.Repositories(rt)

	if _, err := repos.APIKey.Get(ctx, a.User.ID, id); err != nil {
		return nil, notFound(err)
	}
	if a.APIKey != nil && in.Permissions != nil && !permission.IsGranted(in.Permissions, a.APIKey.Permissions) {
		return nil, errs.ErrCannotGrantPermissions
	}

	var permissions permission.List
	if in.Permissions != nil {
		permissions = permission.List(in.Permissions)
	}

	k, err := repos.APIKey.Update(ctx, a.User.ID, id, in.Name, permissions, nil, now())
	return k, notFound(err)
}

func Rotate(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*apikey.APIKey, string, error) {
	repos := gh.Repositories(rt)

	existing, err := repos.APIKey.Get(ctx, a.User.ID, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, "", fmt.Errorf("%w not found", errs.ErrAPIKeyNotFound)
	}
	if err != nil {
		return nil, "", err
	}
	if a.APIKey != nil && !permission.IsGranted(existing.Permissions, a.APIKey.Permissions) {
		return nil, "", errs.ErrCannotRotateAPIKey
	}

	token := secret.Token()
	k, err := repos.APIKey.Update(ctx, a.User.ID, id, nil, nil, secret.Hash(token), now())
	if err != nil {
		return nil, "", notFound(err)
	}

	return k, token, nil
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	repos := gh.Repositories(rt)

	if _, err := repos.APIKey.Get(ctx, a.User.ID, id); err != nil {
		return notFound(err)
	}

	return repos.APIKey.Delete(ctx, a.User.ID, id)
}
