package preferences

import "tty.fail/mrus/korda/models/user"

type Preferences struct {
	Albums             Albums             `json:"albums"`
	Folders            Folders            `json:"folders"`
	Memories           Memories           `json:"memories"`
	People             People             `json:"people"`
	SharedLinks        SharedLinks        `json:"sharedLinks"`
	Ratings            Ratings            `json:"ratings"`
	Tags               Tags               `json:"tags"`
	EmailNotifications EmailNotifications `json:"emailNotifications"`
	Download           Download           `json:"download"`
	Purchase           Purchase           `json:"purchase"`
	Cast               Cast               `json:"cast"`
	RecentlyAdded      RecentlyAdded      `json:"recentlyAdded"`
	Avatar             *Avatar            `json:"avatar,omitempty"`
}

type Albums struct {
	DefaultAssetOrder AssetOrder `json:"defaultAssetOrder"`
}

type Folders struct {
	Enabled    bool `json:"enabled"`
	SidebarWeb bool `json:"sidebarWeb"`
}

type Memories struct {
	Enabled    bool  `json:"enabled"`
	Duration   int64 `json:"duration"`
	SidebarWeb bool  `json:"sidebarWeb"`
}

type People struct {
	Enabled        bool                 `json:"enabled"`
	SidebarWeb     bool                 `json:"sidebarWeb"`
	MinimumFaces   int64                `json:"minimumFaces"`
	UpdateStrategy PersonUpdateStrategy `json:"updateStrategy"`
}

type SharedLinks struct {
	Enabled    bool `json:"enabled"`
	SidebarWeb bool `json:"sidebarWeb"`
}

type Ratings struct {
	Enabled bool `json:"enabled"`
}

type Tags struct {
	Enabled    bool `json:"enabled"`
	SidebarWeb bool `json:"sidebarWeb"`
}

type EmailNotifications struct {
	Enabled     bool `json:"enabled"`
	AlbumInvite bool `json:"albumInvite"`
	AlbumUpdate bool `json:"albumUpdate"`
}

type Download struct {
	ArchiveSize           int64 `json:"archiveSize"`
	IncludeEmbeddedVideos bool  `json:"includeEmbeddedVideos"`
}

type Purchase struct {
	ShowSupportBadge   bool   `json:"showSupportBadge"`
	HideBuyButtonUntil string `json:"hideBuyButtonUntil"`
}

type Cast struct {
	GCastEnabled bool `json:"gCastEnabled"`
}

type RecentlyAdded struct {
	SidebarWeb bool `json:"sidebarWeb"`
}

type Avatar struct {
	Color user.AvatarColor `json:"color"`
}

const gibibyte int64 = 1024 * 1024 * 1024

func Defaults() Preferences {
	return Preferences{
		Albums:             Albums{DefaultAssetOrder: OrderDesc},
		Folders:            Folders{},
		Memories:           Memories{Enabled: true, Duration: 5},
		People:             People{Enabled: true, MinimumFaces: 3, UpdateStrategy: UpdateEveryone},
		SharedLinks:        SharedLinks{Enabled: true},
		Ratings:            Ratings{},
		Tags:               Tags{},
		EmailNotifications: EmailNotifications{Enabled: true, AlbumInvite: true, AlbumUpdate: true},
		Download:           Download{ArchiveSize: 4 * gibibyte},
		Purchase:           Purchase{ShowSupportBadge: true, HideBuyButtonUntil: "2022-02-12T00:00:00.000Z"},
		Cast:               Cast{},
		RecentlyAdded:      RecentlyAdded{},
	}
}
