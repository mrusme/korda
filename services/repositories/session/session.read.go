package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/session"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) GetByToken(ctx context.Context, token []byte, now time.Time) (*session.Session, error) {
	return database.Row[session.Session](ctx, repo.db, `SELECT `+activeColumns+`
		FROM sessions s JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL
		WHERE s.token = $1 AND (s.expires_at IS NULL OR s.expires_at > $2)`,
		token, now,
	)
}

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	return database.Row[session.Session](ctx, repo.db, `SELECT `+columns+` FROM sessions WHERE id = $1`, id)
}

func (repo *Repository) ListByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]session.Session, error) {
	return database.Rows[session.Session](ctx, repo.db, `SELECT `+activeColumns+`
		FROM sessions s JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL
		WHERE s.user_id = $1 AND (s.expires_at IS NULL OR s.expires_at > $2)
		ORDER BY s.updated_at DESC, s.created_at DESC`,
		userID, now,
	)
}

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	owned := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		found, err := database.Value[bool](ctx, repo.db, `SELECT EXISTS (
			SELECT 1 FROM sessions WHERE id = $1 AND user_id = $2
		)`, id, userID)
		if err != nil {
			return nil, err
		}
		if found {
			owned = append(owned, id)
		}
	}

	return owned, nil
}
