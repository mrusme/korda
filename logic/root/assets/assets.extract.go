package assets

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/helpers/optional"
	logictags "tty.fail/mrus/korda/logic/root/tags"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/mediaroot"
	"tty.fail/mrus/korda/services/metadata"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	repoexif "tty.fail/mrus/korda/services/repositories/assetexif"
	"xn--gckvb8fzb.com/glides/runtime"
)

const postgresIntMax = 2147483647

type Dates struct {
	DateTimeOriginal time.Time
	LocalDateTime    time.Time
	TimeZone         *string
}

func Locked(e *assetexif.Exif) []string {
	if e == nil || !e.LockedProperties.Valid {
		return nil
	}

	var out []string
	_ = json.Unmarshal([]byte(e.LockedProperties.V), &out)
	return out
}

func zoneLocation(name string) *time.Location {
	if offset, ok := metadata.ZoneOffset(name); ok {
		return time.FixedZone(name, offset*60)
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}

	return time.UTC
}

func localWall(instant time.Time, zone *string) time.Time {
	loc := time.UTC
	if zone != nil {
		loc = zoneLocation(*zone)
	}
	wall := instant.In(loc)

	return time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), time.UTC)
}

func ComputeDates(fileCreatedAt time.Time, tags *metadata.Tags, modified time.Time, born time.Time) Dates {
	var zone *string
	switch {
	case tags.TZ != "":
		name := tags.TZ
		zone = &name
	case tags.Zone != nil:
		name := metadata.ZoneName(*tags.Zone)
		zone = &name
	}

	d, ok := tags.FirstDate()
	if !ok {
		earliest := modified
		if !born.IsZero() && born.Before(earliest) {
			earliest = born
		}
		if fileCreatedAt.Before(earliest) {
			earliest = fileCreatedAt
		}
		earliest = earliest.UTC().Truncate(time.Millisecond)
		return Dates{DateTimeOriginal: earliest, LocalDateTime: earliest, TimeZone: zone}
	}

	if zone == nil && (strings.HasSuffix(d.Raw, "Z") || strings.HasSuffix(d.Raw, "+00:00")) {
		utc := "UTC+0"
		zone = &utc
	}

	var instant time.Time
	switch {
	case d.Offset != nil:
		instant = d.Instant(nil)
	case tags.IsVideo || strings.Contains(d.Tag, "UTC") || strings.HasPrefix(d.Tag, "GPS"):
		instant = d.Wall
	default:
		instant = tags.Instant(d)
	}
	instant = instant.UTC()

	return Dates{DateTimeOriginal: instant, LocalDateTime: localWall(instant, zone), TimeZone: zone}
}

func validInt(v int) sql.Null[int64] {
	if v <= 0 || v > postgresIntMax {
		return sql.Null[int64]{}
	}

	return sql.Null[int64]{V: int64(v), Valid: true}
}

func nullString(s string) any {
	if s == "" {
		return nil
	}

	return s
}

func lensModel(s string) any {
	s = strings.TrimSpace(s)
	if s == "" || s == "----" || strings.HasPrefix(s, "Unknown") {
		return nil
	}

	return s
}

func rating(r *float64) any {
	if r == nil || *r == 0 || math.IsNaN(*r) || *r < 1 || *r > 5 {
		return nil
	}

	return int64(math.Round(*r))
}

func bitsPerSample(b *int64) any {
	if b == nil {
		return nil
	}
	v := *b
	if v >= 24 && v%3 == 0 {
		v /= 3
	}

	return v
}

func finite(f *float64) any {
	if f == nil || math.IsNaN(*f) || math.IsInf(*f, 0) || *f > postgresIntMax || *f < -postgresIntMax-1 {
		return nil
	}

	return *f
}

