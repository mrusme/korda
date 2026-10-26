package assets

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/checksum"
	"tty.fail/mrus/korda/helpers/jsonb"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/helpers/sanitize"
	"tty.fail/mrus/korda/logic/helpers/emit"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetmetadata"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/sharedlink"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/mediaroot"
	repoalbum "tty.fail/mrus/korda/services/repositories/album"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	FieldAssetData   = "assetData"
	FieldSidecarData = "sidecarData"
)

var ErrDuplicateMetadataKey = errors.New("duplicate metadata key in one upload")

type StoredFile struct {
	Path     string
	Name     string
	Size     int64
	Checksum []byte
}

type Result struct {
	ID        uuid.UUID
	Duplicate bool
}

func AcceptFile(field string, name string) error {
	switch field {
	case FieldAssetData:
		if mimetypes.IsAsset(name) {
			return nil
		}
	case FieldSidecarData:
		if mimetypes.IsSidecar(name) {
			return nil
		}
	default:
		return fmt.Errorf("%w - %s", errs.ErrUnexpectedFileField, field)
	}

	return fmt.Errorf("%w %s", errs.ErrUnsupportedFileType, name)
}

func Store(rt *runtime.Runtime, userID uuid.UUID, field string, name string, originalName string, src io.Reader) (stored StoredFile, err error) {
	ext := filepath.Ext(name)
	if field == FieldSidecarData {
		ext = ".xmp"
	}

	root := gh.MediaRoot(rt)
	relative := root.Nested(mediaroot.Upload, userID, sanitize.Filename(uuid.NewString()+ext))
	if err = root.Ensure(relative); err != nil {
		return stored, err
	}
	abs, err := root.Abs(relative)
	if err != nil {
		return stored, err
	}

	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return stored, err
	}
	defer func() {
		if err != nil {
			os.Remove(abs)
		}
	}()

	var sum hash.Hash
	w := io.Writer(f)
	if field == FieldAssetData {
		sum = sha1.New()
		w = io.MultiWriter(f, sum)
	}

	size, err := io.Copy(w, src)
	if err != nil {
		f.Close()
		return stored, err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return stored, err
	}
	if err = f.Close(); err != nil {
		return stored, err
	}
	if size == 0 {
		return stored, errs.ErrFileEmpty
	}

	stored = StoredFile{Path: relative, Name: originalName, Size: size}
	if sum != nil {
		stored.Checksum = sum.Sum(nil)
	}

	return stored, nil
}

func Discard(rt *runtime.Runtime, files ...*StoredFile) {
	var paths []string
	for _, f := range files {
		if f != nil && f.Path != "" {
			paths = append(paths, f.Path)
		}
	}
	if err := gh.MediaRoot(rt).Remove(paths...); err != nil {
		rt.Warn("assets", "discard upload", "error", err)
	}
}

func DuplicateByChecksum(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, header string) (uuid.UUID, bool, error) {
	if header == "" {
		return uuid.Nil, false, nil
	}

	existing, err := gh.Repositories(rt).Asset.ByChecksum(ctx, a.User.ID, checksum.Decode(header))
	if errors.Is(err, errs.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}

	return existing.ID, true, nil
}

func requireQuota(a *auth.Auth, size int64) error {
	if a.User.QuotaSizeInBytes.Valid && a.User.QuotaSizeInBytes.V < a.User.QuotaUsageInBytes+size {
		return errs.ErrQuotaExceeded
	}

	return nil
}

func beforeLink(ctx context.Context, rt *runtime.Runtime, userID uuid.UUID, motionID uuid.UUID, at time.Time) error {
	motion, err := gh.Repositories(rt).Asset.Get(ctx, motionID)
	if errors.Is(err, errs.ErrNoRows) {
		return errs.ErrLivePhotoNotFound
	}
	if err != nil {
		return err
	}
	if motion.Type != asset.TypeVideo {
		return errs.ErrLivePhotoNotVideo
	}
	if motion.OwnerID != userID {
		return errs.ErrLivePhotoNotOwned
	}
	if motion.Visibility != asset.VisibilityTimeline {
		return nil
	}

	if err = gh.Repositories(rt).Asset.SetVisibility(ctx, []uuid.UUID{motionID}, asset.VisibilityHidden, at); err != nil {
		return err
	}
	gh.Emit(ctx, rt, event.AssetHidden{UserID: userID, AssetID: motionID})

	return nil
}

func metadataItems(in []MetadataItemInput, at time.Time) ([]assetmetadata.Item, error) {
	seen := make(map[string]bool, len(in))
	items := make([]assetmetadata.Item, 0, len(in))
	for _, item := range in {
		if seen[item.Key] {
			return nil, ErrDuplicateMetadataKey
		}
		seen[item.Key] = true

		value, err := jsonb.Normalize(item.Value)
		if err != nil {
			return nil, err
		}
		items = append(items, assetmetadata.Item{Key: item.Key, Value: string(value), UpdatedAt: at})
	}

	return items, nil
}

