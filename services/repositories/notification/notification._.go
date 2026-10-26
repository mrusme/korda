package notification

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/notification"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, created_at, updated_at, deleted_at, update_id, user_id, level, type, data, title, description, read_at`

type Repository struct {
	db *database.Database
}

func New(db *database.Database) (*Repository, error) {
	repo := new(Repository)
	repo.db = db

	return repo, nil
}

func (repo *Repository) Startup() error {
	return nil
}

func (repo *Repository) Shutdown() error {
	return nil
}

type Filter struct {
	ID     uuid.NullUUID
	Level  *notification.Level
	Type   *notification.Type
	Unread bool
}

type Create struct {
	UserID      uuid.UUID
	Level       notification.Level
	Type        notification.Type
	Data        sql.Null[string]
	Title       string
	Description sql.Null[string]
}

func (repo *Repository) Create(ctx context.Context, c Create, now time.Time) (*notification.Notification, error) {
	return database.Row[notification.Notification](ctx, repo.db, `INSERT INTO notifications
		(id, created_at, updated_at, update_id, user_id, level, type, data, title, description)
		VALUES ($1, $2, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING `+columns,
		uuid.New(), now, uuid.Must(uuid.NewV7()), c.UserID, c.Level, c.Type, c.Data, c.Title, c.Description)
}

func (repo *Repository) Search(ctx context.Context, userID uuid.UUID, f Filter) ([]notification.Notification, error) {
	args := []any{userID}
	where := `user_id = $1 AND deleted_at IS NULL`
	add := func(cond string, v any) {
		args = append(args, v)
		where += ` AND ` + cond + ` = $` + strconv.Itoa(len(args))
	}
	if f.ID.Valid {
		add("id", f.ID.UUID)
	}
	if f.Level != nil {
		add("level", *f.Level)
	}
	if f.Type != nil {
		add("type", *f.Type)
	}
	if f.Unread {
		where += ` AND read_at IS NULL`
	}

	return database.Rows[notification.Notification](ctx, repo.db, `SELECT `+columns+` FROM notifications
		WHERE `+where+` ORDER BY created_at DESC, rowid DESC`, args...)
}

func (repo *Repository) Deleted(ctx context.Context, id uuid.UUID) (*notification.Notification, error) {
	return database.Row[notification.Notification](ctx, repo.db, `SELECT `+columns+` FROM notifications
		WHERE id = $1 AND deleted_at IS NOT NULL`, id)
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, readAt sql.Null[time.Time], now time.Time) (*notification.Notification, error) {
	return database.Row[notification.Notification](ctx, repo.db, `UPDATE notifications
		SET read_at = $1, updated_at = $2, update_id = $3 WHERE deleted_at IS NULL AND id = $4 RETURNING `+columns,
		readAt, now, uuid.Must(uuid.NewV7()), id)
}

func (repo *Repository) UpdateAll(ctx context.Context, ids []uuid.UUID, readAt sql.Null[time.Time], now time.Time) error {
	for _, id := range ids {
		if _, err := database.Exec(ctx, repo.db, `UPDATE notifications SET read_at = $1, updated_at = $2, update_id = $3
			WHERE id = $4`, readAt, now, uuid.Must(uuid.NewV7()), id); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) Delete(ctx context.Context, ids []uuid.UUID, now time.Time) error {
	for _, id := range ids {
		if _, err := database.Exec(ctx, repo.db, `UPDATE notifications SET deleted_at = $1, updated_at = $1, update_id = $2
			WHERE id = $3`, now, uuid.Must(uuid.NewV7()), id); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM notifications
			WHERE user_id = $1 AND id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
