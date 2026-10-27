package sessions

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/secret"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/session"
	"xn--gckvb8fzb.com/glides/runtime"
)

type CreateInput struct {
	Duration   *int64  `json:"duration" v:"min=1"`
	DeviceType *string `json:"deviceType"`
	DeviceOS   *string `json:"deviceOS"`
}

type UpdateInput struct {
	IsPendingSyncReset *bool `json:"isPendingSyncReset"`
}

func now() time.Time {
	return time.Now().UTC()
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*session.Session, string, error) {
	if a.Session == nil {
		return nil, "", errs.ErrSessionTokenRequired
	}

	token := secret.Token()
	at := now()
	s := &session.Session{
		ID:        uuid.New(),
		Token:     secret.Hash(token),
		UserID:    a.User.ID,
		ParentID:  uuid.NullUUID{UUID: a.Session.ID, Valid: true},
		CreatedAt: at,
		UpdatedAt: at,
		UpdateID:  uuid.Must(uuid.NewV7()),
	}
	if in.Duration != nil {
		s.ExpiresAt = sql.Null[time.Time]{V: at.Add(time.Duration(*in.Duration) * time.Second), Valid: true}
	}
	if in.DeviceType != nil {
		s.DeviceType = *in.DeviceType
	}
	if in.DeviceOS != nil {
		s.DeviceOS = *in.DeviceOS
	}

	created, err := gh.Repositories(rt).Session.Create(ctx, s)
	if err != nil {
		return nil, "", err
	}

	return created, token, nil
}

func List(ctx context.Context, rt *runtime.Runtime, userID uuid.UUID) ([]session.Session, error) {
	return gh.Repositories(rt).Session.ListByUser(ctx, userID, now())
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*session.Session, error) {
	if err := access.Require(ctx, rt, a, permission.SessionUpdate, id); err != nil {
		return nil, err
	}
	if in.IsPendingSyncReset == nil {
		return nil, errs.ErrNoFieldsToUpdate
	}

	return gh.Repositories(rt).Session.SetPendingSyncReset(ctx, id, *in.IsPendingSyncReset, now())
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.AuthDeviceDelete, id); err != nil {
		return err
	}

	if err := gh.Repositories(rt).Session.Delete(ctx, id); err != nil {
		return err
	}
	gh.Emit(ctx, rt, event.SessionsRevoked{SessionIDs: []uuid.UUID{id}})

	return nil
}

func DeleteAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) error {
	var except uuid.NullUUID
	if a.Session != nil {
		except = uuid.NullUUID{UUID: a.Session.ID, Valid: true}
	}

	ids, err := gh.Repositories(rt).Session.DeleteAllOfUser(ctx, a.User.ID, except)
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		gh.Emit(ctx, rt, event.SessionsRevoked{SessionIDs: ids})
	}

	return nil
}

func Lock(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.SessionLock, id); err != nil {
		return err
	}

	return gh.Repositories(rt).Session.SetPinExpiresAt(ctx, id, sql.Null[time.Time]{}, now())
}

func Cleanup(ctx context.Context, rt *runtime.Runtime) error {
	deleted, err := gh.Repositories(rt).Session.Cleanup(ctx, now())
	if err != nil {
		return err
	}
	if len(deleted) > 0 {
		gh.Emit(ctx, rt, event.SessionsRevoked{SessionIDs: deleted})
	}

	rt.Info("sessions", "cleanup", "deleted", len(deleted))
	return nil
}
