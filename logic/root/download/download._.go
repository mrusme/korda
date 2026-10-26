package download

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/sanitize"
	"tty.fail/mrus/korda/helpers/ziparchive"
	"tty.fail/mrus/korda/logic/helpers/access"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	defaultArchiveSize = 4 << 30
	motionFolder       = "encoded-video/"
)

var ErrEmptyArchive = errors.New("archive without assets")

type InfoInput struct {
	AssetIDs    []string `json:"assetIds" v:"optional" vi:"uuid"`
	AlbumID     *string  `json:"albumId" v:"uuid"`
	UserID      *string  `json:"userId" v:"uuid"`
	ArchiveSize *int64   `json:"archiveSize" v:"min=1"`
}

type ArchiveInput struct {
	AssetIDs    []string `json:"assetIds" v:"split" vi:"uuid"`
	Edited      *bool    `json:"edited" v:"configbool"`
	ArchiveName *string  `json:"archiveName"`
}

type Archive struct {
	Size     int64
	AssetIDs []uuid.UUID
}

type Info struct {
	TotalSize int64
	Archives  []Archive
}

func parseIDs(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		out = append(out, uuid.MustParse(id))
	}

	return out
}

func list(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *InfoInput) ([]repoasset.Download, error) {
	repos := gh.Repositories(rt)

	switch {
	case in.AssetIDs != nil:
		ids := parseIDs(in.AssetIDs)
		if err := access.Require(ctx, rt, a, permission.AssetDownload, ids...); err != nil {
			return nil, err
		}
		return repos.Asset.DownloadIDs(ctx, ids)

	case in.AlbumID != nil:
		id := uuid.MustParse(*in.AlbumID)
		if err := access.Require(ctx, rt, a, permission.AlbumDownload, id); err != nil {
			return nil, err
		}
		return repos.Asset.DownloadAlbum(ctx, id)

	case in.UserID != nil:
		id := uuid.MustParse(*in.UserID)
		if err := access.Require(ctx, rt, a, permission.TimelineDownload, id); err != nil {
			return nil, err
		}
		return repos.Asset.DownloadUser(ctx, id, a.Elevated)
	}

	return nil, errs.ErrDownloadTarget
}

func GetInfo(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *InfoInput) (*Info, error) {
	assets, err := list(ctx, rt, a, in)
	if err != nil {
		return nil, err
	}

	target := int64(defaultArchiveSize)
	if in.ArchiveSize != nil {
		target = *in.ArchiveSize
	}
	prefs, err := logicusers.Preferences(ctx, rt, a.User.ID)
	if err != nil {
		return nil, err
	}

	info := &Info{Archives: []Archive{}}
	open := Archive{AssetIDs: []uuid.UUID{}}
	addTo := func(d repoasset.Download) {
		open.AssetIDs = append(open.AssetIDs, d.ID)
		open.Size += d.Size.V
		if open.Size > target {
			info.Archives = append(info.Archives, open)
			open = Archive{AssetIDs: []uuid.UUID{}}
		}
	}

	var motion []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, d := range assets {
		if d.LivePhotoVideoID.Valid && !seen[d.LivePhotoVideoID.UUID] {
			seen[d.LivePhotoVideoID.UUID] = true
			motion = append(motion, d.LivePhotoVideoID.UUID)
		}
		addTo(d)
	}

	if len(motion) > 0 {
		parts, err := gh.Repositories(rt).Asset.DownloadIDs(ctx, motion)
		if err != nil {
			return nil, err
		}
		for _, d := range parts {
			if strings.HasPrefix(d.OriginalPath, motionFolder) && !prefs.Download.IncludeEmbeddedVideos {
				continue
			}
			addTo(d)
		}
	}

	if len(open.AssetIDs) > 0 {
		info.Archives = append(info.Archives, open)
	}
	for _, archive := range info.Archives {
		info.TotalSize += archive.Size
	}

	return info, nil
}

func ext(name string) (string, string) {
	dot := strings.LastIndex(name, ".")
	if dot <= 0 || strings.Trim(name[:dot], ".") == "" {
		return name, ""
	}

	return name[:dot], name[dot:]
}

func Entries(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *ArchiveInput) ([]ziparchive.Entry, error) {
	ids := parseIDs(in.AssetIDs)
	if err := access.Require(ctx, rt, a, permission.AssetDownload, ids...); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrEmptyArchive
	}

	found, err := gh.Repositories(rt).Asset.Originals(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]repoasset.Original, len(found))
	for _, o := range found {
		byID[o.ID] = o
	}

	root := gh.MediaRoot(rt)
	counts := map[string]int{}
	entries := make([]ziparchive.Entry, 0, len(ids))
	for _, id := range ids {
		o, ok := byID[id]
		if !ok {
			continue
		}

		name := sanitize.Filename(o.OriginalFileName)
		if name == "" {
			name = "unnamed"
		}
		count := counts[name]
		counts[name] = count + 1
		if count != 0 {
			base, extension := ext(name)
			name = base + "+" + strconv.Itoa(count) + extension
		}

		path := o.OriginalPath
		if in.Edited != nil && *in.Edited {
			f, err := gh.Repositories(rt).AssetFile.Get(ctx, id, assetfile.TypeFullSize, true)
			if err != nil && !errors.Is(err, errs.ErrNoRows) {
				return nil, err
			}
			if err == nil {
				path = f.Path
			}
		}
		abs, err := root.Abs(path)
		if err != nil {
			continue
		}
		entries = append(entries, ziparchive.Entry{Name: name, Path: abs})
	}

	return entries, nil
}
