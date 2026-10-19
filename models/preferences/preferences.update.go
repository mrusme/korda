package preferences

import "tty.fail/mrus/korda/models/user"

type Update struct {
	Albums             *AlbumsUpdate             `json:"albums,omitempty"`
	Avatar             *AvatarUpdate             `json:"avatar,omitempty"`
	Cast               *CastUpdate               `json:"cast,omitempty"`
	Download           *DownloadUpdate           `json:"download,omitempty"`
	EmailNotifications *EmailNotificationsUpdate `json:"emailNotifications,omitempty"`
	Folders            *FoldersUpdate            `json:"folders,omitempty"`
	Memories           *MemoriesUpdate           `json:"memories,omitempty"`
	People             *PeopleUpdate             `json:"people,omitempty"`
	Purchase           *PurchaseUpdate           `json:"purchase,omitempty"`
	Ratings            *RatingsUpdate            `json:"ratings,omitempty"`
	SharedLinks        *SharedLinksUpdate        `json:"sharedLinks,omitempty"`
	Tags               *TagsUpdate               `json:"tags,omitempty"`
	RecentlyAdded      *RecentlyAddedUpdate      `json:"recentlyAdded,omitempty"`
}

type AlbumsUpdate struct {
	DefaultAssetOrder *AssetOrder `json:"defaultAssetOrder,omitempty"`
}

type AvatarUpdate struct {
	Color *user.AvatarColor `json:"color,omitempty"`
}

type CastUpdate struct {
	GCastEnabled *bool `json:"gCastEnabled,omitempty"`
}

type DownloadUpdate struct {
	ArchiveSize           *int64 `json:"archiveSize,omitempty" v:"min=1"`
	IncludeEmbeddedVideos *bool  `json:"includeEmbeddedVideos,omitempty"`
}

type EmailNotificationsUpdate struct {
	Enabled     *bool `json:"enabled,omitempty"`
	AlbumInvite *bool `json:"albumInvite,omitempty"`
	AlbumUpdate *bool `json:"albumUpdate,omitempty"`
}

type FoldersUpdate struct {
	Enabled    *bool `json:"enabled,omitempty"`
	SidebarWeb *bool `json:"sidebarWeb,omitempty"`
}

type MemoriesUpdate struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	Duration   *int64 `json:"duration,omitempty" v:"min=1"`
	SidebarWeb *bool  `json:"sidebarWeb,omitempty"`
}

type PeopleUpdate struct {
	Enabled        *bool                 `json:"enabled,omitempty"`
	SidebarWeb     *bool                 `json:"sidebarWeb,omitempty"`
	MinimumFaces   *int64                `json:"minimumFaces,omitempty" v:"min=1"`
	UpdateStrategy *PersonUpdateStrategy `json:"updateStrategy,omitempty"`
}

type PurchaseUpdate struct {
	ShowSupportBadge   *bool   `json:"showSupportBadge,omitempty"`
	HideBuyButtonUntil *string `json:"hideBuyButtonUntil,omitempty"`
}

type RatingsUpdate struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type SharedLinksUpdate struct {
	Enabled    *bool `json:"enabled,omitempty"`
	SidebarWeb *bool `json:"sidebarWeb,omitempty"`
}

type TagsUpdate struct {
	Enabled    *bool `json:"enabled,omitempty"`
	SidebarWeb *bool `json:"sidebarWeb,omitempty"`
}

type RecentlyAddedUpdate struct {
	SidebarWeb *bool `json:"sidebarWeb,omitempty"`
}
