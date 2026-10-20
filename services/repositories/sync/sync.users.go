package sync

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type AuthUser struct {
	ID                uuid.UUID
	Name              string
	Email             string
	AvatarColor       sql.Null[string]
	DeletedAt         sql.Null[time.Time]
	UpdateID          string
	ProfileImagePath  string
	ProfileChangedAt  time.Time
	IsAdmin           bool
	PinCode           sql.Null[string]
	OAuthID           sql.Null[string]
	StorageLabel      sql.Null[string]
	QuotaSizeInBytes  sql.Null[int64]
	QuotaUsageInBytes int64
}

type User struct {
	ID               uuid.UUID
	Name             string
	Email            string
	AvatarColor      sql.Null[string]
	DeletedAt        sql.Null[time.Time]
	UpdateID         string
	ProfileImagePath string
	ProfileChangedAt time.Time
}

type UserDelete struct {
	ID     string
	UserID uuid.UUID
}

const userColumns = `id, name, email, avatar_color, deleted_at, update_id, profile_image_path, profile_changed_at`

func (repo *Repository) AuthUsers(ctx context.Context, userID uuid.UUID, b Bounds) ([]AuthUser, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, userID)

	return database.Rows[AuthUser](ctx, repo.db, `SELECT `+userColumns+`, is_admin, pin_code, oauth_id,
		storage_label, quota_size_in_bytes, quota_usage_in_bytes
		FROM users WHERE `+where+` AND id = $`+strconv.Itoa(len(args))+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) Users(ctx context.Context, b Bounds) ([]User, error) {
	args := []any{}
	where := b.where("update_id", &args)

	return database.Rows[User](ctx, repo.db, `SELECT `+userColumns+` FROM users WHERE `+where+b.tail("update_id"), args...)
}

func (repo *Repository) UserDeletes(ctx context.Context, b Bounds) ([]UserDelete, error) {
	args := []any{}
	where := b.where("id", &args)

	return database.Rows[UserDelete](ctx, repo.db, `SELECT id, user_id FROM user_audit WHERE `+where+b.tail("id"), args...)
}

type UserMetadata struct {
	UserID   uuid.UUID
	Key      string
	Value    string
	UpdateID string
}

type UserMetadataDelete struct {
	ID     string
	UserID uuid.UUID
	Key    string
}

func (repo *Repository) UserMetadata(ctx context.Context, userID uuid.UUID, b Bounds) ([]UserMetadata, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, userID)

	return database.Rows[UserMetadata](ctx, repo.db, `SELECT user_id, key, value, update_id FROM user_metadata
		WHERE `+where+` AND user_id = $`+strconv.Itoa(len(args))+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) UserMetadataDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]UserMetadataDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)

	return database.Rows[UserMetadataDelete](ctx, repo.db, `SELECT id, user_id, key FROM user_metadata_audit
		WHERE `+where+` AND user_id = $`+strconv.Itoa(len(args))+b.tail("id"),
		args...,
	)
}
