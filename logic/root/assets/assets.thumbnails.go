package assets

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/media"
	"tty.fail/mrus/korda/services/mediaroot"
	"tty.fail/mrus/korda/services/metadata/exif"
	"tty.fail/mrus/korda/services/metadata/probe"
	"tty.fail/mrus/korda/services/metadata/video"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
	"xn--gckvb8fzb.com/glides/runtime"
)

const TranscodeDisabled = "disabled"

type renditions struct {
	preview   media.Rendition
	thumbnail media.Rendition
	thumbhash []byte
}

func thumbPath(root *mediaroot.MediaRoot, a *asset.Asset, kind assetfile.Type) string {
	return root.Nested(mediaroot.Thumbnails, a.OwnerID, a.ID.String()+"_"+string(kind)+".jpeg")
}

func imageSource(abs string) (media.Source, error) {
	f, err := os.Open(abs)
	if err != nil {
		return media.Source{}, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return media.Source{}, err
	}

	info, err := probe.Probe(f, st.Size())
	if err != nil && info == nil {
		return media.Source{}, err
	}

	src := media.Source{Path: abs, Info: info, Raw: mimetypes.IsRaw(abs)}
	data := info.EXIF
	if data == nil {
		data = info.CR3Exif
	}
	if data != nil {
		if tiff, err := exif.Parse(data); err == nil {
			src.TIFF = tiff
			if v, ok := tiff.IFD0[0x0112].Uint(0); ok {
				src.Orientation = int(v)
			}
		}
	}

	return src, nil
}

func render(rt *runtime.Runtime, a *asset.Asset, img image.Image) (*renditions, error) {
	cfg, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}

	root := gh.MediaRoot(rt)
	med := gh.Media(rt)

	out := &renditions{}
	previewImage := media.Outside(img, cfg.PreviewSize)
	if out.preview, err = med.Render(previewImage, thumbPath(root, a, assetfile.TypePreview), cfg.PreviewSize, cfg.PreviewQuality); err != nil {
		return nil, err
	}
	if out.thumbnail, err = med.Render(previewImage, thumbPath(root, a, assetfile.TypeThumbnail), cfg.ThumbnailSize, cfg.ThumbnailQuality); err != nil {
		return nil, err
	}
	out.thumbhash = media.Thumbhash(previewImage)

	return out, nil
}

func imageRenditions(rt *runtime.Runtime, a *asset.Asset, abs string) (*renditions, error) {
	src, err := imageSource(abs)
	if err != nil {
		return nil, err
	}

	img, err := gh.Media(rt).DecodeSource(src)
	if err != nil {
		return nil, err
	}

	return render(rt, a, img)
}

func videoRenditions(ctx context.Context, rt *runtime.Runtime, a *asset.Asset, abs string) (*renditions, error) {
	cfg, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}
	ff := gh.Media(rt).FFmpeg()
	if ff == nil {
		return nil, media.ErrNoFFmpeg
	}

	info, err := video.Probe(ctx, cfg.FFprobe, abs)
	if err != nil {
		return nil, err
	}

	frame, err := os.CreateTemp(gh.MediaRoot(rt).Path(), ".korda-frame-*.jpeg")
	if err != nil {
		return nil, err
	}
	frame.Close()
	defer os.Remove(frame.Name())

	if err = ff.Frame(ctx, abs, info, cfg.PreviewSize, frame.Name()); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(frame.Name())
	if err != nil {
		return nil, err
	}
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return render(rt, a, img)
}

func GenerateThumbnails(ctx context.Context, rt *runtime.Runtime, p jobs.AssetPayload) error {
	repos := gh.Repositories(rt)

	a, err := repos.Asset.Get(ctx, p.AssetID)
	if errors.Is(err, errs.ErrNoRows) {
		rt.Warn("assets", "thumbnails", "asset not found", p.AssetID.String())
		return nil
	}
	if err != nil {
		return err
	}

	skipped := false
	if a.Visibility != asset.VisibilityHidden {
		if skipped, err = thumbnails(ctx, rt, a); err != nil {
			return err
		}
	}

	if p.Source != jobs.SourceUpload && !p.Notify {
		return nil
	}
	if a.Type == asset.TypeVideo {
		if err = gh.Jobs(rt).Transcode(jobs.AssetPayload{AssetID: a.ID, Source: p.Source}); err != nil {
			return err
		}
	}
	if !skipped {
		gh.Emit(ctx, rt, event.AssetUploaded{AssetID: a.ID})
	}

	return nil
}

