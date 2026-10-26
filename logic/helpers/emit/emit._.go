package emit

import (
	"context"

	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/event"
	"xn--gckvb8fzb.com/glides/runtime"
)

func AlbumUpdated(ctx context.Context, rt *runtime.Runtime, albumID uuid.UUID, caller uuid.NullUUID, notify bool) error {
	members, err := gh.Repositories(rt).Album.Members(ctx, []uuid.UUID{albumID})
	if err != nil {
		return err
	}

	ev := event.AlbumUpdated{AlbumID: albumID, UserIDs: []uuid.UUID{}, RecipientIDs: []uuid.UUID{}}
	for _, m := range members[albumID] {
		ev.UserIDs = append(ev.UserIDs, m.ID)
		if notify && (!caller.Valid || m.ID != caller.UUID) {
			ev.RecipientIDs = append(ev.RecipientIDs, m.ID)
		}
	}
	gh.Emit(ctx, rt, ev)

	for _, recipientID := range ev.RecipientIDs {
		if err = gh.Jobs(rt).NotifyAlbumUpdate(albumID, recipientID); err != nil {
			return err
		}
	}

	return nil
}
