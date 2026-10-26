package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"strings"
	"sync"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/logic/helpers/access"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetedit"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/assetfile"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/media"
	"tty.fail/mrus/korda/services/mediaroot"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
	repoassetedit "tty.fail/mrus/korda/services/repositories/assetedit"
	"xn--gckvb8fzb.com/glides/runtime"
)

const fullsizeQuality = 80

var editedKinds = []assetfile.Type{assetfile.TypePreview, assetfile.TypeThumbnail, assetfile.TypeFullSize}

type EditInput struct {
	Edits assetedit.Inputs `json:"edits" v:"min=1"`
}

func upright(e *assetexif.Exif) (int64, int64) {
	if e == nil || !e.ExifImageWidth.Valid || !e.ExifImageHeight.Valid || e.ExifImageWidth.V == 0 || e.ExifImageHeight.V == 0 {
		return 0, 0
	}

	switch strings.TrimSpace(e.Orientation.V) {
	case "5", "6", "7", "8", "90", "-90":
		return e.ExifImageHeight.V, e.ExifImageWidth.V
	}

	return e.ExifImageWidth.V, e.ExifImageHeight.V
}

func GetEdits(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) ([]assetedit.Edit, error) {
	if err := access.Require(ctx, rt, a, permission.AssetRead, id); err != nil {
		return nil, err
	}

	return gh.Repositories(rt).AssetEdit.All(ctx, id)
}

func Edit(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID, in *EditInput) ([]assetedit.Edit, error) {
	if err := access.Require(ctx, rt, a, permission.AssetEditCreate, id); err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)
	found, err := repos.Asset.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrAssetNotFound
	}
	if err != nil {
		return nil, err
	}
	exif, err := repos.AssetExif.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil, errs.ErrAssetNotFound
	}
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(found.OriginalPath)
	switch {
	case found.Type != asset.TypeImage:
		return nil, errs.ErrOnlyImagesEditable
	case found.LivePhotoVideoID.Valid:
		return nil, errs.ErrEditLivePhoto
	case exif.ProjectionType.V == "EQUIRECTANGULAR" || strings.HasSuffix(strings.ToLower(found.OriginalFileName), ".insp"):
		return nil, errs.ErrEditPanorama
	case strings.HasSuffix(lower, ".gif"):
		return nil, errs.ErrEditGIF
	case strings.HasSuffix(lower, ".svg"):
		return nil, errs.ErrEditSVG
	}

	width, height := upright(exif)
	if width == 0 || height == 0 {
		return nil, errs.ErrEditDimensions
	}

	items := make([]repoassetedit.Item, 0, len(in.Edits))
	for i, e := range in.Edits {
		if e.Action == assetedit.ActionCrop {
			if in.Edits[0].Action != assetedit.ActionCrop {
				return nil, errs.ErrCropFirst
			}
			var c assetedit.Crop
			if err = json.Unmarshal(e.Parameters, &c); err != nil {
				return nil, err
			}
			if c.X+c.Width > width || c.Y+c.Height > height {
				return nil, errs.ErrCropBounds
			}
		}
		items = append(items, repoassetedit.Item{Action: e.Action, Parameters: string(in.Edits[i].Parameters)})
	}

	edits, err := repos.AssetEdit.ReplaceAll(ctx, id, items, now())
	if err != nil {
		return nil, err
	}

	return edits, gh.Jobs(rt).Edit(id)
}

func RemoveEdits(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, id uuid.UUID) error {
	if err := access.Require(ctx, rt, a, permission.AssetEditDelete, id); err != nil {
		return err
	}

	repos := gh.Repositories(rt)
	if _, err := repos.Asset.Get(ctx, id); errors.Is(err, errs.ErrNoRows) {
		return errs.ErrAssetNotFound
	} else if err != nil {
		return err
	}
	if _, err := repos.AssetEdit.ReplaceAll(ctx, id, nil, now()); err != nil {
		return err
	}

	return gh.Jobs(rt).Edit(id)
}

func editedPath(root *mediaroot.MediaRoot, a *asset.Asset, kind assetfile.Type) string {
	return root.Nested(mediaroot.Thumbnails, a.OwnerID, a.ID.String()+"_"+string(kind)+"_edited.jpeg")
}

func scaled(v int64, actual int, expected int64) int {
	if expected <= 0 || int64(actual) == expected {
		return int(v)
	}

	return int(v * int64(actual) / expected)
}

