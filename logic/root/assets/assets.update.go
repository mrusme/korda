package assets

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/mediaroot"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	repoexif "tty.fail/mrus/korda/services/repositories/assetexif"
	"xn--gckvb8fzb.com/glides/runtime"
)

var (
	ErrDateTimeUnparsable = errors.New("dateTimeOriginal is not a timestamp")

	isoOffset = regexp.MustCompile(`(?:[T ]\d{2}(?::?\d{2}(?::?\d{2}(?:[.,]\d+)?)?)?)(Z|[+-]\d{2}(?::?\d{2})?)$`)
)

func ParseDateTime(s string) (time.Time, error) {
	value := strings.TrimSpace(s)
	layouts := []string{
		"2006-01-02T15:04:05.999999999Z07:00", "2006-01-02T15:04:05.999999999Z0700", "2006-01-02T15:04:05.999999999Z07",
		"2006-01-02T15:04Z07:00", "2006-01-02T15:04:05.999999999", "2006-01-02T15:04",
		"2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999Z0700", "2006-01-02 15:04:05.999999999Z07",
		"2006-01-02 15:04:05.999999999", "2006-01-02 15:04", "2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: %q", ErrDateTimeUnparsable, s)
}

func FixedZone(s string) (string, bool) {
	m := isoOffset.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}

	offset := m[1]
	if offset == "Z" {
		return "UTC", true
	}

	digits := strings.ReplaceAll(offset[1:], ":", "")
	hours, _ := strconv.Atoi(digits[:2])
	minutes := 0
	if len(digits) >= 4 {
		minutes, _ = strconv.Atoi(digits[2:4])
	}

	return ZoneName(offset[0] == '-', hours, minutes), true
}

func ZoneName(negative bool, hours int, minutes int) string {
	if hours == 0 && minutes == 0 {
		return "UTC"
	}

	sign := "+"
	if negative {
		sign = "-"
	}
	name := "UTC" + sign + strconv.Itoa(hours)
	if minutes != 0 {
		name += fmt.Sprintf(":%02d", minutes)
	}

	return name
}

func exifWrites(description *string, dateTimeOriginal *string, latitude *float64, longitude *float64, rating optional.Value[int64]) ([]repoexif.Write, error) {
	var writes []repoexif.Write
	if description != nil {
		writes = append(writes, repoexif.Write{Column: "description", Value: *description})
	}
	if dateTimeOriginal != nil {
		t, err := ParseDateTime(*dateTimeOriginal)
		if err != nil {
			return nil, err
		}
		writes = append(writes, repoexif.Write{Column: "date_time_original", Value: t})
		if zone, ok := FixedZone(*dateTimeOriginal); ok {
			writes = append(writes, repoexif.Write{Column: "time_zone", Value: zone})
		}
	}
	if latitude != nil {
		writes = append(writes, repoexif.Write{Column: "latitude", Value: *latitude})
	}
	if longitude != nil {
		writes = append(writes, repoexif.Write{Column: "longitude", Value: *longitude})
	}
	if rating.Set {
		var value any
		if !rating.Null {
			value = rating.V
		}
		writes = append(writes, repoexif.Write{Column: "rating", Value: value})
	}

	return writes, nil
}

func isAndroidMotionPath(path string) bool {
	return strings.HasPrefix(path, string(mediaroot.EncodedVideo)+"/")
}

func sidecarWrite(rt *runtime.Runtime, ids ...uuid.UUID) {
	for _, id := range ids {
		if err := gh.Jobs(rt).Metadata(jobs.AssetPayload{AssetID: id, Source: jobs.SourceSidecarWrite}); err != nil {
			rt.Error("assets", "sidecar write", "asset", id.String(), "error", err)
		}
	}
}

