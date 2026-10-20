package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	logicpartners "tty.fail/mrus/korda/logic/root/partners"
	"tty.fail/mrus/korda/models/user"
)

type Partner struct {
	ID               string            `json:"id"`
	Email            string            `json:"email"`
	Name             string            `json:"name"`
	ProfileImagePath string            `json:"profileImagePath"`
	AvatarColor      user.AvatarColor  `json:"avatarColor"`
	ProfileChangedAt jsontime.Postgres `json:"profileChangedAt"`
	InTimeline       bool              `json:"inTimeline"`
}

func NewPartner(s *logicpartners.Shared) Partner {
	return Partner{
		ID:               id(s.User.ID),
		Email:            s.User.Email,
		Name:             s.User.Name,
		ProfileImagePath: s.User.ProfileImagePath,
		AvatarColor:      s.User.Color(),
		ProfileChangedAt: jsontime.NewPostgres(s.User.ProfileChangedAt),
		InTimeline:       s.InTimeline,
	}
}

func NewPartners(list []logicpartners.Shared) []Partner {
	out := make([]Partner, 0, len(list))
	for i := range list {
		out = append(out, NewPartner(&list[i]))
	}

	return out
}
