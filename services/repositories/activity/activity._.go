package activity

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/activity"
	"tty.fail/mrus/korda/models/user"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, created_at, updated_at, album_id, user_id, asset_id, comment, is_liked, update_id`

const prefixed = `ac.id, ac.created_at, ac.updated_at, ac.album_id, ac.user_id, ac.asset_id, ac.comment, ac.is_liked,
	ac.update_id`

const userColumns = `u.email, u.name, u.avatar_color, u.profile_image_path, u.profile_changed_at`

type Repository struct {
	db *database.Database
}

type Filter struct {
	AlbumID    uuid.UUID
	UserID     uuid.NullUUID
	AssetID    uuid.NullUUID
	AlbumLevel bool
	IsLiked    *bool
}

type Row struct {
	activity.Activity
	Email            string
	Name             string
	AvatarColor      sql.Null[user.AvatarColor]
	ProfileImagePath string
	ProfileChangedAt time.Time
}

func (r *Row) User() *user.User {
	return &user.User{
		ID: r.UserID, Email: r.Email, Name: r.Name, AvatarColor: r.AvatarColor,
		ProfileImagePath: r.ProfileImagePath, ProfileChangedAt: r.ProfileChangedAt,
	}
}

type Statistics struct {
	Comments int64
	Likes    int64
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

func (repo *Repository) Search(ctx context.Context, f Filter) ([]Row, error) {
	args := []any{f.AlbumID}
	where := ` AND ac.album_id = $1`
	if f.UserID.Valid {
		args = append(args, f.UserID.UUID)
		where += ` AND ac.user_id = $` + strconv.Itoa(len(args))
	}
	switch {
	case f.AlbumLevel:
		where += ` AND ac.asset_id IS NULL`
	case f.AssetID.Valid:
		args = append(args, f.AssetID.UUID)
		where += ` AND ac.asset_id = $` + strconv.Itoa(len(args))
	}
	if f.IsLiked != nil {
		args = append(args, *f.IsLiked)
		where += ` AND ac.is_liked = $` + strconv.Itoa(len(args))
	}

	return database.Rows[Row](ctx, repo.db, `SELECT `+prefixed+`, `+userColumns+`
		FROM activities ac
		JOIN users u ON u.id = ac.user_id AND u.deleted_at IS NULL
		LEFT JOIN assets a ON a.id = ac.asset_id
		WHERE a.deleted_at IS NULL`+where+`
		ORDER BY ac.created_at, ac.rowid`, args...)
}

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*Row, error) {
	return database.Row[Row](ctx, repo.db, `SELECT `+prefixed+`, `+userColumns+`
		FROM activities ac JOIN users u ON u.id = ac.user_id WHERE ac.id = $1`, id)
}

func (repo *Repository) Create(ctx context.Context, a *activity.Activity, now time.Time) (uuid.UUID, error) {
	id := uuid.New()
	_, err := database.Exec(ctx, repo.db, `INSERT INTO activities (`+columns+`)
		VALUES ($1, $2, $2, $3, $4, $5, $6, $7, $8)`,
		id, now, a.AlbumID, a.UserID, a.AssetID, a.Comment, a.IsLiked, uuid.Must(uuid.NewV7()),
	)

	return id, err
}

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM activities WHERE id = $1`, id)
	return err
}

func (repo *Repository) Statistics(ctx context.Context, albumID uuid.UUID, assetID uuid.NullUUID) (*Statistics, error) {
	args := []any{albumID}
	where := ``
	if assetID.Valid {
		args = append(args, assetID.UUID)
		where = ` AND ac.asset_id = $2`
	}

	return database.Row[Statistics](ctx, repo.db, `SELECT
		count(*) FILTER (WHERE ac.is_liked = 0) AS comments,
		count(*) FILTER (WHERE ac.is_liked = 1) AS likes
		FROM activities ac
		JOIN users u ON u.id = ac.user_id AND u.deleted_at IS NULL
		LEFT JOIN assets a ON a.id = ac.asset_id
		WHERE ac.album_id = $1`+where+`
			AND ((a.deleted_at IS NULL AND a.visibility <> 'locked') OR a.id IS NULL)`, args...)
}

func (repo *Repository) CanCreate(ctx context.Context, userID uuid.UUID, albumIDs []uuid.UUID) ([]uuid.UUID, error) {
	args := []any{userID}
	marks := query.Placeholders(&args, albumIDs)

	return database.Values[uuid.UUID](ctx, repo.db, `SELECT DISTINCT al.id FROM albums al
		JOIN album_users au ON au.album_id = al.id
		JOIN users u ON u.id = au.user_id AND u.deleted_at IS NULL
		WHERE al.id IN (`+marks+`) AND al.is_activity_enabled = 1 AND u.id = $1 AND al.deleted_at IS NULL`, args...)
}

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM activities
			WHERE user_id = $1 AND id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) OfOwnedAlbums(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT ac.id FROM activities ac
			JOIN albums al ON al.id = ac.album_id AND al.deleted_at IS NULL
			JOIN album_users au ON au.album_id = al.id AND au.role = 'owner' AND au.user_id = $1
			WHERE ac.id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
