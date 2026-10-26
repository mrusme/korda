package socket

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/assetexif"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/tag"
	"tty.fail/mrus/korda/services/websocket"
	"xn--gckvb8fzb.com/glides/runtime"
)

const sessionDeleteDelay = 500 * time.Millisecond

func Mount(rt *runtime.Runtime, app *fiber.App, timeout time.Duration) {
	hub := gh.Websocket(rt)
	if hub == nil {
		return
	}

	hub.SetAuthenticator(func(r *http.Request) (*websocket.Identity, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		a, err := logicauth.Authenticate(ctx, rt, request.HTTPCredentials(r), logicauth.Requirement{}, websocket.Path)
		if err != nil {
			return nil, err
		}
		id := &websocket.Identity{UserID: a.User.ID}
		if a.Session != nil {
			id.SessionID = uuid.NullUUID{UUID: a.Session.ID, Valid: true}
		}
		return id, nil
	})
	hub.SetConnected(func(id websocket.Identity) {
		hub.ToUser(id.UserID, websocket.EventServerVersion, view.NewVersion())
	})
	if bus := gh.Events(rt); bus != nil {
		bus.On(func(ctx context.Context, ev any) {
			if err := send(ctx, rt, hub, ev); err != nil {
				rt.Error("socket", "event", "error", err)
			}
		})
	}

	app.Use(websocket.Path, adaptor.HTTPHandler(hub.Handler()))
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func send(ctx context.Context, rt *runtime.Runtime, hub *websocket.Hub, ev any) error {
	switch e := ev.(type) {
	case event.AssetUploaded:
		return uploaded(ctx, rt, hub, e.AssetID)
	case event.AssetUpdated:
		return updated(ctx, rt, hub, e)
	case event.AssetHidden:
		hub.ToUser(e.UserID, websocket.EventAssetHidden, e.AssetID.String())
	case event.AssetsTrashed:
		hub.ToUser(e.UserID, websocket.EventAssetTrash, idStrings(e.AssetIDs))
	case event.AssetsRestored:
		hub.ToUser(e.UserID, websocket.EventAssetRestore, idStrings(e.AssetIDs))
	case event.AssetDeleted:
		hub.ToUser(e.UserID, websocket.EventAssetDelete, e.AssetID.String())
	case event.StackChanged:
		hub.ToUser(e.UserID, websocket.EventStackUpdate)
	case event.AlbumUpdated:
		for _, userID := range e.UserIDs {
			hub.ToUser(userID, websocket.EventAlbumUpdate, e.AlbumID.String())
		}
	case event.SessionDeleted:
		time.AfterFunc(sessionDeleteDelay, func() {
			hub.ToSession(e.SessionID, websocket.EventSessionDelete, e.SessionID.String())
		})
	case event.UserDeleted:
		hub.Broadcast(websocket.EventUserDelete, e.UserID.String())
	case event.AssetEditReady:
		a, err := gh.Repositories(rt).Asset.Get(ctx, e.AssetID)
		if errors.Is(err, errs.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		edits, err := gh.Repositories(rt).AssetEdit.All(ctx, e.AssetID)
		if err != nil {
			return err
		}
		hub.ToUser(a.OwnerID, websocket.EventEditReady, view.NewEditReady(a, edits))
	case event.NotificationCreated:
		hub.ToUser(e.Notification.UserID.UUID, websocket.EventNotification, view.NewNotification(e.Notification))
	}

	return nil
}

func load(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) (*asset.Asset, *assetexif.Exif, error) {
	repos := gh.Repositories(rt)
	a, err := repos.Asset.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	exif, err := repos.AssetExif.Get(ctx, id)
	if errors.Is(err, errs.ErrNoRows) {
		return a, nil, nil
	}

	return a, exif, err
}

func tagsOf(ctx context.Context, rt *runtime.Runtime, id uuid.UUID) ([]tag.Tag, error) {
	tags, err := gh.Repositories(rt).Tag.ForAssets(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}

	return tags[id], nil
}

func uploaded(ctx context.Context, rt *runtime.Runtime, hub *websocket.Hub, id uuid.UUID) error {
	a, exif, err := load(ctx, rt, id)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if a.Visibility != asset.VisibilityTimeline && a.Visibility != asset.VisibilityArchive {
		return nil
	}
	tags, err := tagsOf(ctx, rt, id)
	if err != nil {
		return err
	}

	hub.ToUser(a.OwnerID, websocket.EventUploadSuccess,
		view.NewAsset(a, uuid.Nil, view.AssetDetails{Exif: exif, Tags: true, TagList: tags}))
	if exif != nil {
		hub.ToUser(a.OwnerID, websocket.EventUploadReady,
			view.UploadReady{Asset: view.NewUploadReadyAsset(a), Exif: view.NewSocketExif(exif)})
	}

	return nil
}

func updated(ctx context.Context, rt *runtime.Runtime, hub *websocket.Hub, e event.AssetUpdated) error {
	a, exif, err := load(ctx, rt, e.AssetID)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	details := view.AssetDetails{}
	if e.Detail >= event.DetailExif {
		details.Exif = exif
	}
	if e.Detail == event.DetailFull {
		if details.TagList, err = tagsOf(ctx, rt, e.AssetID); err != nil {
			return err
		}
		details.Tags = true
	}
	hub.ToUser(e.UserID, websocket.EventAssetUpdate, view.NewAsset(a, e.UserID, details))

	return nil
}
