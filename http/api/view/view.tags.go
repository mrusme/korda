package view

import (
	"strings"

	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/tag"
)

type TagBulkAssets struct {
	Count int `json:"count"`
}

func NewTag(t *tag.Tag, postgres bool) Tag {
	name := t.Value
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	return Tag{
		ID:        id(t.ID),
		ParentID:  nullableID(t.ParentID),
		Name:      name,
		Value:     t.Value,
		CreatedAt: jsontime.NewStamp(t.CreatedAt, postgres),
		UpdatedAt: jsontime.NewStamp(t.UpdatedAt, postgres),
		Color:     nullString(t.Color),
	}
}

func NewTags(list []tag.Tag) []Tag {
	out := make([]Tag, 0, len(list))
	for i := range list {
		out = append(out, NewTag(&list[i], false))
	}

	return out
}
