package clustergroups

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type RequestInput struct {
	UserID string `json:"userId" v:"uuid"`
}

func member(a *auth.Auth, id uuid.UUID, p permission.Permission) error {
	if a.User.ClusterGroupID == id {
		return nil
	}

	return access.Denied(p)
}

func Users(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) ([]user.User, error) {
	if err := member(a, id, permission.ClusterGroupRead); err != nil {
		return nil, err
	}

	return []user.User{*a.User}, nil
}

func Requests(a *auth.Auth, id uuid.UUID) error {
	return member(a, id, permission.ClusterGroupRead)
}

func CreateRequest(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *RequestInput) error {
	if err := member(a, id, permission.ClusterGroupRequestCreate); err != nil {
		return err
	}
	userID := uuid.MustParse(in.UserID)
	if userID == a.User.ID {
		return errs.ErrOwnClusterGroup
	}
	if _, err := gh.Repositories(rt).User.Get(ctx, userID, false); errors.Is(err, errs.ErrNoRows) {
		return errs.ErrUserNotFound
	} else if err != nil {
		return err
	}

	return errs.UnsupportedClusters
}

func Accept() error {
	return access.Denied(permission.ClusterGroupRequestRead)
}

func DeleteRequest() error {
	return access.Denied(permission.ClusterGroupRequestDelete)
}

func Leave(a *auth.Auth, id uuid.UUID) error {
	if err := member(a, id, permission.ClusterGroupLeave); err != nil {
		return err
	}

	return errs.ErrLastClusterMember
}

func RegeneratePeople(a *auth.Auth, id uuid.UUID) error {
	return member(a, id, permission.ClusterGroupRead)
}
