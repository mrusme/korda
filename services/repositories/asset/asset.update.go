package asset

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Update struct {
	IsFavorite       optional.Value[bool]
	Visibility       optional.Value[asset.Visibility]
	LivePhotoVideoID optional.Value[uuid.UUID]
	DuplicateID      optional.Value[uuid.UUID]
	DeletedAt        optional.Value[time.Time]
	Status           optional.Value[asset.Status]
	FileCreatedAt    optional.Value[time.Time]
	FileModifiedAt   optional.Value[time.Time]
	LocalDateTime    optional.Value[time.Time]
	Duration         optional.Value[int64]
	Thumbhash        optional.Value[[]byte]
	Width            optional.Value[int64]
	Height           optional.Value[int64]
	IsOffline        optional.Value[bool]
	IsEdited         optional.Value[bool]
	StackID          optional.Value[uuid.UUID]
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

	add("is_favorite", u.IsFavorite.Set, nullable(u.IsFavorite))
	add("visibility", u.Visibility.Set, nullable(u.Visibility))
	add("live_photo_video_id", u.LivePhotoVideoID.Set, nullable(u.LivePhotoVideoID))
	add("duplicate_id", u.DuplicateID.Set, nullable(u.DuplicateID))
	add("deleted_at", u.DeletedAt.Set, nullable(u.DeletedAt))
	add("status", u.Status.Set, nullable(u.Status))
	add("file_created_at", u.FileCreatedAt.Set, nullable(u.FileCreatedAt))
	add("file_modified_at", u.FileModifiedAt.Set, nullable(u.FileModifiedAt))
	add("local_date_time", u.LocalDateTime.Set, nullable(u.LocalDateTime))
	add("duration", u.Duration.Set, nullable(u.Duration))
	add("thumbhash", u.Thumbhash.Set, nullable(u.Thumbhash))
	add("width", u.Width.Set, nullable(u.Width))
	add("height", u.Height.Set, nullable(u.Height))
	add("is_offline", u.IsOffline.Set, nullable(u.IsOffline))
	add("is_edited", u.IsEdited.Set, nullable(u.IsEdited))
	add("stack_id", u.StackID.Set, nullable(u.StackID))

	return list
}

func (u Update) Empty() bool {
	return len(u.assignments()) == 0
}

func statement(list []assignment, now time.Time) (string, []any) {
	var set strings.Builder
	args := make([]any, 0, len(list)+3)
	for _, a := range list {
		args = append(args, a.value)
		set.WriteString(a.column + ` = $` + strconv.Itoa(len(args)) + `, `)
	}
	args = append(args, now)
	set.WriteString(`updated_at = $` + strconv.Itoa(len(args)))
	args = append(args, nil)
	set.WriteString(`, update_id = $` + strconv.Itoa(len(args)))

	return set.String(), args
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, u Update, now time.Time) (*asset.Asset, error) {
	list := u.assignments()
	if len(list) == 0 {
		return repo.Get(ctx, id)
	}

	set, args := statement(list, now)
	args[len(args)-1] = uuid.Must(uuid.NewV7())
	args = append(args, id)

	return database.Row[asset.Asset](ctx, repo.db, `UPDATE assets SET `+set+`
		WHERE id = $`+strconv.Itoa(len(args))+` RETURNING `+columns,
		args...,
	)
}

func (repo *Repository) UpdateAll(ctx context.Context, ids []uuid.UUID, u Update, now time.Time) error {
	list := u.assignments()
	if len(list) == 0 || len(ids) == 0 {
		return nil
	}

	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	set, args := statement(list, now)
	args = append(args, nil)
	query := `UPDATE assets SET ` + set + ` WHERE id = $` + strconv.Itoa(len(args))

	for _, id := range ids {
		args[len(args)-2] = uuid.Must(uuid.NewV7())
		args[len(args)-1] = id
		if _, err = database.Exec(ctx, tx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) SetVisibility(ctx context.Context, ids []uuid.UUID, v asset.Visibility, now time.Time) error {
	return repo.UpdateAll(ctx, ids, Update{Visibility: optional.Of(v)}, now)
}

func optionalStatus(s asset.Status) optional.Value[asset.Status] {
	return optional.Of(s)
}
