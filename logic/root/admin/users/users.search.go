package users

import (
	"context"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type SearchInput struct {
	WithDeleted *bool   `json:"withDeleted"`
	ID          *string `json:"id" v:"uuid"`
}

func Search(ctx context.Context, rt *runtime.Runtime, in *SearchInput) ([]user.User, error) {
	var id uuid.NullUUID
	if in.ID != nil {
		id = uuid.NullUUID{UUID: uuid.MustParse(*in.ID), Valid: true}
	}

	return gh.Repositories(rt).User.List(ctx, id, in.WithDeleted != nil && *in.WithDeleted)
}