func thumbnails(ctx context.Context, rt *runtime.Runtime, a *asset.Asset) (bool, error) {
	abs, err := gh.MediaRoot(rt).Abs(a.OriginalPath)
	if err != nil {
		return false, err
	}

	var out *renditions
	switch {
	case a.Type == asset.TypeVideo || strings.EqualFold(filepath.Ext(a.OriginalFileName), ".gif"):
		out, err = videoRenditions(ctx, rt, a, abs)
		if errors.Is(err, media.ErrNoFFmpeg) && a.Type == asset.TypeImage {
			out, err = imageRenditions(rt, a, abs)
		}
	case a.Type == asset.TypeImage:
		out, err = imageRenditions(rt, a, abs)
	default:
		return false, nil
	}
	if errors.Is(err, errs.ErrImageUndecodable) || errors.Is(err, media.ErrNoFFmpeg) || errors.Is(err, video.ErrNoProbe) {
		rt.Warn("assets", "thumbnails", "asset", a.ID.String(), "file", a.OriginalFileName, "skipped", err)
		return true, nil
	}
	if err != nil {
		return false, err
	}

	repos := gh.Repositories(rt)
	at := now()
	for kind, rendition := range map[assetfile.Type]media.Rendition{
		assetfile.TypePreview:   out.preview,
		assetfile.TypeThumbnail: out.thumbnail,
	} {
		previous, err := repos.AssetFile.Upsert(ctx, NewFile(a.ID, kind, rendition.Path, at))
		if err != nil {
			return false, err
		}
		if previous != "" && previous != rendition.Path {
			if err = gh.MediaRoot(rt).Remove(previous); err != nil {
				rt.Warn("assets", "thumbnails", "remove", previous, "error", err)
			}
		}
	}

	lock := renderLock(a.ID)
	lock.Lock()
	defer lock.Unlock()

	edits, err := repos.AssetEdit.All(ctx, a.ID)
	if err != nil {
		return false, err
	}
	if len(edits) > 0 && a.Type == asset.TypeImage {
		exif, err := repos.AssetExif.Get(ctx, a.ID)
		if err != nil && !errors.Is(err, errs.ErrNoRows) {
			return false, err
		}
		if _, err = renderEdits(ctx, rt, a, edits, exif); err != nil {
			return false, err
		}
	} else if !bytes.Equal(a.Thumbhash, out.thumbhash) {
		if _, err = repos.Asset.Update(ctx, a.ID, repoasset.Update{Thumbhash: optional.Of(out.thumbhash)}, at); err != nil {
			return false, err
		}
	}

	return false, repos.Asset.ThumbnailsGenerated(ctx, a.ID, now())
}

func targetResolution(value string, s *video.Stream) int {
	if value == "original" && s != nil {
		return min(s.Width, s.Height)
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 720
	}

	return n
}

func Transcode(ctx context.Context, rt *runtime.Runtime, p jobs.AssetPayload) error {
	repos := gh.Repositories(rt)
	root := gh.MediaRoot(rt)

	a, err := repos.Asset.Get(ctx, p.AssetID)
	if errors.Is(err, errs.ErrNoRows) || (err == nil && a.Type != asset.TypeVideo) {
		return nil
	}
	if err != nil {
		return err
	}

	cfg, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return err
	}
	ff := gh.Media(rt).FFmpeg()
	if ff == nil || cfg.Transcode == TranscodeDisabled {
		return nil
	}

	abs, err := root.Abs(a.OriginalPath)
	if err != nil {
		return err
	}
	info, err := video.Probe(ctx, cfg.FFprobe, abs)
	if errors.Is(err, video.ErrNoProbe) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Video == nil || info.Video.Width == 0 || info.Video.Height == 0 {
		rt.Warn("assets", "transcode", "asset", a.ID.String(), "skipped", "no video stream")
		return nil
	}

	relative := root.Nested(mediaroot.EncodedVideo, a.OwnerID, a.ID.String()+".mp4")
	if media.Decide(info) == media.TranscodeNone {
		existing, err := repos.AssetFile.Get(ctx, a.ID, assetfile.TypeEncodedVideo, false)
		if errors.Is(err, errs.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err = repos.AssetFile.Delete(ctx, a.ID, assetfile.TypeEncodedVideo, false); err != nil {
			return err
		}
		return root.Remove(existing.Path)
	}

	out, err := root.Abs(relative)
	if err != nil {
		return err
	}
	if err = ff.Transcode(ctx, abs, info, targetResolution(cfg.TargetResolution, info.Video), out); err != nil {
		return err
	}

	_, err = repos.AssetFile.Upsert(ctx, NewFile(a.ID, assetfile.TypeEncodedVideo, relative, now()))
	return err
}

func QueueMissingThumbnails(ctx context.Context, rt *runtime.Runtime) error {
	ids, err := gh.Repositories(rt).Asset.MissingThumbnails(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err = gh.Jobs(rt).Thumbnails(jobs.AssetPayload{AssetID: id}); err != nil {
			return err
		}
	}

	return nil
}

func QueueAll(ctx context.Context, rt *runtime.Runtime, jobType asyncjob.JobType, force bool) error {
	repos := gh.Repositories(rt)

	var ids []uuid.UUID
	var err error
	queue := gh.Jobs(rt).Thumbnails
	switch jobType {
	case jobs.ThumbnailsAll:
		ids, err = repos.Asset.ForThumbnails(ctx, force)
	case jobs.MetadataAll:
		ids, err = repos.Asset.ForMetadata(ctx, force)
		queue = gh.Jobs(rt).Metadata
	case jobs.TranscodeAll:
		ids, err = repos.Asset.ForTranscode(ctx, force)
		queue = gh.Jobs(rt).Transcode
	}
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err = queue(jobs.AssetPayload{AssetID: id, Force: force}); err != nil {
			return err
		}
	}

	return nil
}