func Extract(ctx context.Context, rt *runtime.Runtime, p jobs.AssetPayload) error {
	repos := gh.Repositories(rt)
	root := gh.MediaRoot(rt)

	a, err := repos.Asset.Get(ctx, p.AssetID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	abs, err := root.Abs(a.OriginalPath)
	if err != nil {
		return err
	}

	mediacfg, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return err
	}

	tags, err := metadata.Read(ctx, abs, mediacfg.FFprobe, string(a.Type))
	if err != nil {
		return err
	}

	files, err := repos.AssetFile.List(ctx, a.ID)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Type != assetfile.TypeSidecar {
			continue
		}
		sidecarPath, err := root.Abs(f.Path)
		if err != nil {
			continue
		}
		if sidecar, err := metadata.ReadSidecar(sidecarPath); err == nil {
			tags.Merge(sidecar)
		}
	}

	size, modified, born, err := metadata.Stat(abs)
	if err != nil {
		return err
	}

	existing, err := Exif(ctx, rt, a.ID)
	if err != nil {
		return err
	}

	dates := ComputeDates(a.FileCreatedAt, tags, modified, born)
	if locked := Locked(existing); slices.Contains(locked, "dateTimeOriginal") && existing.DateTimeOriginal.Valid {
		dates.DateTimeOriginal = existing.DateTimeOriginal.V.UTC()
		zone := dates.TimeZone
		if existing.TimeZone.Valid {
			zone = &existing.TimeZone.V
		}
		dates.LocalDateTime = localWall(dates.DateTimeOriginal, zone)
	}

	at := now()
	width, height := validInt(tags.Width), validInt(tags.Height)
	if metadata.Sidewards(tags.Orientation) {
		width, height = height, width
	}

	u := repoasset.Update{
		LocalDateTime:  optional.Of(dates.LocalDateTime),
		FileCreatedAt:  optional.Of(dates.DateTimeOriginal),
		FileModifiedAt: optional.Of(modified.UTC().Truncate(time.Millisecond)),
	}
	switch {
	case tags.Duration != nil:
		u.Duration = optional.Of(int64(math.Round(*tags.Duration * 1000)))
	case a.Type == asset.TypeVideo && tags.Video == nil:
	default:
		u.Duration.Set, u.Duration.Null = true, true
	}
	if !a.IsEdited || !a.Width.Valid {
		u.Width = optional.Value[int64]{Set: true, Null: !width.Valid, V: width.V}
	}
	if !a.IsEdited || !a.Height.Valid {
		u.Height = optional.Value[int64]{Set: true, Null: !height.Valid, V: height.V}
	}

	updated, err := repos.Asset.Update(ctx, a.ID, u, at)
	if err != nil {
		return err
	}
	if err = writeExif(ctx, rt, updated, tags, dates, size, modified, at); err != nil {
		return err
	}
	if err = applyTagList(ctx, rt, updated); err != nil {
		return err
	}

	if updated.Type == asset.TypeImage && (tags.MotionPhoto || tags.MicroVideo) {
		if err := extractMotion(ctx, rt, updated, tags, dates, abs, size, modified); err != nil {
			rt.Error("assets", "motion photo", "asset", a.ID.String(), "error", err)
		}
	}

	if tags.ContentIdentifier != "" {
		if err = linkLivePhotos(ctx, rt, updated, tags.ContentIdentifier); err != nil {
			return err
		}
	}

	if err = repos.Asset.MetadataExtracted(ctx, a.ID, now()); err != nil {
		return err
	}
	if p.Source == jobs.SourceSidecarWrite {
		gh.Emit(ctx, rt, event.AssetUpdated{UserID: a.OwnerID, AssetID: a.ID, Detail: event.DetailFull})
	}

	if p.Source == jobs.SourceUpload || p.Source == jobs.SourceCopy {
		return gh.Jobs(rt).Thumbnails(jobs.AssetPayload{AssetID: a.ID, Source: p.Source})
	}

	return nil
}

