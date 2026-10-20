package bulk

import (
	"github.com/google/uuid"
)

type Reason string

const (
	Duplicate    Reason = "duplicate"
	NoPermission Reason = "no_permission"
	NotFound     Reason = "not_found"
	Unknown      Reason = "unknown"
)

type Result struct {
	ID      uuid.UUID
	Success bool
	Error   Reason
}

func set(ids []uuid.UUID) map[uuid.UUID]bool {
	out := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}

	return out
}

func Missing(ids []uuid.UUID, present []uuid.UUID) []uuid.UUID {
	have := set(present)
	out := []uuid.UUID{}
	for _, id := range ids {
		if !have[id] {
			out = append(out, id)
		}
	}

	return out
}

func Add(ids []uuid.UUID, present []uuid.UUID, allowed []uuid.UUID) ([]Result, []uuid.UUID) {
	existing, permitted := set(present), set(allowed)
	results := make([]Result, 0, len(ids))
	added := []uuid.UUID{}
	for _, id := range ids {
		switch {
		case existing[id]:
			results = append(results, Result{ID: id, Error: Duplicate})
		case !permitted[id]:
			results = append(results, Result{ID: id, Error: NoPermission})
		default:
			existing[id] = true
			added = append(added, id)
			results = append(results, Result{ID: id, Success: true})
		}
	}

	return results, added
}

func Remove(ids []uuid.UUID, present []uuid.UUID, allowed []uuid.UUID) ([]Result, []uuid.UUID) {
	existing, permitted := set(present), set(allowed)
	results := make([]Result, 0, len(ids))
	removed := []uuid.UUID{}
	for _, id := range ids {
		switch {
		case !existing[id]:
			results = append(results, Result{ID: id, Error: NotFound})
		case !permitted[id]:
			results = append(results, Result{ID: id, Error: NoPermission})
		default:
			delete(existing, id)
			removed = append(removed, id)
			results = append(results, Result{ID: id, Success: true})
		}
	}

	return results, removed
}
