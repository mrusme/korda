package view

import (
	"tty.fail/mrus/korda/logic/helpers/bulk"
)

type BulkID struct {
	ID      string      `json:"id"`
	Success bool        `json:"success"`
	Error   bulk.Reason `json:"error,omitempty"`
}

func NewBulkIDs(results []bulk.Result) []BulkID {
	out := make([]BulkID, 0, len(results))
	for _, r := range results {
		out = append(out, BulkID{ID: id(r.ID), Success: r.Success, Error: r.Error})
	}

	return out
}