func Upload(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *UploadInput, file StoredFile, sidecar *StoredFile) (res Result, err error) {
	defer func() {
		if err != nil || res.Duplicate {
			Discard(rt, &file, sidecar)
		}
	}()

	if err = require(ctx, rt, a, permission.AssetUpload, a.User.ID); err != nil {
		return res, err
	}
	if err = requireQuota(a, file.Size); err != nil {
		return res, err
	}

	at := now()
	var livePhotoVideoID uuid.NullUUID
	if in.LivePhotoVideoID != nil {
		livePhotoVideoID = uuid.NullUUID{UUID: uuid.MustParse(*in.LivePhotoVideoID), Valid: true}
		if err = beforeLink(ctx, rt, a.User.ID, livePhotoVideoID.UUID, at); err != nil {
			return res, err
		}
	}

	items, err := metadataItems(in.Metadata, at)
	if err != nil {
		return res, err
	}

	originalFileName := file.Name
	if in.Filename != nil && *in.Filename != "" {
		originalFileName = *in.Filename
	}
	visibility := asset.VisibilityTimeline
	if in.Visibility != nil {
		visibility = *in.Visibility
	}

	created := &asset.Asset{
		ID:                uuid.New(),
		OwnerID:           a.User.ID,
		Type:              assetType(file.Path),
		OriginalPath:      file.Path,
		OriginalFileName:  originalFileName,
		Checksum:          file.Checksum,
		ChecksumAlgorithm: asset.ChecksumSHA1,
		FileCreatedAt:     in.FileCreatedAt.UTC(),
		FileModifiedAt:    in.FileModifiedAt.UTC(),
		LocalDateTime:     in.FileCreatedAt.UTC(),
		IsFavorite:        in.IsFavorite != nil && *in.IsFavorite,
		LivePhotoVideoID:  livePhotoVideoID,
		Visibility:        visibility,
		Status:            asset.StatusActive,
		CreatedAt:         at,
		UpdatedAt:         at,
		UpdateID:          uuid.Must(uuid.NewV7()),
	}
	if in.Duration != nil && *in.Duration != 0 {
		created.Duration.V, created.Duration.Valid = *in.Duration, true
	}

	up := repoasset.Upload{Asset: created, FileSizeInByte: file.Size, Metadata: items}
	if sidecar != nil {
		up.Sidecar = sidecar.Path
	}

	stored, err := gh.Repositories(rt).Asset.Create(ctx, up)
	if errors.Is(err, errs.ErrUniqueViolationOn) {
		existing, findErr := gh.Repositories(rt).Asset.ByChecksum(ctx, a.User.ID, file.Checksum)
		if findErr != nil {
			return res, findErr
		}
		if a.SharedLink != nil {
			if err = addToSharedLink(ctx, rt, a.SharedLink, existing.ID); err != nil {
				return res, err
			}
		}
		return Result{ID: existing.ID, Duplicate: true}, nil
	}
	if err != nil {
		return res, err
	}

	root := gh.MediaRoot(rt)
	for _, f := range []*StoredFile{sidecar, &file} {
		if f == nil {
			continue
		}
		abs, absErr := root.Abs(f.Path)
		if absErr == nil {
			absErr = os.Chtimes(abs, time.Now(), in.FileModifiedAt)
		}
		if absErr != nil {
			rt.Warn("assets", "upload", "chtimes", f.Path, "error", absErr)
		}
	}

	if err = gh.Jobs(rt).Metadata(jobs.AssetPayload{AssetID: stored.ID, Source: jobs.SourceUpload}); err != nil {
		rt.Error("assets", "upload", "queue metadata", stored.ID.String(), "error", err)
		err = nil
	}

	if a.SharedLink != nil {
		if err = addToSharedLink(ctx, rt, a.SharedLink, stored.ID); err != nil {
			return res, err
		}
	}

	return Result{ID: stored.ID}, nil
}

func RequireUploadAccess(a *auth.Auth) error {
	if a.SharedLink != nil && !a.SharedLink.AllowUpload {
		return errs.ErrUnauthorized
	}

	return nil
}

func addToSharedLink(ctx context.Context, rt *runtime.Runtime, link *sharedlink.SharedLink, assetID uuid.UUID) error {
	repos := gh.Repositories(rt)
	if !link.AlbumID.Valid {
		return repos.SharedLink.AttachAsset(ctx, link.ID, assetID)
	}

	if _, err := repos.Album.Get(ctx, link.AlbumID.UUID); errors.Is(err, errs.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}

	if err := repos.Album.AddAssets(ctx, []repoalbum.Pair{{AlbumID: link.AlbumID.UUID, AssetID: assetID}}, now()); err != nil {
		return err
	}

	return emit.AlbumUpdated(ctx, rt, link.AlbumID.UUID, uuid.NullUUID{}, true)
}

func assetType(path string) asset.Type {
	switch t := mimetypes.Lookup(path); {
	case strings.HasPrefix(t, "image/"):
		return asset.TypeImage
	case t == "application/mxf" || strings.HasPrefix(t, "video/"):
		return asset.TypeVideo
	}

	return asset.TypeOther
}

type BulkCheckResult struct {
	ID        string
	Accept    bool
	AssetID   uuid.UUID
	IsTrashed bool
}

func CheckBulkUpload(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *BulkUploadCheckInput) ([]BulkCheckResult, error) {
	checksums := make([][]byte, len(in.Assets))
	for i, item := range in.Assets {
		checksums[i] = checksum.Decode(item.Checksum)
	}

	found, err := gh.Repositories(rt).Asset.ByChecksums(ctx, a.User.ID, checksums)
	if err != nil {
		return nil, err
	}

	byChecksum := make(map[string]*asset.Asset, len(found))
	for i := range found {
		byChecksum[hex.EncodeToString(found[i].Checksum)] = &found[i]
	}

	results := make([]BulkCheckResult, len(in.Assets))
	for i, item := range in.Assets {
		results[i] = BulkCheckResult{ID: item.ID, Accept: true}
		if existing, ok := byChecksum[hex.EncodeToString(checksums[i])]; ok {
			results[i] = BulkCheckResult{ID: item.ID, AssetID: existing.ID, IsTrashed: existing.DeletedAt.Valid}
		}
	}

	return results, nil
}
