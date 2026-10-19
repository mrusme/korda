package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Get(ctx context.Context, id uuid.UUID, withDeleted bool) (*user.User, error) {
	return database.Row[user.User](ctx, repo.db, `SELECT `+columns+` FROM users
		WHERE id = $1 AND ($2 OR deleted_at IS NULL)`,
		id, withDeleted,
	)
}

func (repo *Repository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return database.Row[user.User](ctx, repo.db, `SELECT `+columns+` FROM users
		WHERE email = $1 AND deleted_at IS NULL`,
		email,
	)
}

func (repo *Repository) GetByStorageLabel(ctx context.Context, label string) (*user.User, error) {
	return database.Row[user.User](ctx, repo.db, `SELECT `+columns+` FROM users
		WHERE storage_label = $1 AND deleted_at IS NULL`,
		label,
	)
}

func (repo *Repository) GetAdmin(ctx context.Context) (*user.User, error) {
	return database.Row[user.User](ctx, repo.db, `SELECT `+columns+` FROM users
		WHERE is_admin = 1 AND deleted_at IS NULL
		ORDER BY created_at LIMIT 1`,
	)
}

func (repo *Repository) HasAdmin(ctx context.Context) (bool, error) {
	return database.Value[bool](ctx, repo.db, `SELECT EXISTS (
		SELECT 1 FROM users WHERE is_admin = 1 AND deleted_at IS NULL
	)`)
}

func (repo *Repository) List(ctx context.Context, id uuid.NullUUID, withDeleted bool) ([]user.User, error) {
	return database.Rows[user.User](ctx, repo.db, `SELECT `+columns+` FROM users
		WHERE ($1 IS NULL OR id = $1) AND ($2 OR deleted_at IS NULL)
		ORDER BY created_at DESC`,
		id, withDeleted,
	)
}

func (repo *Repository) DeletedBefore(ctx context.Context, before time.Time) ([]uuid.UUID, error) {
	return database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM users
		WHERE deleted_at IS NOT NULL AND deleted_at < $1`,
		before,
	)
}
