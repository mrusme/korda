package asset

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

type accessRow struct {
	ID               uuid.UUID
	LivePhotoVideoID uuid.NullUUID
}

func matching(rows []accessRow, ids []uuid.UUID) []uuid.UUID {
	wanted := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}

	out := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	add := func(id uuid.UUID) {
		if wanted[id] && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, r := range rows {
		add(r.ID)
		if r.LivePhotoVideoID.Valid {
			add(r.LivePhotoVideoID.UUID)
		}
	}

	return out
}

func (repo *Repository) AlbumAccess(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		rows, err := database.Rows[accessRow](ctx, repo.db, `SELECT a.id, a.live_photo_video_id
			FROM albums al
			JOIN album_assets aa ON aa.album_id = al.id
			JOIN assets a ON a.id = aa.asset_id AND a.deleted_at IS NULL
			JOIN album_users au ON au.album_id = al.id
			JOIN users u ON u.id = au.user_id AND u.deleted_at IS NULL
			WHERE u.id = $1 AND al.deleted_at IS NULL
				AND (a.id IN (`+marks+`) OR a.live_photo_video_id IN (`+marks+`))`,
			args...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, matching(rows, chunk)...)
	}

	return out, nil
}

func (repo *Repository) PartnerAccess(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT a.id
			FROM partners p
			JOIN users b ON b.id = p.shared_by_id AND b.deleted_at IS NULL
			JOIN assets a ON a.owner_id = b.id AND a.deleted_at IS NULL
			WHERE p.shared_with_id = $1 AND a.visibility IN ('timeline', 'hidden') AND a.id IN (`+marks+`)`,
			args...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
