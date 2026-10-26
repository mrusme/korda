package routes

import (
	"strings"

	"xn--gckvb8fzb.com/glides/http/route"
)

const DefaultTitle string = "korda"

var Routes route.Table = route.Table{
	"Root":                    {Hierarchy: []string{"root"}},
	"CustomCSS":               {Hierarchy: []string{"root", "custom.css"}, NoBreadcrumb: true},
	"WellKnown":               {Hierarchy: []string{"root", ".well-known"}, NoBreadcrumb: true},
	"WellKnownImmich":         {Hierarchy: []string{"root", ".well-known", "immich"}, NoBreadcrumb: true},
	"API":                     {Hierarchy: []string{"root", "api"}, NoBreadcrumb: true},
	"APIActivities":           {Hierarchy: []string{"root", "api", "activities"}, NoBreadcrumb: true},
	"APIAdmin":                {Hierarchy: []string{"root", "api", "admin"}, NoBreadcrumb: true},
	"APIAdminUsers":           {Hierarchy: []string{"root", "api", "admin", "users"}, NoBreadcrumb: true},
	"APIAdminUser":            {Hierarchy: []string{"root", "api", "admin", "users", ":id"}, NoBreadcrumb: true},
	"APIAdminAuth":            {Hierarchy: []string{"root", "api", "admin", "auth"}, NoBreadcrumb: true},
	"APIAdminConfig":          {Hierarchy: []string{"root", "api", "admin", "config"}, NoBreadcrumb: true},
	"APIAdminDatabaseBackups": {Hierarchy: []string{"root", "api", "admin", "database-backups"}, NoBreadcrumb: true},
	"APIAdminIntegrity":       {Hierarchy: []string{"root", "api", "admin", "integrity"}, NoBreadcrumb: true},
	"APIAdminMaintenance":     {Hierarchy: []string{"root", "api", "admin", "maintenance"}, NoBreadcrumb: true},
	"APIAdminNotifications":   {Hierarchy: []string{"root", "api", "admin", "notifications"}, NoBreadcrumb: true},
	"APIAlbums":               {Hierarchy: []string{"root", "api", "albums"}, NoBreadcrumb: true},
	"APIAPIKeys":              {Hierarchy: []string{"root", "api", "api-keys"}, NoBreadcrumb: true},
	"APIAPIKey":               {Hierarchy: []string{"root", "api", "api-keys", ":id"}, NoBreadcrumb: true},
	"APIAssetFiles":           {Hierarchy: []string{"root", "api", "asset-files"}, NoBreadcrumb: true},
	"APIAssets":               {Hierarchy: []string{"root", "api", "assets"}, NoBreadcrumb: true},
	"APIAsset":                {Hierarchy: []string{"root", "api", "assets", ":id"}, NoBreadcrumb: true},
	"APIAuth":                 {Hierarchy: []string{"root", "api", "auth"}, NoBreadcrumb: true},
	"APIAuthLogin":            {Hierarchy: []string{"root", "api", "auth", "login"}, NoBreadcrumb: true},
	"APIClusterGroups":        {Hierarchy: []string{"root", "api", "cluster-groups"}, NoBreadcrumb: true},
	"APIConfig":               {Hierarchy: []string{"root", "api", "config"}, NoBreadcrumb: true},
	"APIDownload":             {Hierarchy: []string{"root", "api", "download"}, NoBreadcrumb: true},
	"APIDuplicates":           {Hierarchy: []string{"root", "api", "duplicates"}, NoBreadcrumb: true},
	"APIFaces":                {Hierarchy: []string{"root", "api", "faces"}, NoBreadcrumb: true},
	"APIJobs":                 {Hierarchy: []string{"root", "api", "jobs"}, NoBreadcrumb: true},
	"APILibraries":            {Hierarchy: []string{"root", "api", "libraries"}, NoBreadcrumb: true},
	"APIMap":                  {Hierarchy: []string{"root", "api", "map"}, NoBreadcrumb: true},
	"APIMemories":             {Hierarchy: []string{"root", "api", "memories"}, NoBreadcrumb: true},
	"APINotifications":        {Hierarchy: []string{"root", "api", "notifications"}, NoBreadcrumb: true},
	"APIOAuth":                {Hierarchy: []string{"root", "api", "oauth"}, NoBreadcrumb: true},
	"APIPartners":             {Hierarchy: []string{"root", "api", "partners"}, NoBreadcrumb: true},
	"APIPeople":               {Hierarchy: []string{"root", "api", "people"}, NoBreadcrumb: true},
	"APIPlugins":              {Hierarchy: []string{"root", "api", "plugins"}, NoBreadcrumb: true},
	"APIPublicConfig":         {Hierarchy: []string{"root", "api", "public/config"}, NoBreadcrumb: true},
	"APIQueues":               {Hierarchy: []string{"root", "api", "queues"}, NoBreadcrumb: true},
	"APISearch":               {Hierarchy: []string{"root", "api", "search"}, NoBreadcrumb: true},
	"APIServer":               {Hierarchy: []string{"root", "api", "server"}, NoBreadcrumb: true},
	"APIStacks":               {Hierarchy: []string{"root", "api", "stacks"}, NoBreadcrumb: true},
	"APISync":                 {Hierarchy: []string{"root", "api", "sync"}, NoBreadcrumb: true},
	"APISessions":             {Hierarchy: []string{"root", "api", "sessions"}, NoBreadcrumb: true},
	"APISharedLinks":          {Hierarchy: []string{"root", "api", "shared-links"}, NoBreadcrumb: true},
	"APISession":              {Hierarchy: []string{"root", "api", "sessions", ":id"}, NoBreadcrumb: true},
	"APISystemConfig":         {Hierarchy: []string{"root", "api", "system-config"}, NoBreadcrumb: true},
	"APISystemMetadata":       {Hierarchy: []string{"root", "api", "system-metadata"}, NoBreadcrumb: true},
	"APITags":                 {Hierarchy: []string{"root", "api", "tags"}, NoBreadcrumb: true},
	"APITimeline":             {Hierarchy: []string{"root", "api", "timeline"}, NoBreadcrumb: true},
	"APITrash":                {Hierarchy: []string{"root", "api", "trash"}, NoBreadcrumb: true},
	"APIUsers":                {Hierarchy: []string{"root", "api", "users"}, NoBreadcrumb: true},
	"APIUsersMe":              {Hierarchy: []string{"root", "api", "users", "me"}, NoBreadcrumb: true},
	"APIUser":                 {Hierarchy: []string{"root", "api", "users", ":id"}, NoBreadcrumb: true},
	"APIView":                 {Hierarchy: []string{"root", "api", "view"}, NoBreadcrumb: true},
	"APIWorkflows":            {Hierarchy: []string{"root", "api", "workflows"}, NoBreadcrumb: true},
}

func APIPath(id string) string {
	url := strings.TrimPrefix(route.For(id).AsURL(), route.For("API").AsURL())
	return "/" + strings.TrimPrefix(url, "/")
}
