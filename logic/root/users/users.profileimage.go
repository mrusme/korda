package users

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/user"
	"tty.fail/mrus/korda/services/mediaroot"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

func CreateProfileImage(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, src io.Reader) (*user.User, error) {
	current, err := Get(ctx, rt, a.User.ID, false)
	if err != nil {
		return nil, err
	}

	root := gh.MediaRoot(rt)
	path := root.UserFolder(mediaroot.Profile, a.User.ID) + "/" + uuid.NewString() + ".jpg"

	if err = gh.Media(rt).Thumbnail(src, path); err != nil {
		if errors.Is(err, errs.ErrImageUndecodable) {
			return nil, errs.ErrProfileImageUnprocessed
		}
		return nil, err
	}

	at := now()
	updated, err := gh.Repositories(rt).User.Update(ctx, a.User.ID, repouser.Update{
		ProfileImagePath: optional.Of(path),
		ProfileChangedAt: optional.Of(at),
	}, at)
	if err != nil {
		root.Remove(path)
		return nil, err
	}

	if current.ProfileImagePath != "" {
		if err = root.Remove(current.ProfileImagePath); err != nil {
			rt.Warn("users", "profile image", "old file", current.ProfileImagePath, "error", err)
		}
	}

	return updated, nil
}

func DeleteProfileImage(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) error {
	current, err := Get(ctx, rt, a.User.ID, false)
	if err != nil {
		return err
	}
	if current.ProfileImagePath == "" {
		return errs.ErrProfileImageMissing
	}

	at := now()
	if _, err = gh.Repositories(rt).User.Update(ctx, a.User.ID, repouser.Update{
		ProfileImagePath: optional.Of(""),
		ProfileChangedAt: optional.Of(at),
	}, at); err != nil {
		return err
	}

	if err = gh.MediaRoot(rt).Remove(current.ProfileImagePath); err != nil {
		rt.Warn("users", "profile image", "file", current.ProfileImagePath, "error", err)
	}

	return nil
}

func ProfileImage(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (string, error) {
	u, err := gh.Repositories(rt).User.Get(ctx, id, false)
	if errors.Is(err, errs.ErrNoRows) || (err == nil && u.ProfileImagePath == "") {
		return "", errs.ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return gh.MediaRoot(rt).Abs(u.ProfileImagePath)
}

var profileExtensions = []string{".avif", ".dng", ".heic", ".heif", ".jfif", ".jpeg", ".jpg", ".png", ".webp", ".svg"}

func AcceptProfileImage(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if slices.Contains(profileExtensions, ext) {
		return nil
	}

	return fmt.Errorf("%w %s", errs.ErrUnsupportedFileType, filename)
}