func Update(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *UpdateInput) (*Details, error) {
	if err := require(ctx, rt, a, permission.AssetUpdate, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	at := now()

	var previousMotion *asset.Asset
	switch {
	case in.LivePhotoVideoID.Valid():
		if err := beforeLink(ctx, rt, a.User.ID, uuid.MustParse(in.LivePhotoVideoID.V), at); err != nil {
			return nil, err
		}
	case in.LivePhotoVideoID.Set && in.LivePhotoVideoID.Null:
		current, err := find(ctx, rt, id)
		if err != nil {
			return nil, err
		}
		if current.LivePhotoVideoID.Valid {
			motion, err := repos.Asset.Get(ctx, current.LivePhotoVideoID.UUID)
			if err != nil && !errors.Is(err, errs.ErrNoRows) {
				return nil, err
			}
			if motion != nil && isAndroidMotionPath(motion.OriginalPath) {
				return nil, errs.ErrUnlinkAndroidMotion
			}
			previousMotion = motion
		}
	}

	writes, err := exifWrites(in.Description, in.DateTimeOriginal, in.Latitude, in.Longitude, in.Rating)
	if err != nil {
		return nil, err
	}
	if len(writes) > 0 {
		if err = repos.AssetExif.Write(ctx, nil, id, writes, repoexif.Append, repoexif.LockedFor(writes), at); err != nil {
			return nil, err
		}
	}

	u := repoasset.Update{}
	if in.IsFavorite != nil {
		u.IsFavorite = optional.Of(*in.IsFavorite)
	}
	if in.Visibility != nil {
		u.Visibility = optional.Of(*in.Visibility)
	}
	if in.LivePhotoVideoID.Set {
		u.LivePhotoVideoID.Set, u.LivePhotoVideoID.Null = true, in.LivePhotoVideoID.Null
		if !in.LivePhotoVideoID.Null {
			u.LivePhotoVideoID.V = uuid.MustParse(in.LivePhotoVideoID.V)
		}
	}

	updated, err := repos.Asset.Update(ctx, id, u, at)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrAssetNotFound
	}
	if err != nil {
		return nil, err
	}

	if previousMotion != nil {
		if err = repos.Asset.SetVisibility(ctx, []uuid.UUID{previousMotion.ID}, updated.Visibility, at); err != nil {
			return nil, err
		}
		if err = gh.Jobs(rt).Thumbnails(jobs.AssetPayload{AssetID: previousMotion.ID, Notify: true}); err != nil {
			rt.Error("assets", "update", "queue thumbnails", previousMotion.ID.String(), "error", err)
		}
	}

	if len(writes) > 0 {
		sidecarWrite(rt, id)
	}

	return Get(ctx, rt, a, id)
}

func UpdateAll(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BulkUpdateInput) error {
	ids := parseIDs(in.IDs)
	if err := require(ctx, rt, a, permission.AssetUpdate, ids...); err != nil {
		return err
	}

	repos := gh.Repositories(rt)
	at := now()

	var writes []repoexif.Write
	if in.Latitude != nil {
		writes = append(writes, repoexif.Write{Column: "latitude", Value: *in.Latitude})
	}
	if in.Longitude != nil {
		writes = append(writes, repoexif.Write{Column: "longitude", Value: *in.Longitude})
	}
	if in.Rating.Set {
		var value any
		if !in.Rating.Null {
			value = in.Rating.V
		}
		writes = append(writes, repoexif.Write{Column: "rating", Value: value})
	}
	if in.Description != nil {
		writes = append(writes, repoexif.Write{Column: "description", Value: *in.Description})
	}
	if in.DateTimeOriginal != nil {
		t, err := ParseDateTime(*in.DateTimeOriginal)
		if err != nil {
			return err
		}
		writes = append(writes, repoexif.Write{Column: "date_time_original", Value: t})
	}

	writeSidecar := false
	if len(writes) > 0 {
		lock := repoexif.Names(writes)
		if err := repos.AssetExif.UpdateAll(ctx, ids, writes, lock, at); err != nil {
			return err
		}
		writeSidecar = true
	}

	var zone *string
	if in.TimeZone != nil {
		zone = in.TimeZone
	} else if in.DateTimeOriginal != nil {
		if name, ok := FixedZone(*in.DateTimeOriginal); ok {
			zone = &name
		}
	}
	if (in.DateTimeRelative != nil && *in.DateTimeRelative != 0) || zone != nil {
		var minutes int64
		if in.DateTimeRelative != nil {
			minutes = *in.DateTimeRelative
		}
		if err := repos.AssetExif.ShiftDateTimeOriginal(ctx, ids, minutes, zone, at); err != nil {
			return err
		}
		writeSidecar = true
	}

	u := repoasset.Update{}
	if in.IsFavorite != nil {
		u.IsFavorite = optional.Of(*in.IsFavorite)
	}
	if in.Visibility != nil {
		u.Visibility = optional.Of(*in.Visibility)
	}
	if in.DuplicateID.Set {
		u.DuplicateID.Set, u.DuplicateID.Null = true, in.DuplicateID.Null
		if !in.DuplicateID.Null {
			parsed, err := uuid.Parse(in.DuplicateID.V)
			if err != nil {
				return fmt.Errorf("duplicateId %q: %w", in.DuplicateID.V, err)
			}
			u.DuplicateID.V = parsed
		}
	}
	if err := repos.Asset.UpdateAll(ctx, ids, u, at); err != nil {
		return err
	}
	if in.Visibility != nil && *in.Visibility == asset.VisibilityLocked {
		if err := repos.Album.RemoveAssetsFromAll(ctx, ids, at); err != nil {
			return err
		}
	}

	if writeSidecar {
		sidecarWrite(rt, ids...)
	}

	return nil
}