func writeExif(ctx context.Context, rt *runtime.Runtime, a *asset.Asset, tags *metadata.Tags, dates Dates, size int64, modified time.Time, at time.Time) error {
	var orientation any
	if tags.Orientation != nil && *tags.Orientation > 0 {
		orientation = strconv.Itoa(*tags.Orientation)
	}
	var latitude, longitude any
	if tags.Latitude != nil && tags.Longitude != nil {
		latitude, longitude = *tags.Latitude, *tags.Longitude
	}
	var iso any
	if tags.ISO != nil && *tags.ISO <= postgresIntMax {
		iso = *tags.ISO
	}
	var zone any
	if dates.TimeZone != nil {
		zone = *dates.TimeZone
	}
	var keywords any
	if len(tags.Keywords) > 0 {
		encoded, err := json.Marshal(tags.Keywords)
		if err != nil {
			return err
		}
		keywords = string(encoded)
	}
	width, height := validInt(tags.Width), validInt(tags.Height)

	writes := []repoexif.Write{
		{Column: "date_time_original", Value: dates.DateTimeOriginal},
		{Column: "modify_date", Value: modified.UTC().Truncate(time.Microsecond)},
		{Column: "time_zone", Value: zone},
		{Column: "latitude", Value: latitude},
		{Column: "longitude", Value: longitude},
		{Column: "country", Value: nil},
		{Column: "state", Value: nil},
		{Column: "city", Value: nil},
		{Column: "file_size_in_byte", Value: size},
		{Column: "exif_image_height", Value: height},
		{Column: "exif_image_width", Value: width},
		{Column: "orientation", Value: orientation},
		{Column: "projection_type", Value: nullString(tags.ProjectionType)},
		{Column: "bits_per_sample", Value: bitsPerSample(tags.BitsPerSample)},
		{Column: "colorspace", Value: nullString(tags.ColorSpace)},
		{Column: "make", Value: nullString(tags.Make)},
		{Column: "model", Value: nullString(tags.Model)},
		{Column: "fps", Value: finite(tags.FrameRate)},
		{Column: "iso", Value: iso},
		{Column: "exposure_time", Value: nullString(tags.ExposureTime)},
		{Column: "lens_model", Value: lensModel(tags.LensModel)},
		{Column: "f_number", Value: finite(tags.FNumber)},
		{Column: "focal_length", Value: finite(tags.FocalLength)},
		{Column: "description", Value: strings.TrimSpace(tags.Description)},
		{Column: "profile_description", Value: nil},
		{Column: "rating", Value: rating(tags.Rating)},
		{Column: "live_photo_cid", Value: nullString(tags.ContentIdentifier)},
		{Column: "auto_stack_id", Value: nullString(tags.BurstID)},
		{Column: "tags", Value: keywords},
	}

	return gh.Repositories(rt).AssetExif.Write(ctx, nil, a.ID, writes, repoexif.Skip, nil, at)
}

func linkLivePhotos(ctx context.Context, rt *runtime.Runtime, a *asset.Asset, cid string) error {
	repos := gh.Repositories(rt)
	other := asset.TypeVideo
	if a.Type == asset.TypeVideo {
		other = asset.TypeImage
	}

	match, err := repos.Asset.LivePhotoMatch(ctx, a.OwnerID, cid, a.ID, other)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	photo, motion := a, match
	if a.Type == asset.TypeVideo {
		photo, motion = match, a
	}

	at := now()
	if _, err = repos.Asset.Update(ctx, photo.ID, repoasset.Update{LivePhotoVideoID: optional.Of(motion.ID)}, at); err != nil {
		return err
	}
	if err = repos.Asset.SetVisibility(ctx, []uuid.UUID{motion.ID}, asset.VisibilityHidden, at); err != nil {
		return err
	}
	if err = repos.Album.RemoveAssetsFromAll(ctx, []uuid.UUID{motion.ID}, at); err != nil {
		return err
	}
	gh.Emit(ctx, rt, event.AssetHidden{UserID: motion.OwnerID, AssetID: motion.ID})

	return nil
}

