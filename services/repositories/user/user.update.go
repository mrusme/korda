package user

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Update struct {
	Email                optional.Value[string]
	Password             optional.Value[string]
	PinCode              optional.Value[string]
	Name                 optional.Value[string]
	IsAdmin              optional.Value[bool]
	ShouldChangePassword optional.Value[bool]
	AvatarColor          optional.Value[user.AvatarColor]
	ProfileImagePath     optional.Value[string]
	ProfileChangedAt     optional.Value[time.Time]
	StorageLabel         optional.Value[string]
	QuotaSizeInBytes     optional.Value[int64]
	Status               optional.Value[user.Status]
	DeletedAt            optional.Value[time.Time]
}

type assignment struct {
	column string
	value  any
}

func nullable[T any](v optional.Value[T]) any {
	if v.Null {
		return nil
	}

	return v.V
}

func (u Update) assignments() (list []assignment) {
	add := func(column string, set bool, value any) {
		if set {
			list = append(list, assignment{column, value})
		}
	}

	add("email", u.Email.Set, nullable(u.Email))
	add("password", u.Password.Set, nullable(u.Password))
	add("pin_code", u.PinCode.Set, nullable(u.PinCode))
	add("name", u.Name.Set, nullable(u.Name))
	add("is_admin", u.IsAdmin.Set, nullable(u.IsAdmin))
	add("should_change_password", u.ShouldChangePassword.Set, nullable(u.ShouldChangePassword))
	add("avatar_color", u.AvatarColor.Set, nullable(u.AvatarColor))
	add("profile_image_path", u.ProfileImagePath.Set, nullable(u.ProfileImagePath))
	add("profile_changed_at", u.ProfileChangedAt.Set, nullable(u.ProfileChangedAt))
	add("storage_label", u.StorageLabel.Set, nullable(u.StorageLabel))
	add("quota_size_in_bytes", u.QuotaSizeInBytes.Set, nullable(u.QuotaSizeInBytes))
	add("status", u.Status.Set, nullable(u.Status))
	add("deleted_at", u.DeletedAt.Set, nullable(u.DeletedAt))

	return list
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, u Update, now time.Time) (*user.User, error) {
	return repo.update(ctx, id, u.assignments(), now, `deleted_at IS NULL`)
}

func (repo *Repository) Restore(ctx context.Context, id uuid.UUID, now time.Time) (*user.User, error) {
	return repo.update(ctx, id, []assignment{
		{"status", user.StatusActive},
		{"deleted_at", nil},
	}, now, `1 = 1`)
}

func (repo *Repository) update(ctx context.Context, id uuid.UUID, list []assignment, now time.Time, condition string) (*user.User, error) {
	var set strings.Builder
	args := make([]any, 0, len(list)+3)

	for _, a := range list {
		args = append(args, a.value)
		set.WriteString(a.column + ` = $` + strconv.Itoa(len(args)) + `, `)
	}

	args = append(args, now, uuid.Must(uuid.NewV7()), id)
	set.WriteString(`updated_at = $` + strconv.Itoa(len(args)-2) + `, update_id = $` + strconv.Itoa(len(args)-1))

	return database.Row[user.User](ctx, repo.db, `UPDATE users SET `+set.String()+`
		WHERE id = $`+strconv.Itoa(len(args))+` AND `+condition+`
		RETURNING `+columns,
		args...,
	)
}

func (repo *Repository) AddUsage(ctx context.Context, id uuid.UUID, delta int64, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE users
		SET quota_usage_in_bytes = quota_usage_in_bytes + $1, updated_at = $2, update_id = $3
		WHERE id = $4 AND deleted_at IS NULL`,
		delta, now, uuid.Must(uuid.NewV7()), id,
	)

	return err
}
