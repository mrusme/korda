package share

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/bytesize"
	logicdownload "tty.fail/mrus/korda/logic/root/download"
	logicsharedlinks "tty.fail/mrus/korda/logic/root/sharedlinks"
	"tty.fail/mrus/korda/logic/root/systemconfig"
	logictimeline "tty.fail/mrus/korda/logic/root/timeline"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/preferences"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	titleShared   = "Shared photos"
	titlePublic   = "Public Share"
	archiveShared = "shared"
)

type page struct {
	Title       string
	Description string
	Count       string
	Stylesheet  string
	NotFound    bool
	Password    bool
	Error       string
	Meta        *meta
	Tiles       []tile
	Downloads   []download
}

type meta struct {
	Title       string
	Description string
	Image       string
}

type tile struct {
	Href      template.URL
	Thumbnail template.URL
	Duration  string
}

type download struct {
	Action   template.URL
	AssetIDs string
	Name     string
	Label    string
}

func origin(rt *runtime.Runtime, c fiber.Ctx) string {
	if cfg, err := systemconfig.Effective(rt); err == nil && cfg.Server.ExternalDomain != "" {
		return cfg.Server.ExternalDomain
	}

	return c.Scheme() + "://" + c.Host()
}

func gallery(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, query url.Values, base string) (*page, error) {
	d, err := logicsharedlinks.Mine(ctx, rt, a)
	if err != nil {
		return nil, err
	}

	link := d.Link
	p := &page{Title: titleShared, Description: link.Description.V}
	assets := d.Assets
	name := archiveShared
	ogTitle := titlePublic
	var cover uuid.NullUUID
	var target logicdownload.InfoInput

	if d.Album != nil {
		al := d.Album.Album
		if al.AlbumName != "" {
			p.Title, name = al.AlbumName, al.AlbumName
		}
		if p.Description == "" {
			p.Description = al.Description.V
		}
		if assets, err = albumAssets(ctx, rt, a, al); err != nil {
			return nil, err
		}
		id := al.ID.String()
		target.AlbumID = &id
		ogTitle, cover = al.AlbumName, al.AlbumThumbnailAssetID
	} else {
		target.AssetIDs = make([]string, 0, len(assets))
		for i := range assets {
			target.AssetIDs = append(target.AssetIDs, assets[i].ID.String())
		}
		if len(assets) > 0 {
			cover = uuid.NullUUID{UUID: assets[0].ID, Valid: true}
		}
	}

	encoded := strings.ReplaceAll(query.Encode(), "+", "%20")
	assetsURL := "/" + route.For("APIAssets").AsURL() + "/"
	photos, videos := 0, 0
	p.Tiles = make([]tile, 0, len(assets))
	for i := range assets {
		x := &assets[i]
		id := x.ID.String()
		t := tile{
			Href:      template.URL(assetsURL + id + "/thumbnail?size=preview&" + encoded),
			Thumbnail: template.URL(assetsURL + id + "/thumbnail?" + encoded),
		}
		switch x.Type {
		case asset.TypeVideo:
			videos++
			t.Href = template.URL(assetsURL + id + "/video/playback?" + encoded)
			t.Duration = clock(x.Duration.V)
		case asset.TypeImage:
			photos++
		}
		p.Tiles = append(p.Tiles, t)
	}
	p.Count = count(photos, videos)

	if link.AllowDownload && len(assets) > 0 {
		if p.Downloads, err = downloads(ctx, rt, a, &target, name, encoded); err != nil {
			return nil, err
		}
	}

	if !link.HasPassword() {
		p.Meta = &meta{Title: ogTitle, Description: link.Description.V}
		if p.Meta.Description == "" {
			p.Meta.Description = fmt.Sprintf("%d shared photos & videos", len(assets))
		}
		if cover.Valid {
			p.Meta.Image = base + assetsURL + cover.UUID.String() + "/thumbnail?key=" + base64.RawURLEncoding.EncodeToString(link.Key)
		}
	}

	return p, nil
}

func albumAssets(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, al *album.Album) ([]asset.Asset, error) {
	id := al.ID.String()
	order := preferences.AssetOrder(al.Order)
	in := logictimeline.BucketsInput{AlbumID: &id, Order: &order}

	buckets, err := logictimeline.Buckets(ctx, rt, a, &in)
	if err != nil {
		return nil, err
	}

	out := []asset.Asset{}
	for _, b := range buckets {
		bucket, err := logictimeline.Get(ctx, rt, a, &logictimeline.BucketInput{BucketsInput: in, TimeBucket: b.TimeBucket})
		if err != nil {
			return nil, err
		}
		for i := range bucket.Assets {
			out = append(out, bucket.Assets[i].Asset)
		}
	}

	return out, nil
}

func downloads(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, target *logicdownload.InfoInput, name string, query string) ([]download, error) {
	info, err := logicdownload.GetInfo(ctx, rt, a, target)
	if err != nil {
		return nil, err
	}

	action := template.URL("/" + route.For("APIDownload").AsURL() + "/archive?" + query)
	out := make([]download, 0, len(info.Archives))
	for i, archive := range info.Archives {
		ids := make([]string, 0, len(archive.AssetIDs))
		for _, id := range archive.AssetIDs {
			ids = append(ids, id.String())
		}
		d := download{Action: action, AssetIDs: strings.Join(ids, ","), Name: name, Label: "Download (" + bytesize.HumanReadable(archive.Size) + ")"}
		if len(info.Archives) > 1 {
			d.Name = name + "+" + strconv.Itoa(i+1)
			d.Label = fmt.Sprintf("Download part %d of %d (%s)", i+1, len(info.Archives), bytesize.HumanReadable(archive.Size))
		}
		out = append(out, d)
	}

	return out, nil
}

func clock(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	h, m, s := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}

	return fmt.Sprintf("%d:%02d", m, s)
}

func count(photos int, videos int) string {
	plural := func(n int, one string, many string) string {
		if n == 1 {
			return "1 " + one
		}
		return strconv.Itoa(n) + " " + many
	}

	switch {
	case photos == 0 && videos == 0:
		return "No photos or videos"
	case videos == 0:
		return plural(photos, "photo", "photos")
	case photos == 0:
		return plural(videos, "video", "videos")
	}

	return plural(photos, "photo", "photos") + " and " + plural(videos, "video", "videos")
}
