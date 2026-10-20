package tag

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/tag"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, user_id, value, color, parent_id, created_at, updated_at, update_id`

const prefixed = `t.id, t.user_id, t.value, t.color, t.parent_id, t.created_at, t.updated_at, t.update_id`

const descendants = `WITH RECURSIVE tree(id) AS (
	SELECT $1
	UNION SELECT c.id FROM tags c JOIN tree ON c.parent_id = tree.id
)`

type Repository struct {
	db *database.Database
}

type Pair struct {
	AssetID uuid.UUID
	TagID   uuid.UUID
}

type Tagged struct {
	AssetID uuid.UUID
	tag.Tag
}

func New(db *database.Database) (*Repository, error) {
	repo := new(Repository)
	repo.db = db

	return repo, nil
}

func (repo *Repository) Startup() error {
	return nil
}

func (repo *Repository) Shutdown() error {
	return nil
}

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	return database.Row[tag.Tag](ctx, repo.db, `SELECT `+columns+` FROM tags WHERE id = $1`, id)
}

func (repo *Repository) ByValue(ctx context.Context, userID uuid.UUID, value string) (*tag.Tag, error) {
	return database.Row[tag.Tag](ctx, repo.db, `SELECT `+columns+` FROM tags WHERE user_id = $1 AND value = $2`,
		userID, value)
}

func (repo *Repository) All(ctx context.Context, userID uuid.UUID) ([]tag.Tag, error) {
	return database.Rows[tag.Tag](ctx, repo.db, `SELECT `+columns+` FROM tags WHERE user_id = $1 ORDER BY value`, userID)
}

func (repo *Repository) Create(ctx context.Context, userID uuid.UUID, value string, color sql.Null[string], parentID uuid.NullUUID, now time.Time) (*tag.Tag, error) {
	return database.Row[tag.Tag](ctx, repo.db, `INSERT INTO tags (`+columns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7)
		RETURNING `+columns,
		uuid.New(), userID, value, color, parentID, now, uuid.Must(uuid.NewV7()),
	)
}

func (repo *Repository) UpsertValue(ctx context.Context, ex database.Executor, userID uuid.UUID, value string, parentID uuid.NullUUID, now time.Time) (*tag.Tag, error) {
	if ex == nil {
		ex = repo.db
	}

	return database.Row[tag.Tag](ctx, ex, `INSERT INTO tags (`+columns+`)
		VALUES ($1, $2, $3, NULL, $4, $5, $5, $6)
		ON CONFLICT (user_id, value) DO UPDATE SET
			parent_id = excluded.parent_id,
			updated_at = excluded.updated_at,
			update_id = excluded.update_id
		RETURNING `+columns,
		uuid.New(), userID, value, parentID, now, uuid.Must(uuid.NewV7()),
	)
}

func lastSegment(value string) string {
	if i := strings.LastIndex(value, "/"); i >= 0 {
		return value[i+1:]
	}

	return value
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, value string, color optional.Value[string], now time.Time) (*tag.Tag, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	previous, err := database.Value[string](ctx, tx, `SELECT value FROM tags WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}

	args := []any{value, now, uuid.Must(uuid.NewV7()), id}
	set := `value = $1, updated_at = $2, update_id = $3`
	if color.Set {
		var c any
		if !color.Null {
			c = color.V
		}
		args = append(args, c)
		set += `, color = $5`
	}
	updated, err := database.Row[tag.Tag](ctx, tx, `UPDATE tags SET `+set+` WHERE id = $4 RETURNING `+columns, args...)
	if err != nil {
		return nil, err
	}

	if value != previous {
		type node struct {
			id    uuid.UUID
			value string
		}
		frontier := []node{{id: id, value: value}}
		for len(frontier) > 0 {
			parent := frontier[0]
			frontier = frontier[1:]

			type child struct {
				ID    uuid.UUID
				Value string
			}
			children, err := database.Rows[child](ctx, tx, `SELECT id, value FROM tags WHERE parent_id = $1`, parent.id)
			if err != nil {
				return nil, err
			}
			for _, c := range children {
				next := parent.value + "/" + lastSegment(c.Value)
				if _, err = database.Exec(ctx, tx, `UPDATE tags SET value = $1, updated_at = $2, update_id = $3 WHERE id = $4`,
					next, now, uuid.Must(uuid.NewV7()), c.ID,
				); err != nil {
					return nil, err
				}
				frontier = append(frontier, node{id: c.ID, value: next})
			}
		}
	}

	return updated, tx.Commit(ctx)
}

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM tags WHERE id = $1`, id)
	return err
}

func (repo *Repository) AssetIDsDeep(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	return database.Values[uuid.UUID](ctx, repo.db, descendants+`
		SELECT DISTINCT asset_id FROM tag_assets WHERE tag_id IN (SELECT id FROM tree)`, id)
}

func (repo *Repository) AssetIDs(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{id}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT asset_id FROM tag_assets
			WHERE tag_id = $1 AND asset_id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) AddAssets(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) error {
	for _, assetID := range assetIDs {
		if _, err := database.Exec(ctx, repo.db, `INSERT INTO tag_assets (asset_id, tag_id) VALUES ($1, $2)`,
			assetID, id); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) RemoveAssets(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) error {
	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{id}
		marks := query.Placeholders(&args, chunk)
		if _, err := database.Exec(ctx, repo.db, `DELETE FROM tag_assets WHERE tag_id = $1 AND asset_id IN (`+marks+`)`,
			args...); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) UpsertAssets(ctx context.Context, pairs []Pair) ([]Pair, error) {
	out := []Pair{}
	for _, p := range pairs {
		inserted, err := database.Rows[Pair](ctx, repo.db, `INSERT INTO tag_assets (asset_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING RETURNING asset_id, tag_id`, p.AssetID, p.TagID)
		if err != nil {
			return nil, err
		}
		out = append(out, inserted...)
	}

	return out, nil
}

func (repo *Repository) ReplaceAssetTags(ctx context.Context, ex database.Executor, assetID uuid.UUID, tagIDs []uuid.UUID) error {
	if ex == nil {
		ex = repo.db
	}

	if _, err := database.Exec(ctx, ex, `DELETE FROM tag_assets WHERE asset_id = $1`, assetID); err != nil {
		return err
	}
	for _, id := range tagIDs {
		if _, err := database.Exec(ctx, ex, `INSERT INTO tag_assets (asset_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			assetID, id); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM tags
			WHERE user_id = $1 AND id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) ForAssets(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID][]tag.Tag, error) {
	out := map[uuid.UUID][]tag.Tag{}
	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{}
		marks := query.Placeholders(&args, chunk)
		rows, err := database.Rows[Tagged](ctx, repo.db, `SELECT ta.asset_id, `+prefixed+`
			FROM tag_assets ta JOIN tags t ON t.id = ta.tag_id
			WHERE ta.asset_id IN (`+marks+`)
			ORDER BY ta.rowid`, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[r.AssetID] = append(out[r.AssetID], r.Tag)
		}
	}

	return out, nil
}
