package view

import (
	"tty.fail/mrus/korda/logic/helpers/bulk"
	logicpeople "tty.fail/mrus/korda/logic/root/people"
)

type People struct {
	People      []struct{} `json:"people"`
	HasNextPage bool       `json:"hasNextPage"`
	Total       int        `json:"total"`
	Hidden      int        `json:"hidden"`
}

type PersonResult struct {
	ID      string      `json:"id"`
	Success bool        `json:"success"`
	Error   bulk.Reason `json:"error,omitempty"`
}

func NewPeople() People {
	return People{People: []struct{}{}}
}

func NewPersonResults(results []logicpeople.Result) []PersonResult {
	out := make([]PersonResult, 0, len(results))
	for _, r := range results {
		out = append(out, PersonResult{ID: r.ID, Success: r.Success, Error: r.Error})
	}

	return out
}
