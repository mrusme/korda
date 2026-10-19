package session

import (
	"context"

	"tty.fail/mrus/korda/models/session"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Create(ctx context.Context, s *session.Session) (*session.Session, error) {
	return database.Row[session.Session](ctx, repo.db, `INSERT INTO sessions (`+columns+`) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING `+columns,
		s.ID,
		s.Token,
		s.UserID,
		s.ParentID,
		s.DeviceType,
		s.DeviceOS,
		s.AppVersion,
		s.ExpiresAt,
		s.PinExpiresAt,
		s.IsPendingSyncReset,
		s.CreatedAt,
		s.UpdatedAt,
		s.UpdateID,
	)
}
