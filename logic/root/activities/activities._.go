package activities

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/activity"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	repoactivity "tty.fail/mrus/korda/services/repositories/activity"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Level string

const (
	LevelAlbum Level = "album"
	LevelAsset Level = "asset"
)

func (Level) Enum() []string {
	return []string{string(LevelAlbum), string(LevelAsset)}
}

type SearchInput struct {
	AlbumID string         `json:"albumId" v:"uuid"`
	AssetID *string        `json:"assetId" v:"uuid"`
	Type    *activity.Type `json:"type"`
	Level   *Level         `json:"level"`
	UserID  *string        `json:"userId" v:"uuid"`
}

type StatisticsInput struct {
	AlbumID string  `json:"albumId" v:"uuid"`
	AssetID *string `json:"assetId" v:"uuid"`
}

type CreateInput struct {
	AlbumID string        `json:"albumId" v:"uuid"`
	AssetID *string       `json:"assetId" v:"uuid"`
	Type    activity.Type `json:"type"`
	Comment *string       `json:"comment"`
}

func (CreateInput) Refine(in refine.Fields) []refine.Issue {
	kind, _ := in.Get("type")
	raw, _ := in.Get("comment")
	comment, _ := raw.(string)

	var out []refine.Issue
	if kind == string(activity.TypeComment) && strings.TrimSpace(comment) == "" {
		out = append(out, refine.At([]any{"comment"}, "Comment is required when type is COMMENT"))
	}
	if kind != string(activity.TypeComment) && comment != "" {
		out = append(out, refine.At([]any{"comment"}, "Comment must not be provided when type is not COMMENT"))
	}

	return out
}

func now() time.Time {
	return time.Now().UTC()
}

func optionalID(s *string) uuid.NullUUID {
	if s == nil {
		return uuid.NullUUID{}
	}

	return uuid.NullUUID{UUID: uuid.MustParse(*s), Valid: true}
}

func Search(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *SearchInput) ([]repoactivity.Row, error) {
	albumID := uuid.MustParse(in.AlbumID)
	if err := access.Require(ctx, rt, a, permission.AlbumRead, albumID); err != nil {
		return nil, err
	}

	f := repoactivity.Filter{AlbumID: albumID, UserID: optionalID(in.UserID)}
	if in.Level != nil && *in.Level == LevelAlbum {
		f.AlbumLevel = true
	} else {
		f.AssetID = optionalID(in.AssetID)
	}
	if in.Type != nil {
		liked := *in.Type == activity.TypeLike
		f.IsLiked = &liked
	}

	return gh.Repositories(rt).Activity.Search(ctx, f)
}

func Statistics(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *StatisticsInput) (*repoactivity.Statistics, error) {
	albumID := uuid.MustParse(in.AlbumID)
	if err := access.Require(ctx, rt, a, permission.AlbumRead, albumID); err != nil {
		return nil, err
	}

	return gh.Repositories(rt).Activity.Statistics(ctx, albumID, optionalID(in.AssetID))
}

func Create(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *CreateInput) (*repoactivity.Row, bool, error) {
	albumID := uuid.MustParse(in.AlbumID)
	if err := access.Require(ctx, rt, a, permission.ActivityCreate, albumID); err != nil {
		return nil, false, err
	}

	repos := gh.Repositories(rt)
	assetID := optionalID(in.AssetID)
	if in.Type == activity.TypeLike {
		liked := true
		f := repoactivity.Filter{AlbumID: albumID, UserID: uuid.NullUUID{UUID: a.User.ID, Valid: true}, IsLiked: &liked}
		if assetID.Valid {
			f.AssetID = assetID
		} else {
			f.AlbumLevel = true
		}
		existing, err := repos.Activity.Search(ctx, f)
		if err != nil {
			return nil, false, err
		}
		if len(existing) > 0 {
			return &existing[0], true, nil
		}
	}

	created := &activity.Activity{AlbumID: albumID, UserID: a.User.ID, AssetID: assetID, IsLiked: in.Type == activity.TypeLike}
	if in.Type != activity.TypeLike && in.Comment != nil {
		created.Comment = sql.Null[string]{V: *in.Comment, Valid: true}
	}

	id, err := repos.Activity.Create(ctx, created, now())
	if err != nil {
		return nil, false, err
	}

	row, err := repos.Activity.Get(ctx, id)

	return row, false, err
}

func Delete(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.ActivityDelete, id); err != nil {
		return err
	}

	return gh.Repositories(rt).Activity.Delete(ctx, id)
}
