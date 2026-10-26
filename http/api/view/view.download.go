package view

import (
	logicdownload "tty.fail/mrus/korda/logic/root/download"
)

type DownloadArchive struct {
	Size     int64    `json:"size"`
	AssetIDs []string `json:"assetIds"`
}

type DownloadInfo struct {
	TotalSize int64             `json:"totalSize"`
	Archives  []DownloadArchive `json:"archives"`
}

func NewDownloadInfo(info *logicdownload.Info) DownloadInfo {
	out := DownloadInfo{TotalSize: info.TotalSize, Archives: make([]DownloadArchive, 0, len(info.Archives))}
	for _, a := range info.Archives {
		ids := make([]string, 0, len(a.AssetIDs))
		for _, id := range a.AssetIDs {
			ids = append(ids, id.String())
		}
		out.Archives = append(out.Archives, DownloadArchive{Size: a.Size, AssetIDs: ids})
	}

	return out
}