func applyEdits(img image.Image, edits []assetedit.Edit, width int64, height int64) (image.Image, error) {
	for _, e := range edits {
		switch e.Action {
		case assetedit.ActionCrop:
			var c assetedit.Crop
			if err := json.Unmarshal([]byte(e.Parameters), &c); err != nil {
				return nil, err
			}
			b := img.Bounds()
			img = media.Crop(img, scaled(c.X, b.Dx(), width), scaled(c.Y, b.Dy(), height),
				max(1, scaled(c.Width, b.Dx(), width)), max(1, scaled(c.Height, b.Dy(), height)))
		case assetedit.ActionRotate:
			var r assetedit.Rotate
			if err := json.Unmarshal([]byte(e.Parameters), &r); err != nil {
				return nil, err
			}
			img = media.RotateClockwise(img, int(r.Angle))
		case assetedit.ActionMirror:
			var m assetedit.Mirror
			if err := json.Unmarshal([]byte(e.Parameters), &m); err != nil {
				return nil, err
			}
			img = media.Mirror(img, m.Axis == assetedit.AxisHorizontal)
		}
	}

	return img, nil
}

var renderLocks [64]sync.Mutex

func renderLock(id uuid.UUID) *sync.Mutex {
	return &renderLocks[int(id[15])%len(renderLocks)]
}

func RenderEdits(ctx context.Context, rt *runtime.Runtime, p jobs.AssetPayload) error {
	lock := renderLock(p.AssetID)
	lock.Lock()
	defer lock.Unlock()

	repos := gh.Repositories(rt)
	a, err := repos.Asset.Get(ctx, p.AssetID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if a.Type != asset.TypeImage {
		rt.Warn("assets", "edit", "asset", a.ID.String(), "skipped", "not an image")
		return nil
	}

	edits, err := repos.AssetEdit.All(ctx, a.ID)
	if err != nil {
		return err
	}
	exif, err := repos.AssetExif.Get(ctx, a.ID)
	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		return err
	}

	if _, err = renderEdits(ctx, rt, a, edits, exif); errors.Is(err, errs.ErrImageUndecodable) {
		rt.Warn("assets", "edit", "asset", a.ID.String(), "file", a.OriginalFileName, "skipped", err)
		return nil
	} else if err != nil {
		return err
	}

	gh.Emit(ctx, rt, event.AssetEditReady{AssetID: a.ID})

	return nil
}

func renderEdits(ctx context.Context, rt *runtime.Runtime, a *asset.Asset, edits []assetedit.Edit, exif *assetexif.Exif) ([]byte, error) {
	cfg, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}
	repos := gh.Repositories(rt)
	root := gh.MediaRoot(rt)
	med := gh.Media(rt)

	abs, err := root.Abs(a.OriginalPath)
	if err != nil {
		return nil, err
	}
	src, err := imageSource(abs)
	if err != nil {
		return nil, err
	}
	img, err := med.DecodeSource(src)
	if err != nil {
		return nil, err
	}

	width, height := upright(exif)
	at := now()
	update := repoasset.Update{}

	var thumbhash []byte
	if len(edits) == 0 {
		for _, kind := range editedKinds {
			f, err := repos.AssetFile.Get(ctx, a.ID, kind, true)
			if errors.Is(err, errs.ErrNoRows) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if err = repos.AssetFile.Delete(ctx, a.ID, kind, true); err != nil {
				return nil, err
			}
			if err = root.Remove(f.Path); err != nil {
				rt.Warn("assets", "edit", "remove", f.Path, "error", err)
			}
		}
		thumbhash = media.Thumbhash(media.Outside(img, cfg.PreviewSize))
		if width > 0 && height > 0 {
			update.Width, update.Height = optional.Of(width), optional.Of(height)
		}
	} else {
		result, err := applyEdits(img, edits, width, height)
		if err != nil {
			return nil, err
		}
		b := result.Bounds()
		sizes := map[assetfile.Type][2]int{
			assetfile.TypePreview:   {cfg.PreviewSize, cfg.PreviewQuality},
			assetfile.TypeThumbnail: {cfg.ThumbnailSize, cfg.ThumbnailQuality},
			assetfile.TypeFullSize:  {max(b.Dx(), b.Dy()), fullsizeQuality},
		}
		for _, kind := range editedKinds {
			rendition, err := med.Render(result, editedPath(root, a, kind), sizes[kind][0], sizes[kind][1])
			if err != nil {
				return nil, err
			}
			f := NewFile(a.ID, kind, rendition.Path, at)
			f.IsEdited = true
			previous, err := repos.AssetFile.Upsert(ctx, f)
			if err != nil {
				return nil, err
			}
			if previous != "" && previous != rendition.Path {
				if err = root.Remove(previous); err != nil {
					rt.Warn("assets", "edit", "remove", previous, "error", err)
				}
			}
		}
		thumbhash = media.Thumbhash(result)
		update.Width, update.Height = optional.Of(int64(b.Dx())), optional.Of(int64(b.Dy()))
	}

	if !bytes.Equal(a.Thumbhash, thumbhash) {
		update.Thumbhash = optional.Of(thumbhash)
	}
	if _, err = repos.Asset.Update(ctx, a.ID, update, at); err != nil {
		return nil, err
	}

	return thumbhash, nil
}
