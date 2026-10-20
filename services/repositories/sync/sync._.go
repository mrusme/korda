package sync

import (
	"strconv"

	"xn--gckvb8fzb.com/glides/services/database"
)

type Repository struct {
	db *database.Database
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

type Bounds struct {
	After  *string
	Before string
	Limit  int
}

func (b Bounds) where(column string, args *[]any) string {
	*args = append(*args, b.Before)
	clause := column + ` < $` + strconv.Itoa(len(*args))
	if b.After != nil {
		*args = append(*args, *b.After)
		clause += ` AND ` + column + ` > $` + strconv.Itoa(len(*args))
	}

	return clause
}

func (b Bounds) tail(column string) string {
	tail := ` ORDER BY ` + column
	if b.Limit > 0 {
		tail += ` LIMIT ` + strconv.Itoa(b.Limit)
	}

	return tail
}

type Backfill struct {
	After  *string
	End    string
	Before string
	Limit  int
}

func (b Backfill) where(column string, args *[]any) string {
	*args = append(*args, b.Before, b.End)
	clause := column + ` < $` + strconv.Itoa(len(*args)-1) + ` AND ` + column + ` <= $` + strconv.Itoa(len(*args))
	if b.After != nil {
		*args = append(*args, *b.After)
		clause += ` AND ` + column + ` > $` + strconv.Itoa(len(*args))
	}

	return clause
}

func (b Backfill) tail(column string) string {
	return Bounds{Limit: b.Limit}.tail(column)
}
