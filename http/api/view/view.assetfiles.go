package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/assetfile"
)

type AssetFile struct {
	ID            string         `json:"id"`
	CreatedAt     jsontime.Time  `json:"createdAt"`
	UpdatedAt     jsontime.Time  `json:"updatedAt"`
	Type          assetfile.Type `json:"type"`
	Path          string         `json:"path"`
	IsEdited      bool           `json:"isEdited"`
	IsProgressive bool           `json:"isProgressive"`
	IsTransparent bool           `json:"isTransparent"`
}

func NewAssetFile(f *assetfile.File) AssetFile {
	return AssetFile{
		ID: id(f.ID), CreatedAt: time3(f.CreatedAt), UpdatedAt: time3(f.UpdatedAt), Type: f.Type, Path: f.Path,
		IsEdited: f.IsEdited, IsProgressive: f.IsProgressive, IsTransparent: f.IsTransparent,
	}
}

func NewAssetFiles(list []assetfile.File) []AssetFile {
	out := make([]AssetFile, 0, len(list))
	for i := range list {
		out = append(out, NewAssetFile(&list[i]))
	}

	return out
}
