package users

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	"tty.fail/mrus/korda/services/mediaroot"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type DeleteInput struct {
	Force *bool `json:"force"`
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *DeleteInput) (*user.User, error) {
	if _, err := logicusers.Get(ctx, rt, id, false); err != nil {
		return nil, err
	}
	if a.User.ID == id {
		return nil, errs.ErrCannotDeleteOwnAccount
	}

	force := in.Force != nil && *in.Force
	status := user.StatusDeleted
	if force {
		status = user.StatusRemoving
	}

	at := now()
	deleted, err := gh.Repositories(rt).User.Update(ctx, id, repouser.Update{
		Status:    optional.Of(status),
		DeletedAt: optional.Of(at),
	}, at)
	if err != nil {
		return nil, err
	}

	if force {
		if err = gh.Jobs(rt).DeleteUser(id, true); err != nil {
			return nil, err
		}
	}

	return deleted, nil
}

func Restore(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*user.User, error) {
	if _, err := logicusers.Get(ctx, rt, id, true); err != nil {
		return nil, err
	}

	return gh.Repositories(rt).User.Restore(ctx, id, now())
}

func QueueDueDeletions(ctx context.Context, rt *runtime.Runtime) error {
	cfg, err := gh.ServerConfigFrom(rt.Config())
	if err != nil {
		return err
	}

	ids, err := gh.Repositories(rt).User.DeletedBefore(ctx, now().AddDate(0, 0, -cfg.UserDeleteDelay))
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err = gh.Jobs(rt).DeleteUser(id, false); err != nil {
			return err
		}
	}

	return nil
}

func Purge(ctx context.Context, rt *runtime.Runtime, id uuid.UUID, force bool) error {
	cfg, err := gh.ServerConfigFrom(rt.Config())
	if err != nil {
		return err
	}

	u, err := gh.Repositories(rt).User.Get(ctx, id, true)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if !force && !readyForDeletion(u, cfg.UserDeleteDelay, now()) {
		rt.Warn("admin users", "purge", "skipped", "not ready for deletion", "id", id)
		return nil
	}

	rt.Info("admin users", "purge", "id", id)

	root := gh.MediaRoot(rt)
	for _, folder := range []mediaroot.Folder{mediaroot.Upload, mediaroot.Profile, mediaroot.Thumbnails, mediaroot.EncodedVideo} {
		if err = root.RemoveAll(root.UserFolder(folder, id)); err != nil {
			return err
		}
	}

	return gh.Repositories(rt).User.Delete(ctx, id, now())
}

func readyForDeletion(u *user.User, delayDays int, at time.Time) bool {
	if !u.DeletedAt.Valid {
		return false
	}

	return at.AddDate(0, 0, -delayDays).After(u.DeletedAt.V)
}
