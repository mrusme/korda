package view

import (
	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/session"
)

type Session struct {
	ID                 string         `json:"id"`
	CreatedAt          jsontime.Time  `json:"createdAt"`
	UpdatedAt          jsontime.Time  `json:"updatedAt"`
	ExpiresAt          *jsontime.Time `json:"expiresAt,omitempty"`
	Current            bool           `json:"current"`
	AppVersion         *string        `json:"appVersion"`
	DeviceOS           string         `json:"deviceOS"`
	DeviceType         string         `json:"deviceType"`
	IsPendingSyncReset bool           `json:"isPendingSyncReset"`
}

type SessionCreate struct {
	Session
	Token string `json:"token"`
}

func NewSession(s *session.Session, current uuid.UUID) Session {
	return Session{
		ID:                 id(s.ID),
		CreatedAt:          time3(s.CreatedAt),
		UpdatedAt:          time3(s.UpdatedAt),
		ExpiresAt:          jsontime.Nullable(s.ExpiresAt),
		Current:            current != uuid.Nil && s.ID == current,
		AppVersion:         nullable(s.AppVersion),
		DeviceOS:           s.DeviceOS,
		DeviceType:         s.DeviceType,
		IsPendingSyncReset: s.IsPendingSyncReset,
	}
}

func NewSessions(list []session.Session, current uuid.UUID) []Session {
	out := make([]Session, 0, len(list))
	for i := range list {
		out = append(out, NewSession(&list[i], current))
	}

	return out
}

func NewSessionCreate(s *session.Session, token string) SessionCreate {
	return SessionCreate{Session: NewSession(s, uuid.Nil), Token: token}
}