func extractMotion(ctx context.Context, rt *runtime.Runtime, still *asset.Asset, tags *metadata.Tags, dates Dates, abs string, size int64, modified time.Time) error {
	var length, padding int64
	if tags.MotionPhoto {
		for _, item := range tags.Directory {
			if item.Semantic == "MotionPhoto" {
				length, padding = item.Length, item.Padding
				break
			}
		}
	}
	if tags.MicroVideo && tags.MicroVideoOffset > 0 {
		length = tags.MicroVideoOffset
	}
	position := size - length - padding
	if length <= 0 || position < 0 {
		return nil
	}

	f, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer f.Close()

	videoBytes := make([]byte, length)
	if _, err = f.ReadAt(videoBytes, position); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	sum := sha1.Sum(videoBytes)
	checksum := sum[:]

	repos := gh.Repositories(rt)
	root := gh.MediaRoot(rt)
	at := now()

	motion, err := repos.Asset.ByChecksum(ctx, still.OwnerID, checksum)
	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		return err
	}
	if motion == nil {
		id := uuid.New()
		created := &asset.Asset{
			ID:                id,
			OwnerID:           still.OwnerID,
			Type:              asset.TypeVideo,
			OriginalPath:      root.Nested(mediaroot.EncodedVideo, still.OwnerID, id.String()+"-MP.mp4"),
			OriginalFileName:  mimetypes.Base(still.OriginalFileName) + ".mp4",
			Checksum:          checksum,
			ChecksumAlgorithm: asset.ChecksumSHA1,
			FileCreatedAt:     dates.DateTimeOriginal,
			FileModifiedAt:    modified.UTC().Truncate(time.Millisecond),
			LocalDateTime:     dates.LocalDateTime,
			Visibility:        asset.VisibilityHidden,
			Status:            asset.StatusActive,
			LibraryID:         still.LibraryID,
			CreatedAt:         at,
			UpdatedAt:         at,
			UpdateID:          uuid.Must(uuid.NewV7()),
		}
		usage := int64(len(videoBytes))
		if still.IsExternal {
			usage = 0
		}
		motion, err = repos.Asset.Insert(ctx, created, usage)
		if errors.Is(err, errs.ErrUniqueViolationOn) {
			if motion, err = repos.Asset.ByChecksum(ctx, still.OwnerID, checksum); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	if motion.Visibility == asset.VisibilityTimeline {
		if err = repos.Asset.SetVisibility(ctx, []uuid.UUID{motion.ID}, asset.VisibilityHidden, at); err != nil {
			return err
		}
	}

	if !still.LivePhotoVideoID.Valid || still.LivePhotoVideoID.UUID != motion.ID {
		if _, err = repos.Asset.Update(ctx, still.ID, repoasset.Update{LivePhotoVideoID: optional.Of(motion.ID)}, at); err != nil {
			return err
		}
		if still.LivePhotoVideoID.Valid {
			if err = gh.Jobs(rt).DeleteAsset(still.LivePhotoVideoID.UUID, true); err != nil {
				return err
			}
		}
	}

	motionAbs, err := root.Abs(motion.OriginalPath)
	if err != nil {
		return err
	}
	if _, err = os.Stat(motionAbs); errors.Is(err, os.ErrNotExist) {
		if err = root.Ensure(motion.OriginalPath); err != nil {
			return err
		}
		if err = os.WriteFile(motionAbs, videoBytes, 0o640); err != nil {
			return err
		}
		if err = Extract(ctx, rt, jobs.AssetPayload{AssetID: motion.ID}); err != nil {
			return err
		}
		return gh.Jobs(rt).Transcode(jobs.AssetPayload{AssetID: motion.ID})
	}

	return err
}

func applyTagList(ctx context.Context, rt *runtime.Runtime, a *asset.Asset) error {
	exif, err := Exif(ctx, rt, a.ID)
	if err != nil {
		return err
	}

	var keywords []string
	if exif != nil && exif.Tags.Valid {
		if err = json.Unmarshal([]byte(exif.Tags.V), &keywords); err != nil {
			return err
		}
	}

	return logictags.ApplyKeywords(ctx, rt, a.ID, a.OwnerID, keywords)
}
