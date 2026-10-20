package album

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT al.id FROM albums al
			JOIN album_users au ON au.album_id = al.id AND au.role = 'owner' AND au.user_id = $1
			WHERE al.deleted_at IS NULL AND al.id IN (`+marks+`)`,
			args...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) SharedWith(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, least album.Role) ([]uuid.UUID, error) {
	roles := `'editor'`
	if least == album.RoleViewer {
		roles = `'editor', 'viewer'`
	}

	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT al.id FROM albums al
			JOIN album_users au ON au.album_id = al.id
			JOIN users u ON u.id = au.user_id AND u.deleted_at IS NULL
			WHERE al.deleted_at IS NULL AND u.id = $1 AND au.role IN (`+roles+`) AND al.id IN (`+marks+`)`,
			args...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
