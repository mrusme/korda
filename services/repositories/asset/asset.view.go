package asset

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

const viewable = `owner_id = $1 AND visibility = 'timeline' AND deleted_at IS NULL`

func (repo *Repository) UniqueOriginalDirectories(ctx context.Context, ownerID uuid.UUID) ([]string, error) {
	paths, err := database.Values[string](ctx, repo.db, `SELECT DISTINCT original_path FROM assets WHERE `+viewable, ownerID)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var dirs []string
	for _, p := range paths {
		i := strings.LastIndex(p, "/")
		if i < 0 {
			continue
		}
		dir := p[:i+1]
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	return dirs, nil
}

func Glob(like string) string {
	var b strings.Builder
	escaped := false
	for _, r := range like {
		switch {
		case escaped:
			escaped = false
			writeGlobLiteral(&b, r)
		case r == '\\':
			escaped = true
		case r == '%':
			b.WriteByte('*')
		case r == '_':
			b.WriteByte('?')
		default:
			writeGlobLiteral(&b, r)
		}
	}

	return b.String()
}

func writeGlobLiteral(b *strings.Builder, r rune) {
	switch r {
	case '*', '?', '[':
		b.WriteByte('[')
		b.WriteRune(r)
		b.WriteByte(']')
	default:
		b.WriteRune(r)
	}
}

func (repo *Repository) ByOriginalDirectory(ctx context.Context, ownerID uuid.UUID, dir string) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+columns+` FROM assets WHERE `+viewable+`
		AND original_path GLOB $2 AND NOT original_path GLOB $3
		ORDER BY substr(original_path, length(rtrim(original_path, replace(original_path, '/', ''))) + 1)`,
		ownerID, Glob("%"+dir+"/%"), Glob("%"+dir+"/%/%"),
	)
}
