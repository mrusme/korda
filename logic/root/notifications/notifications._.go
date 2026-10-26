package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/jsonb"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/notification"
	"tty.fail/mrus/korda/models/permission"
	reponotification "tty.fail/mrus/korda/services/repositories/notification"
	"xn--gckvb8fzb.com/glides/runtime"
)

var (
	ErrNoReadAt   = errors.New("update without readAt")
	ErrNotUpdated = errors.New("no notification to update")
)

type SearchInput struct {
	ID     *string             `json:"id" v:"uuid"`
	Level  *notification.Level `json:"level"`
	Type   *notification.Type  `json:"type"`
	Unread *bool               `json:"unread"`
}

type UpdateInput struct {
	ReadAt optional.Value[time.Time] `json:"readAt" v:"datetime,nullable"`
}

type UpdateAllInput struct {
	IDs    []string                  `json:"ids" v:"min=1" vi:"uuid"`
	ReadAt optional.Value[time.Time] `json:"readAt" v:"datetime,nullable"`
}

type DeleteAllInput struct {
	IDs []string `json:"ids" v:"min=1" vi:"uuid"`
}

type CreateInput struct {
	Level       *notification.Level       `json:"level"`
	Type        *notification.Type        `json:"type"`
	Title       string                    `json:"title"`
	Description optional.Value[string]    `json:"description" v:"nullable"`
	Data        *json.RawMessage          `json:"data"`
	ReadAt      optional.Value[time.Time] `json:"readAt" v:"datetime,nullable"`
	UserID      string                    `json:"userId" v:"uuid"`
}

type TemplateInput struct {
	Template string `json:"template"`
}

type SMTPTransportInput struct {
	IgnoreCert bool   `json:"ignoreCert" v:"configbool"`
	Host       string `json:"host"`
	Port       int64  `json:"port" v:"min=0,max=65535"`
	Secure     bool   `json:"secure" v:"configbool"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type SMTPInput struct {
	Enabled   bool               `json:"enabled" v:"configbool"`
	From      string             `json:"from"`
	ReplyTo   string             `json:"replyTo"`
	Transport SMTPTransportInput `json:"transport"`
}

type Template struct {
	Name string
	HTML string
}

func now() time.Time {
	return time.Now().UTC()
}

func parseIDs(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		out = append(out, uuid.MustParse(id))
	}

	return out
}

func readAt(v optional.Value[time.Time]) sql.Null[time.Time] {
	return sql.Null[time.Time]{V: v.V.UTC(), Valid: v.Valid()}
}

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]notification.Notification, error) {
	f := reponotification.Filter{Level: in.Level, Type: in.Type, Unread: in.Unread != nil && *in.Unread}
	if in.ID != nil {
		f.ID = uuid.NullUUID{UUID: uuid.MustParse(*in.ID), Valid: true}
	}

	return gh.Repositories(rt).Notification.Search(ctx, a.User.ID, f)
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) (*notification.Notification, error) {
	if err := access.Require(ctx, rt, a, permission.NotificationRead, id); err != nil {
		return nil, err
	}

	n, err := gh.Repositories(rt).Notification.Deleted(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrNotificationNotFound
	}

	return n, err
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*notification.Notification, error) {
	if err := access.Require(ctx, rt, a, permission.NotificationUpdate, id); err != nil {
		return nil, err
	}
	if !in.ReadAt.Set {
		return nil, ErrNoReadAt
	}

	n, err := gh.Repositories(rt).Notification.Update(ctx, id, readAt(in.ReadAt), now())
	if errors.Is(err, errs.ErrNoRows) {
		return nil, ErrNotUpdated
	}

	return n, err
}

func UpdateAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UpdateAllInput) error {
	ids := parseIDs(in.IDs)
	if err := access.Require(ctx, rt, a, permission.NotificationUpdate, ids...); err != nil {
		return err
	}
	if !in.ReadAt.Set {
		return ErrNoReadAt
	}

	return gh.Repositories(rt).Notification.UpdateAll(ctx, ids, readAt(in.ReadAt), now())
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, ids []uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.NotificationDelete, ids...); err != nil {
		return err
	}

	return gh.Repositories(rt).Notification.Delete(ctx, ids, now())
}

func DeleteAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *DeleteAllInput) error {
	return Delete(ctx, rt, a, parseIDs(in.IDs))
}

func Create(ctx context.Context, rt *runtime.Runtime, in *CreateInput) (*notification.Notification, error) {
	c := reponotification.Create{
		UserID: uuid.MustParse(in.UserID), Level: notification.LevelInfo, Type: notification.TypeCustom, Title: in.Title,
		Description: sql.Null[string]{V: in.Description.V, Valid: in.Description.Valid()},
	}
	if in.Level != nil {
		c.Level = *in.Level
	}
	if in.Type != nil {
		c.Type = *in.Type
	}
	if in.Data != nil {
		normalized, err := jsonb.Normalize(*in.Data)
		if err != nil {
			return nil, err
		}
		c.Data = sql.Null[string]{V: string(normalized), Valid: true}
	}

	return gh.Repositories(rt).Notification.Create(ctx, c, now())
}

func RenderTemplate(name string, in *TemplateInput) Template {
	return Template{Name: name}
}

func SendTestEmail(in *SMTPInput) error {
	return errs.ErrSMTPVerify
}

func albumData(albumID uuid.UUID) (sql.Null[string], error) {
	inner, err := json.Marshal(map[string]string{"albumId": albumID.String()})
	if err != nil {
		return sql.Null[string]{}, err
	}
	outer, err := json.Marshal(string(inner))

	return sql.Null[string]{V: string(outer), Valid: true}, err
}

func albumNotification(ctx context.Context, rt *runtime.Runtime, albumID uuid.UUID, recipientID uuid.UUID, kind notification.Type, sender string) error {
	repos := gh.Repositories(rt)
	album, err := repos.Album.Get(ctx, albumID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = repos.User.Get(ctx, recipientID, false); errors.Is(err, errs.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}

	data, err := albumData(albumID)
	if err != nil {
		return err
	}
	c := reponotification.Create{UserID: recipientID, Type: kind, Data: data}
	if kind == notification.TypeAlbumInvite {
		c.Level, c.Title = notification.LevelSuccess, "Shared Album Invitation"
		c.Description = sql.Null[string]{V: sender + " shared an album (" + album.AlbumName + ") with you", Valid: true}
	} else {
		c.Level, c.Title = notification.LevelInfo, "Shared Album Update"
		c.Description = sql.Null[string]{V: "New media has been added to the album (" + album.AlbumName + ")", Valid: true}
	}

	n, err := repos.Notification.Create(ctx, c, now())
	if err != nil {
		return err
	}
	gh.Emit(ctx, rt, event.NotificationCreated{Notification: n})

	return nil
}

func AlbumInvite(ctx context.Context, rt *runtime.Runtime, albumID uuid.UUID, recipientID uuid.UUID, sender string) error {
	return albumNotification(ctx, rt, albumID, recipientID, notification.TypeAlbumInvite, sender)
}

func AlbumUpdate(ctx context.Context, rt *runtime.Runtime, albumID uuid.UUID, recipientID uuid.UUID) error {
	return albumNotification(ctx, rt, albumID, recipientID, notification.TypeAlbumUpdate, "")
}
