package timeline

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/collation"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/geo"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/preferences"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

type BucketsInput struct {
	UserID          *string                 `json:"userId" v:"uuid"`
	AlbumID         *string                 `json:"albumId" v:"uuid"`
	PersonID        *string                 `json:"personId" v:"uuid"`
	TagID           *string                 `json:"tagId" v:"uuid"`
	IsFavorite      *bool                   `json:"isFavorite"`
	IsTrashed       *bool                   `json:"isTrashed"`
	WithStacked     *bool                   `json:"withStacked"`
	WithPartners    *bool                   `json:"withPartners"`
	Order           *preferences.AssetOrder `json:"order"`
	OrderBy         *asset.OrderBy          `json:"orderBy"`
	Visibility      *asset.Visibility       `json:"visibility"`
	WithCoordinates *bool                   `json:"withCoordinates"`
	Key             *string                 `json:"key"`
	Slug            *string                 `json:"slug"`
	BBox            *geo.BBox               `json:"bbox"`
}

type BucketInput struct {
	BucketsInput
	TimeBucket string `json:"timeBucket"`
}

type Bucket struct {
	Assets          []repoasset.TimelineAsset
	Viewer          uuid.UUID
	ShowExif        bool
	WithCoordinates bool
	WithStacked     bool
}

func is(b *bool) bool {
	return b != nil && *b
}

func parseID(s *string) uuid.NullUUID {
	if s == nil {
		return uuid.NullUUID{}
	}

	return uuid.NullUUID{UUID: uuid.MustParse(*s), Valid: true}
}

func options(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BucketsInput) (*repoasset.Timeline, error) {
	if in.Visibility != nil && *in.Visibility == asset.VisibilityLocked && !a.Elevated {
		return nil, errs.ErrElevatedRequired
	}

	userID := parseID(in.UserID)
	albumID := parseID(in.AlbumID)
	if albumID.Valid {
		if err := access.Require(ctx, rt, a, permission.AlbumRead, albumID.UUID); err != nil {
			return nil, err
		}
	} else if !userID.Valid {
		userID = uuid.NullUUID{UUID: a.User.ID, Valid: true}
	}

	if userID.Valid {
		if err := access.Require(ctx, rt, a, permission.TimelineRead, userID.UUID); err != nil {
			return nil, err
		}
		if in.Visibility != nil && *in.Visibility == asset.VisibilityArchive {
			if err := access.Require(ctx, rt, a, permission.ArchiveRead, userID.UUID); err != nil {
				return nil, err
			}
		}
		if in.Visibility != nil && *in.Visibility == asset.VisibilityLocked && userID.UUID != a.User.ID {
			return nil, errs.ErrOtherLockedTimeline
		}
	}

	tagID := parseID(in.TagID)
	if tagID.Valid {
		if err := access.Require(ctx, rt, a, permission.TagRead, tagID.UUID); err != nil {
			return nil, err
		}
	}
	if personID := parseID(in.PersonID); personID.Valid {
		if err := access.Require(ctx, rt, a, permission.PersonRead, personID.UUID); err != nil {
			return nil, err
		}
	}

	if is(in.WithPartners) {
		v := in.Visibility
		if v == nil || *v == asset.VisibilityArchive || *v == asset.VisibilityLocked || in.IsFavorite != nil || is(in.IsTrashed) {
			return nil, errs.ErrWithPartners
		}
	}

	t := &repoasset.Timeline{
		Caller:      a.User.ID,
		AlbumID:     albumID,
		TagID:       tagID,
		IsFavorite:  in.IsFavorite,
		IsTrashed:   is(in.IsTrashed),
		WithStacked: is(in.WithStacked),
		Visibility:  in.Visibility,
		ByCreatedAt: in.OrderBy != nil && *in.OrderBy == asset.OrderByCreatedAt,
		Ascending:   in.Order != nil && *in.Order == preferences.OrderAsc,
		BBox:        in.BBox,
	}
	if userID.Valid {
		t.UserIDs = []uuid.UUID{userID.UUID}
		if is(in.WithPartners) {
			partners, err := gh.Repositories(rt).Partner.InTimeline(ctx, a.User.ID)
			if err != nil {
				return nil, err
			}
			for _, id := range partners {
				if !slices.Contains(t.UserIDs, id) {
					t.UserIDs = append(t.UserIDs, id)
				}
			}
		}
	}

	return t, nil
}

func Buckets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BucketsInput) ([]repoasset.TimeBucket, error) {
	t, err := options(ctx, rt, a, in)
	if err != nil {
		return nil, err
	}

	return gh.Repositories(rt).Asset.TimeBuckets(ctx, t)
}

var bucketForms = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02",
	"2006-1-2",
	"20060102",
}

var largeYear = regexp.MustCompile(`^0*([1-9][0-9]{4,})-(0?[1-9]|1[0-2])-(0?[1-9]|[12][0-9]|3[01])`)

func month(raw string) (string, bool, error) {
	s := raw
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		s = s[1:]
	}
	s = strings.TrimSpace(s)

	if largeYear.MatchString(s) {
		return "", false, nil
	}
	for _, form := range bucketForms {
		t, err := time.Parse(form, s)
		if err != nil {
			continue
		}
		t = t.UTC()
		if t.Day() != 1 || t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			return "", false, nil
		}
		return t.Format("2006-01"), true, nil
	}

	return "", false, errs.ErrTimeBucketFormat
}

func Get(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BucketInput) (*Bucket, error) {
	t, err := options(ctx, rt, a, &in.BucketsInput)
	if err != nil {
		return nil, err
	}

	showExif := a.SharedLink == nil || a.SharedLink.ShowExif
	b := &Bucket{
		Viewer:          a.User.ID,
		ShowExif:        showExif,
		WithCoordinates: showExif && is(in.WithCoordinates),
		WithStacked:     t.WithStacked,
		Assets:          []repoasset.TimelineAsset{},
	}

	m, ok, err := month(in.TimeBucket)
	if err != nil || !ok {
		return b, err
	}
	if b.Assets, err = gh.Repositories(rt).Asset.TimeBucket(ctx, t, m); err != nil {
		return nil, err
	}

	sign := -1
	if t.Ascending {
		sign = 1
	}
	slices.SortStableFunc(b.Assets, func(x, y repoasset.TimelineAsset) int {
		var c int
		if t.ByCreatedAt {
			c = x.CreatedAt.Truncate(time.Microsecond).Compare(y.CreatedAt.Truncate(time.Microsecond))
		} else {
			c = strings.Compare(x.LocalDateTime.UTC().Format(time.DateOnly), y.LocalDateTime.UTC().Format(time.DateOnly))
		}
		if c == 0 {
			c = x.FileCreatedAt.Compare(y.FileCreatedAt)
		}
		if c == 0 {
			c = collation.Compare(x.OriginalFileName, y.OriginalFileName)
		}
		return sign * c
	})

	return b, nil
}
