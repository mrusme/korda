package partners

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/partner"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type SearchInput struct {
	Direction partner.Direction `json:"direction"`
}

type CreateInput struct {
	SharedWithID string `json:"sharedWithId" v:"uuid"`
}

type UpdateInput struct {
	InTimeline bool `json:"inTimeline"`
}

type Shared struct {
	User       *user.User
	InTimeline bool
}

func now() time.Time {
	return time.Now().UTC()
}

func other(ctx context.Context, rt *runtime.Runtime, p *partner.Partner, direction partner.Direction) (*Shared, error) {
	id := p.SharedByID
	if direction == partner.DirectionSharedBy {
		id = p.SharedWithID
	}

	u, err := gh.Repositories(rt).User.Get(ctx, id, true)
	if err != nil {
		return nil, err
	}

	return &Shared{User: u, InTimeline: p.InTimeline}, nil
}

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]Shared, error) {
	list, err := gh.Repositories(rt).Partner.All(ctx, a.User.ID)
	if err != nil {
		return nil, err
	}

	out := []Shared{}
	for i := range list {
		p := &list[i]
		mine := p.SharedWithID
		if in.Direction == partner.DirectionSharedBy {
			mine = p.SharedByID
		}
		if mine != a.User.ID {
			continue
		}
		shared, err := other(ctx, rt, p, in.Direction)
		if err != nil {
			return nil, err
		}
		out = append(out, *shared)
	}

	return out, nil
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, sharedWithID uuid.UUID) (*Shared, error) {
	repos := gh.Repositories(rt)
	if _, err := repos.Partner.Get(ctx, a.User.ID, sharedWithID); err == nil {
		return nil, errs.ErrPartnerExists
	} else if !errors.Is(err, errs.ErrNoRows) {
		return nil, err
	}

	if _, err := repos.User.Get(ctx, sharedWithID, false); errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrInvalidUser
	} else if err != nil {
		return nil, err
	}

	p, err := repos.Partner.Create(ctx, a.User.ID, sharedWithID, now())
	if err != nil {
		return nil, err
	}

	return other(ctx, rt, p, partner.DirectionSharedBy)
}

func Remove(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, sharedWithID uuid.UUID) error {
	repos := gh.Repositories(rt)
	if _, err := repos.Partner.Get(ctx, a.User.ID, sharedWithID); errors.Is(err, errs.ErrNoRows) {
		return errs.ErrPartnerNotFound
	} else if err != nil {
		return err
	}

	return repos.Partner.Remove(ctx, a.User.ID, sharedWithID, now())
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, sharedByID uuid.UUID, in *UpdateInput) (*Shared, error) {
	if err := access.Require(ctx, rt, a, permission.PartnerUpdate, sharedByID); err != nil {
		return nil, err
	}

	p, err := gh.Repositories(rt).Partner.Update(ctx, sharedByID, a.User.ID, in.InTimeline, now())
	if err != nil {
		return nil, err
	}

	return other(ctx, rt, p, partner.DirectionSharedWith)
}
