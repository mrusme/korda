package view

import (
	"encoding/json"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/logic/helpers/bulk"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
)

type Duplicate struct {
	DuplicateID           string   `json:"duplicateId"`
	Assets                []Asset  `json:"assets"`
	SuggestedKeepAssetIDs []string `json:"suggestedKeepAssetIds"`
}

type ResolveResult struct {
	ID           string      `json:"id"`
	Success      bool        `json:"success"`
	Error        bulk.Reason `json:"error,omitempty"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t != ""
	}

	return true
}

func exifCount(a Asset) int {
	if a.ExifInfo == nil {
		return 0
	}
	encoded, err := json.Marshal(a.ExifInfo)
	if err != nil {
		return 0
	}
	var fields map[string]any
	if json.Unmarshal(encoded, &fields) != nil {
		return 0
	}

	n := 0
	for _, v := range fields {
		if truthy(v) {
			n++
		}
	}

	return n
}

func fileSize(a Asset) int64 {
	if a.ExifInfo == nil || a.ExifInfo.FileSizeInByte == nil {
		return 0
	}

	return *a.ExifInfo.FileSizeInByte
}

func suggestKeep(assets []Asset) []string {
	if len(assets) == 0 {
		return []string{}
	}

	var largest int64
	for _, a := range assets {
		largest = max(largest, fileSize(a))
	}
	var best *Asset
	for i := range assets {
		if fileSize(assets[i]) != largest {
			continue
		}
		if best == nil || exifCount(assets[i]) >= exifCount(*best) {
			best = &assets[i]
		}
	}

	return []string{best.ID}
}

func NewDuplicates(groups []logicassets.DuplicateGroup, viewer uuid.UUID) []Duplicate {
	out := make([]Duplicate, 0, len(groups))
	for _, g := range groups {
		assets := make([]Asset, 0, len(g.Assets))
		for i := range g.Assets {
			a := &g.Assets[i]
			assets = append(assets, NewAsset(a, viewer, AssetDetails{Exif: g.Exif[a.ID], Tags: true, TagList: g.Tags[a.ID], Aggregated: true}))
		}
		out = append(out, Duplicate{DuplicateID: id(g.DuplicateID), Assets: assets, SuggestedKeepAssetIDs: suggestKeep(assets)})
	}

	return out
}

func NewResolveResults(results []logicassets.ResolveResult) []ResolveResult {
	out := make([]ResolveResult, 0, len(results))
	for _, r := range results {
		out = append(out, ResolveResult{ID: id(r.ID), Success: r.Success, Error: r.Error, ErrorMessage: r.ErrorMessage})
	}

	return out
}
