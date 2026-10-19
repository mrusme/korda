package view

import (
	"tty.fail/mrus/korda/helpers/bytesize"
	"tty.fail/mrus/korda/helpers/jsontime"
	logicserver "tty.fail/mrus/korda/logic/root/server"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/versionhistory"
	"tty.fail/mrus/korda/services/mediaroot"
)

type Ping struct {
	Res string `json:"res"`
}

type Version struct {
	Major      int  `json:"major"`
	Minor      int  `json:"minor"`
	Patch      int  `json:"patch"`
	Prerelease *int `json:"prerelease"`
}

type VersionHistory struct {
	ID        string        `json:"id"`
	CreatedAt jsontime.Time `json:"createdAt"`
	Version   string        `json:"version"`
}

type Features struct {
	SmartSearch         bool `json:"smartSearch"`
	FacialRecognition   bool `json:"facialRecognition"`
	DuplicateDetection  bool `json:"duplicateDetection"`
	Map                 bool `json:"map"`
	ReverseGeocoding    bool `json:"reverseGeocoding"`
	ImportFaces         bool `json:"importFaces"`
	Sidecar             bool `json:"sidecar"`
	Search              bool `json:"search"`
	Trash               bool `json:"trash"`
	OAuth               bool `json:"oauth"`
	OAuthAutoLaunch     bool `json:"oauthAutoLaunch"`
	OCR                 bool `json:"ocr"`
	PasswordLogin       bool `json:"passwordLogin"`
	ConfigFile          bool `json:"configFile"`
	Email               bool `json:"email"`
	RealtimeTranscoding bool `json:"realtimeTranscoding"`
}

type Config struct {
	LoginPageMessage          string `json:"loginPageMessage"`
	TrashDays                 int    `json:"trashDays"`
	UserDeleteDelay           int    `json:"userDeleteDelay"`
	OAuthButtonText           string `json:"oauthButtonText"`
	OAuthAccountManagementURL string `json:"oauthAccountManagementUrl"`
	IsInitialized             bool   `json:"isInitialized"`
	IsOnboarded               bool   `json:"isOnboarded"`
	ExternalDomain            string `json:"externalDomain"`
	PublicUsers               bool   `json:"publicUsers"`
	MapDarkStyleURL           string `json:"mapDarkStyleUrl"`
	MapLightStyleURL          string `json:"mapLightStyleUrl"`
	MaintenanceMode           bool   `json:"maintenanceMode"`
	MinFaces                  int    `json:"minFaces"`
}

type MediaTypes struct {
	Video   []string `json:"video"`
	Image   []string `json:"image"`
	Sidecar []string `json:"sidecar"`
}

type Storage struct {
	DiskAvailable       string  `json:"diskAvailable"`
	DiskSize            string  `json:"diskSize"`
	DiskUse             string  `json:"diskUse"`
	DiskAvailableRaw    int64   `json:"diskAvailableRaw"`
	DiskSizeRaw         int64   `json:"diskSizeRaw"`
	DiskUseRaw          int64   `json:"diskUseRaw"`
	DiskUsagePercentage float64 `json:"diskUsagePercentage"`
}

type About struct {
	Version       string `json:"version"`
	VersionURL    string `json:"versionUrl"`
	Licensed      bool   `json:"licensed"`
	Repository    string `json:"repository,omitempty"`
	RepositoryURL string `json:"repositoryUrl,omitempty"`
	SourceRef     string `json:"sourceRef,omitempty"`
	SourceCommit  string `json:"sourceCommit,omitempty"`
	SourceURL     string `json:"sourceUrl,omitempty"`
}

type APKLinks struct {
	Arm64v8a   string `json:"arm64v8a"`
	Armeabiv7a string `json:"armeabiv7a"`
	Universal  string `json:"universal"`
	X86_64     string `json:"x86_64"`
}

type UsageByUser struct {
	UserID           string `json:"userId"`
	UserName         string `json:"userName"`
	Photos           int64  `json:"photos"`
	Videos           int64  `json:"videos"`
	Usage            int64  `json:"usage"`
	UsagePhotos      int64  `json:"usagePhotos"`
	UsageVideos      int64  `json:"usageVideos"`
	QuotaSizeInBytes *int64 `json:"quotaSizeInBytes"`
}

type Statistics struct {
	Photos      int64         `json:"photos"`
	Videos      int64         `json:"videos"`
	Usage       int64         `json:"usage"`
	UsagePhotos int64         `json:"usagePhotos"`
	UsageVideos int64         `json:"usageVideos"`
	UsageByUser []UsageByUser `json:"usageByUser"`
}

type AssetStats struct {
	Images int64 `json:"images"`
	Videos int64 `json:"videos"`
	Total  int64 `json:"total"`
}

type HeatmapDay struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type CalendarHeatmap struct {
	From       string       `json:"from"`
	To         string       `json:"to"`
	Series     []HeatmapDay `json:"series"`
	TotalCount int64        `json:"totalCount"`
}

func NewVersion() Version {
	return Version{
		Major: logicserver.ImmichMajor,
		Minor: logicserver.ImmichMinor,
		Patch: logicserver.ImmichPatch,
	}
}

func NewVersionHistory(list []versionhistory.VersionHistory) []VersionHistory {
	out := make([]VersionHistory, 0, len(list))
	for _, v := range list {
		out = append(out, VersionHistory{ID: id(v.ID), CreatedAt: time3(v.CreatedAt), Version: v.Version})
	}

	return out
}

func NewFeatures(f logicserver.Features) Features {
	return Features(f)
}

func NewConfig(c *logicserver.Config) Config {
	return Config(*c)
}

func NewMediaTypes(m logicserver.MediaTypes) MediaTypes {
	return MediaTypes(m)
}

func NewStorage(u mediaroot.Usage) Storage {
	used := u.Total - u.Free

	return Storage{
		DiskAvailable:       bytesize.HumanReadable(u.Available),
		DiskSize:            bytesize.HumanReadable(u.Total),
		DiskUse:             bytesize.HumanReadable(used),
		DiskAvailableRaw:    u.Available,
		DiskSizeRaw:         u.Total,
		DiskUseRaw:          used,
		DiskUsagePercentage: bytesize.Percentage(used, u.Total),
	}
}

func NewAbout(a logicserver.About) About {
	return About(a)
}

func NewAPKLinks(l logicserver.APKLinks) APKLinks {
	return APKLinks(l)
}

func NewStatistics(s *logicserver.Statistics) Statistics {
	out := Statistics{
		Photos:      s.Photos,
		Videos:      s.Videos,
		Usage:       s.Usage,
		UsagePhotos: s.UsagePhotos,
		UsageVideos: s.UsageVideos,
		UsageByUser: make([]UsageByUser, 0, len(s.Users)),
	}
	for _, u := range s.Users {
		out.UsageByUser = append(out.UsageByUser, UsageByUser{
			UserID:           id(u.UserID),
			UserName:         u.UserName,
			Photos:           u.Photos,
			Videos:           u.Videos,
			Usage:            u.Usage,
			UsagePhotos:      u.UsagePhotos,
			UsageVideos:      u.UsageVideos,
			QuotaSizeInBytes: nullable(u.QuotaSizeInBytes),
		})
	}

	return out
}

func NewCalendarHeatmap(h *logicusers.Heatmap) CalendarHeatmap {
	out := CalendarHeatmap{From: h.From, To: h.To, Series: make([]HeatmapDay, 0, len(h.Series))}
	for _, day := range h.Series {
		out.Series = append(out.Series, HeatmapDay{Date: day.Date, Count: day.Count})
		out.TotalCount += day.Count
	}

	return out
}
